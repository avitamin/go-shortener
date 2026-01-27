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

func NewInMemoryStorage() *inMemoryStorage {
	return &inMemoryStorage{
		origByShort:    make(map[string]string),
		shortByOrig:    make(map[string]string),
		userIDByShort:  make(map[string]string),
		deletedByShort: make(map[string]bool),
	}
}

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

func (r *inMemoryStorage) Save(ctx context.Context, url model.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.saveNoLock(url)
}

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

func (r *inMemoryStorage) SaveBatch(ctx context.Context, urls []model.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	err := r.SaveBatchNoLock(ctx, urls)
	if err != nil {
		return err
	}

	return nil
}

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

func (r *inMemoryStorage) Close() error {
	return nil
}

func (r *inMemoryStorage) PingContext(ctx context.Context) error {
	return ErrDBNotConfigured
}

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

func (r *inMemoryStorage) GetUserURLs(ctx context.Context) ([]model.URL, error) {
	result := make([]model.URL, 0)

	r.mu.Lock()
	defer r.mu.Unlock()

	return result, nil
}

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
