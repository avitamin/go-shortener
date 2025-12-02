package repository

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/avitamin/go-shortener/internal/model"
)

type inMemoryStorage struct {
	mu          sync.Mutex
	origByShort map[string]string
	shortByOrig map[string]string
}

func NewInMemoryStorage() *inMemoryStorage {
	return &inMemoryStorage{
		origByShort: make(map[string]string),
		shortByOrig: make(map[string]string),
	}
}

func (r *inMemoryStorage) Find(short string) (model.URL, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	orig, ok := r.origByShort[short]
	if !ok {
		return model.URL{}, ErrNotFound
	}

	return model.URL{Short: short, Original: orig}, nil
}

func (r *inMemoryStorage) Save(url model.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.saveNoLock(url)
}

func (r *inMemoryStorage) GetShort(orig string) (short string, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	short, ok = r.shortByOrig[orig]
	if ok {
		log.Println("найдена запись ", orig, "->", short)
		return short, true
	}

	return "", false
}

func (r *inMemoryStorage) saveNoLock(url model.URL) error {
	r.origByShort[url.Short] = url.Original
	r.shortByOrig[url.Original] = url.Short
	log.Println("Saved URL:", url.Short, "->", url.Original)

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
