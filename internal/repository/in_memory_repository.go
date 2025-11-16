package repository

import (
	"sync"

	"github.com/avitamin/go-shortener/internal/model"
)

type inMemoryReposity struct {
	mu   sync.Mutex
	data map[string]string
}

func NewInMemoryRepository() *inMemoryReposity {
	return &inMemoryReposity{
		data: make(map[string]string),
	}
}

func (r *inMemoryReposity) Find(id string) (model.URL, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

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
