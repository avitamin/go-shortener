package repository

import (
	"context"
	"sync"

	"github.com/avitamin/go-shortener/internal/model"
)

type inMemoryStorage struct {
	mu             sync.Mutex
	origByShort    map[string]string
	shortByOrig    map[string]string
	userIDByShort  map[string]string
	deletedByShort map[string]bool
}

// NewInMemoryStorage создает новый in-memory репозиторий для хранения URL в памяти.
// Данные не сохраняются после перезапуска приложения.
// Рекомендуется для тестирования и разработки.
func NewInMemoryStorage() *inMemoryStorage {
	return &inMemoryStorage{
		origByShort:    make(map[string]string),
		shortByOrig:    make(map[string]string),
		userIDByShort:  make(map[string]string),
		deletedByShort: make(map[string]bool),
	}
}

// Find находит URL по короткому идентификатору.
// Возвращает ErrNotFound, если URL не найден.
// Возвращает ErrIsDeleted, если URL помечен как удаленный.
func (r *inMemoryStorage) Find(short string) (model.URL, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	orig, ok := r.origByShort[short]
	if !ok {
		return model.URL{}, ErrNotFound
	}

	if del, ok := r.deletedByShort[short]; ok && del {
		return model.URL{}, ErrIsDeleted
	}

	return model.URL{Short: short, Original: orig}, nil
}

// Save сохраняет URL в памяти с блокировкой для потокобезопасности.
func (r *inMemoryStorage) Save(ctx context.Context, url model.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.saveNoLock(url)
}

// GetShort проверяет существование короткого идентификатора для исходного URL.
// Возвращает короткий идентификатор и true, если найден.
func (r *inMemoryStorage) GetShort(ctx context.Context, orig string) (short string, ok bool) {
	select {
	case <-ctx.Done():
		return "", false
	default:
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	short, ok = r.shortByOrig[orig]
	return short, ok
}

func (r *inMemoryStorage) saveNoLock(url model.URL) error {
	r.origByShort[url.Short] = url.Original
	r.shortByOrig[url.Original] = url.Short
	r.userIDByShort[url.Short] = url.UserID
	if url.DeletedFlag {
		r.deletedByShort[url.Short] = true
	}

	return nil
}

// SaveBatch сохраняет пакет URL в памяти с блокировкой для потокобезопасности.
func (r *inMemoryStorage) SaveBatch(ctx context.Context, urls []model.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	err := r.SaveBatchNoLock(ctx, urls)
	if err != nil {
		return err
	}

	return nil
}

// SaveBatchNoLock сохраняет пакет URL в памяти без блокировки.
// ВАЖНО: должна вызываться только внутри уже заблокированной критической секции.
func (r *inMemoryStorage) SaveBatchNoLock(ctx context.Context, urls []model.URL) error {
	for _, u := range urls {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := r.saveNoLock(u); err != nil {
				return err
			}
		}

	}
	return nil
}

// Close закрывает хранилище. Для in-memory хранилища не требуется закрытие.
func (r *inMemoryStorage) Close() error {
	return nil
}

// PingContext возвращает ошибку, так как in-memory хранилище не имеет внешних зависимостей для проверки.
func (r *inMemoryStorage) PingContext(ctx context.Context) error {
	return ErrDBNotConfigured
}

// GetAll возвращает все URL из хранилища.
// Используется для отладки и тестирования.
func (r *inMemoryStorage) GetAll() []model.URL {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]model.URL, 0, len(r.origByShort))
	for short, orig := range r.origByShort {
		result = append(result, model.URL{
			Short:    short,
			Original: orig,
		})
	}
	return result

}

// GetUserURLs возвращает пустой список, так как in-memory хранилище не отслеживает привязку URL к пользователям.
func (r *inMemoryStorage) GetUserURLs(ctx context.Context) ([]model.URL, error) {
	result := make([]model.URL, 0)

	r.mu.Lock()
	defer r.mu.Unlock()

	return result, nil
}

// DeleteUserURLs помечает URL пользователя как удаленные.
// Проверяет принадлежность URL пользователю перед удалением.
func (r *inMemoryStorage) DeleteUserURLs(ctx context.Context, userID string, shortens []string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, short := range shortens {
		if shortUserID, ok := r.userIDByShort[short]; ok {
			if shortUserID != userID {
				continue
			}

			r.deletedByShort[short] = true
		}
	}

	return nil
}
