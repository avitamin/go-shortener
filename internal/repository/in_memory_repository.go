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

	r.data[url.Short] = url.Original

	return nil

}

func (r *inMemoryStorage) Close() error {
	return nil
}

func (r *inMemoryStorage) PingContext(ctx context.Context) error {
	return errors.New("db not configured")
}
