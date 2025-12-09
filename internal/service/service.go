package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"sync"
	"time"

	"github.com/avitamin/go-shortener/internal/config"
	"github.com/avitamin/go-shortener/internal/model"
	"github.com/avitamin/go-shortener/internal/repository"
)

var ErrNoUserIDInContext = errors.New("no user ID in context")

const deleteUserURLsWorkersCoount = 5

type ShortenerService struct {
	repo           repository.Repository
	Config         *config.Config
	deleteUserURLs chan DeleteUserURLs
}

type DeleteUserURLs struct {
	ShortURLs []string
	UserID    string
}

func NewShortenerService(repo repository.Repository, config *config.Config) *ShortenerService {
	instance := &ShortenerService{
		repo:           repo,
		Config:         config,
		deleteUserURLs: make(chan DeleteUserURLs, 10),
	}

	instance.initDeleteWorkers()

	return instance
}

func (s *ShortenerService) Shorten(ctx context.Context, orig string) (string, error) {

	short, url := s.createModel(ctx, orig)

	if err := s.repo.Save(ctx, url); err != nil {
		return "", err
	}

	return s.GetAbsoluteShortURL(short), nil
}

func (s *ShortenerService) GetAbsoluteShortURL(short string) string {
	return s.Config.BaseURL + "/" + short
}

func (s *ShortenerService) createModel(ctx context.Context, orig string) (string, model.URL) {
	short := generateID()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		userID = ""
	}

	model := model.URL{
		Short:    short,
		Original: orig,
		UserID:   userID,
	}

	return short, model
}

func (s *ShortenerService) Resolve(id string) (string, error) {
	url, err := s.repo.Find(id)
	if err != nil {
		return "", err
	}

	return url.Original, nil

}

func (s *ShortenerService) PingContext(ctx context.Context) error {

	if err := s.repo.PingContext(ctx); err != nil {
		return err
	}

	return nil
}

func (s *ShortenerService) GetShort(ctx context.Context, orig string) (string, bool) {
	short, ok := s.repo.GetShort(ctx, orig)
	if ok {
		return s.GetAbsoluteShortURL(short), true
	}

	return "", false
}

func (s *ShortenerService) ShortenBatch(ctx context.Context, originals []string) ([]string, error) {
	if len(originals) == 0 {
		return nil, errors.New("empty batch")
	}

	// Сначала вычислим уникальные оригиналы и индексы
	uniqMap := make(map[string]int) // original -> index в uniq
	uniques := make([]string, 0, len(originals))
	indices := make([]int, len(originals)) // для восстановления порядка
	for i, orig := range originals {
		if idx, ok := uniqMap[orig]; ok {
			indices[i] = idx
		} else {
			idx := len(uniques)
			uniqMap[orig] = idx
			uniques = append(uniques, orig)
			indices[i] = idx
		}
	}
	shortForUnique := make([]string, len(uniques))

	urlsToSave := make([]model.URL, 0, len(uniques))
	for i, orig := range uniques {
		short, ok := s.repo.GetShort(ctx, orig)
		if ok {
			shortForUnique[i] = short
			continue
		}

		short, model := s.createModel(ctx, orig)
		shortForUnique[i] = short

		urlsToSave = append(urlsToSave, model)
	}

	if err := s.repo.SaveBatch(ctx, urlsToSave); err != nil {
		return nil, err
	}

	// Восстанавливаем массив short'ов в исходном порядке
	result := make([]string, len(originals))
	for i, idx := range indices {

		result[i] = s.Config.BaseURL + "/" + shortForUnique[idx]
	}

	return result, nil
}

// GetUserURLs возвращает все URL, созданные пользователем с userId.
func (s *ShortenerService) GetUserURLs(ctx context.Context) ([]model.URL, error) {
	var result []model.URL

	_, ok := GetUserIDFromContext(ctx)
	if !ok {
		return nil, ErrNoUserIDInContext
	}

	urls, err := s.repo.GetUserURLs(ctx)
	if err != nil {
		return nil, err
	}

	for _, url := range urls {
		var resURL model.URL
		resURL.Original = url.Original
		resURL.Short = s.GetAbsoluteShortURL(url.Short)
		result = append(result, resURL)
	}

	return result, nil

}

// DeleteUserURLs помечает на удаление URL, созданные пользователем с userId.
func (s *ShortenerService) DeleteUserURLs(ctx context.Context, shortURLs []string) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		return ErrNoUserIDInContext
	}

	s.deleteUserURLs <- DeleteUserURLs{
		ShortURLs: shortURLs,
		UserID:    userID,
	}

	return nil
}

func (s *ShortenerService) initDeleteWorkers() {
	for range deleteUserURLsWorkersCoount {
		go s.deleteWorker()
	}
}

type deleteQueue struct {
	sync.Mutex
	Items map[string][]string
}

func newDeleteQueue() *deleteQueue {
	return &deleteQueue{
		Items: make(map[string][]string), // userID -> []shortURL
	}
}

var dQ = newDeleteQueue()

func (s *ShortenerService) deleteWorker() {

	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case deleteReq := <-s.deleteUserURLs:
			dQ.Lock()
			dQ.Items[deleteReq.UserID] = append(dQ.Items[deleteReq.UserID], deleteReq.ShortURLs...)
			dQ.Unlock()
			log.Printf("Queued %d URLs for deletion for user %s", len(deleteReq.ShortURLs), deleteReq.UserID)

			log.Printf("%v", dQ.Items)
		case <-ticker.C:
			dQ.Lock()
			batch := dQ.Items
			dQ.Items = make(map[string][]string)
			dQ.Unlock()

			for userID, shortURLs := range batch {
				log.Printf("Flushing deletion of %v for user %s", shortURLs, userID)

				err := s.flushDeleteUserURLs(userID, shortURLs)
				if err != nil {
					log.Printf("Error deleting URLs for user %s: %v", userID, err)
					continue
				}

				// После успешного удаления очищаем список
				delete(dQ.Items, userID)
			}
		}
	}
}

func (s *ShortenerService) flushDeleteUserURLs(userID string, shortURLs []string) error {
	ctx := context.Background()
	err := s.repo.DeleteUserURLs(ctx, userID, shortURLs)
	if err != nil {
		log.Printf("Error deleting URLs for user %s: %v", userID, err)
		return err
	}

	return nil
}

func generateID() string {
	b := make([]byte, 6)
	io.ReadFull(rand.Reader, b)

	return base64.URLEncoding.EncodeToString(b)
}

func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(model.ContextUserID).(string)
	return userID, ok
}
