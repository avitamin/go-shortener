package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"github.com/avitamin/go-shortener/internal/model"
	"github.com/avitamin/go-shortener/internal/repository"
)

var ErrNoUserIDInContext = errors.New("no user ID in context")

type ShortenerService struct {
	repo    repository.Repository
	baseURL string
}

func NewShortenerService(repo repository.Repository, baseURL string) *ShortenerService {
	return &ShortenerService{repo: repo, baseURL: baseURL}
}

func (s *ShortenerService) Shorten(ctx context.Context, orig string) (string, error) {

	short, url := s.createModel(ctx, orig)

	if err := s.repo.Save(url); err != nil {
		return "", err
	}

	return s.GetAbsoluteShortURL(short), nil
}

func (s *ShortenerService) GetAbsoluteShortURL(short string) string {
	return s.baseURL + "/" + short
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

		result[i] = s.baseURL + "/" + shortForUnique[idx]
	}

	return result, nil
}

// GetUserURLs возвращает все URL, созданные пользователем с userId.
func (s *ShortenerService) GetUserURLs(ctx context.Context) ([]model.URL, error) {
	_, ok := GetUserIDFromContext(ctx)
	if !ok {
		return nil, ErrNoUserIDInContext
	}

	return s.repo.GetUserURLs(ctx)
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
