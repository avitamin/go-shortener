package repository

import (
	"sync"

	"github.com/avitamin/go-shortener/internal/model"
)

type inMemoryReposity struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewInMemoryRepository() Repository {
	return &inMemoryReposity{
		data: make(map[string]string),
	}
}

func (r *inMemoryReposity) Find(id string) (model.URL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orig, ok := r.data[id]

	if !ok {
		return model.URL{}, ErrNotFound
	}

	return model.URL{ID: id, Original: orig}, nil
}

func (r *inMemoryReposity) Save(url model.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[url.ID] = url.Original

	return nil

}

func (r *inMemoryReposity) Close() error {
	return nil
}
