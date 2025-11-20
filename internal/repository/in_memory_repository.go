package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/avitamin/go-shortener/internal/model"
)

type inMemoryStorage struct {
	mu   sync.Mutex
	data map[string]string
}

func NewInMemoryStorage() *inMemoryStorage {
	return &inMemoryStorage{
		data: make(map[string]string),
	}
}

func (r *inMemoryStorage) Find(id string) (model.URL, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	orig, ok := r.data[id]

	if !ok {
		return model.URL{}, ErrNotFound
	}

	return model.URL{Short: id, Original: orig}, nil
}

func (r *inMemoryStorage) Save(url model.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.saveNoLock(url)
}

func (r *inMemoryStorage) saveNoLock(url model.URL) error {
	r.data[url.Short] = url.Original

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
	return errors.New("db not configured")
}

func (r *inMemoryStorage) GetAll() []model.URL {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]model.URL, 0, len(r.data))
	for short, orig := range r.data {
		result = append(result, model.URL{
			Short:    short,
			Original: orig,
		})
	}
	return result

}
