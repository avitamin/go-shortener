package repository

import (
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

	return model.URL{ID: id, Original: orig}, nil
}

func (r *inMemoryStorage) Save(url model.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[url.ID] = url.Original

	return nil

}
