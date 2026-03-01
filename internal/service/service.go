// Package service содержит бизнес-логику сервиса сокращения URL.
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/avitamin/go-shortener/internal/audit"
	"github.com/avitamin/go-shortener/internal/config"
	"github.com/avitamin/go-shortener/internal/model"
	"github.com/avitamin/go-shortener/internal/repository"
)

// ErrNoUserIDInContext возвращается, когда в контексте отсутствует идентификатор пользователя.
var ErrNoUserIDInContext = errors.New("no user ID in context")

// ErrServiceShuttingDown возвращается, когда сервис находится в процессе остановки.
var ErrServiceShuttingDown = errors.New("service is shutting down")

const deleteUserURLsWorkersCoount = 5

// ShortenerService — основной сервис для работы с сокращенными URL.
type ShortenerService struct {
	repo repository.Repository
	// Config содержит конфигурацию сервиса.
	Config         *config.Config
	deleteUserURLs chan DeleteUserURLs
	deleteQueue    *deleteQueue
	stopWorkers    chan struct{}
	workersWG      sync.WaitGroup
	shutdownOnce   sync.Once
	// Audit — сервис аудита для логирования операций.
	Audit *audit.Service
}

// DeleteUserURLs представляет запрос на удаление URL пользователя.
type DeleteUserURLs struct {
	// ShortURLs — список коротких идентификаторов для удаления.
	ShortURLs []string
	// UserID — идентификатор пользователя.
	UserID string
}

// NewShortenerService создает новый экземпляр ShortenerService с указанным репозиторием,
// конфигурацией и сервисом аудита, а также запускает фоновые воркеры для удаления URL.
func NewShortenerService(repo repository.Repository, config *config.Config, auditService *audit.Service) *ShortenerService {
	instance := &ShortenerService{
		repo:           repo,
		Config:         config,
		deleteUserURLs: make(chan DeleteUserURLs, 10),
		deleteQueue:    newDeleteQueue(),
		stopWorkers:    make(chan struct{}),
		Audit:          auditService,
	}

	instance.initDeleteWorkers()

	return instance
}

// Shorten создает короткую ссылку для указанного исходного URL.
// Возвращает полный URL короткой ссылки (базовый адрес + короткий идентификатор).
// В случае ошибки сохранения возвращает пустую строку и ошибку.
func (s *ShortenerService) Shorten(ctx context.Context, orig string) (string, error) {

	short, url := s.createModel(ctx, orig)

	if err := s.repo.Save(ctx, url); err != nil {
		return "", err
	}

	return s.GetAbsoluteShortURL(short), nil
}

// GetAbsoluteShortURL формирует полный URL короткой ссылки из короткого идентификатора.
// Использует strings.Builder для эффективной конкатенации строк.
func (s *ShortenerService) GetAbsoluteShortURL(short string) string {
	// Используем strings.Builder для более эффективной конкатенации
	var builder strings.Builder
	builder.Grow(len(s.Config.BaseURL) + 1 + len(short))
	builder.WriteString(s.Config.BaseURL)
	builder.WriteByte('/')
	builder.WriteString(short)
	return builder.String()
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

// Resolve получает исходный URL по короткому идентификатору.
// Возвращает исходный URL и nil в случае успеха.
// Возвращает пустую строку и ошибку, если URL не найден или помечен как удаленный.
func (s *ShortenerService) Resolve(id string) (string, error) {
	url, err := s.repo.Find(id)
	if err != nil {
		return "", err
	}

	return url.Original, nil

}

// PingContext проверяет доступность хранилища данных.
// Используется для проверки работоспособности сервиса (health check).
func (s *ShortenerService) PingContext(ctx context.Context) error {

	if err := s.repo.PingContext(ctx); err != nil {
		return err
	}

	return nil
}

// GetShort проверяет, существует ли короткая ссылка для указанного исходного URL.
// Возвращает полный URL короткой ссылки и true, если найдена.
// Возвращает пустую строку и false, если не найдена.
func (s *ShortenerService) GetShort(ctx context.Context, orig string) (string, bool) {
	short, ok := s.repo.GetShort(ctx, orig)
	if ok {
		return s.GetAbsoluteShortURL(short), true
	}

	return "", false
}

// ShortenBatch создает короткие ссылки для пакета исходных URL.
// Возвращает массив полных URL коротких ссылок в том же порядке, что и входные URL.
// Автоматически определяет дубликаты и не создает их повторно.
// Возвращает ошибку, если пакет пустой или произошла ошибка при сохранении.
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

// GetUserURLs возвращает все URL, созданные пользователем.
// Извлекает идентификатор пользователя из контекста.
// Возвращает список URL с полными адресами коротких ссылок.
// Возвращает ErrNoUserIDInContext, если пользователь не аутентифицирован.
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

// DeleteUserURLs помечает URL на удаление для текущего пользователя.
// Удаление выполняется асинхронно через очередь воркеров.
// Извлекает идентификатор пользователя из контекста.
// Возвращает ErrNoUserIDInContext, если пользователь не аутентифицирован.
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

	req := DeleteUserURLs{
		ShortURLs: slices.Clone(shortURLs),
		UserID:    userID,
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.stopWorkers:
		return ErrServiceShuttingDown
	case s.deleteUserURLs <- req:
	}

	return nil
}

func (s *ShortenerService) initDeleteWorkers() {
	for range deleteUserURLsWorkersCoount {
		s.workersWG.Add(1)
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

func (s *ShortenerService) deleteWorker() {
	defer s.workersWG.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case deleteReq := <-s.deleteUserURLs:
			s.enqueueDelete(deleteReq)
			log.Printf("Queued %d URLs for deletion for user %s", len(deleteReq.ShortURLs), deleteReq.UserID)
		case <-ticker.C:
			s.flushDeleteQueueOnce()
		case <-s.stopWorkers:
			s.drainDeleteRequests()
			return
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

func (s *ShortenerService) enqueueDelete(req DeleteUserURLs) {
	s.deleteQueue.Lock()
	s.deleteQueue.Items[req.UserID] = append(s.deleteQueue.Items[req.UserID], req.ShortURLs...)
	s.deleteQueue.Unlock()
}

func (s *ShortenerService) drainDeleteRequests() {
	for {
		select {
		case req := <-s.deleteUserURLs:
			s.enqueueDelete(req)
		default:
			return
		}
	}
}

func (s *ShortenerService) popDeleteBatch() map[string][]string {
	s.deleteQueue.Lock()
	defer s.deleteQueue.Unlock()

	batch := s.deleteQueue.Items
	s.deleteQueue.Items = make(map[string][]string)

	return batch
}

func (s *ShortenerService) pendingDeleteCount() int {
	s.deleteQueue.Lock()
	defer s.deleteQueue.Unlock()

	count := 0
	for _, shortURLs := range s.deleteQueue.Items {
		count += len(shortURLs)
	}

	return count
}

func (s *ShortenerService) flushDeleteQueueOnce() {
	batch := s.popDeleteBatch()
	for userID, shortURLs := range batch {
		log.Printf("Flushing deletion of %v for user %s", shortURLs, userID)

		if err := s.flushDeleteUserURLs(userID, shortURLs); err != nil {
			log.Printf("Error deleting URLs for user %s: %v", userID, err)
			s.enqueueDelete(DeleteUserURLs{
				ShortURLs: shortURLs,
				UserID:    userID,
			})
		}
	}
}

func (s *ShortenerService) flushDeleteQueueUntilEmpty(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		s.flushDeleteQueueOnce()
		if s.pendingDeleteCount() == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// Shutdown останавливает фоновые воркеры и дожидается сохранения накопленных удалений.
func (s *ShortenerService) Shutdown(ctx context.Context) error {
	s.shutdownOnce.Do(func() {
		close(s.stopWorkers)
	})

	workersDone := make(chan struct{})
	go func() {
		s.workersWG.Wait()
		close(workersDone)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-workersDone:
	}

	if err := s.flushDeleteQueueUntilEmpty(ctx); err != nil {
		return err
	}

	if s.Audit != nil {
		s.Audit.Close()
	}

	return nil
}

// Пул для повторного использования буферов байтов
var bytePool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 6)
		return &b
	},
}

func generateID() string {
	// Получаем буфер из пула
	bp := bytePool.Get().(*[]byte)
	b := *bp
	defer bytePool.Put(bp)

	io.ReadFull(rand.Reader, b)

	return base64.URLEncoding.EncodeToString(b)
}

// GetUserIDFromContext извлекает идентификатор пользователя из контекста.
// Возвращает идентификатор и true, если найден.
// Возвращает пустую строку и false, если не найден.
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(model.ContextUserID).(string)
	return userID, ok
}
