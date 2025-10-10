package repository

import (
	"errors"
	"sync"

	"github.com/avitamin/go-shortener/internal/model"
)

var ErrNotFound = errors.New("url не найден")

type Repository interface {
	Save(url model.Url) error
	Find(id string) (model.Url, error)
}

type inMemoryReposity struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewInMemoryRepository() Repository {
	return &inMemoryReposity{
		data: make(map[string]string),
	}
}

func (r *inMemoryReposity) Find(id string) (model.Url, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orig, ok := r.data[id]

	if !ok {
		return model.Url{}, ErrNotFound
	}

	return model.Url{ID: id, Original: orig}, nil
}

func (r *inMemoryReposity) Save(url model.Url) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[url.ID] = url.Original

	return nil

}
