package repository

import (
	"context"
	"errors"

	"github.com/avitamin/go-shortener/internal/model"
)

var ErrNotFound = errors.New("url не найден")

type Repository interface {
	Save(url model.URL) error
	Find(short string) (model.URL, error)
	GetShort(orig string) (short string, ok bool)
	SaveBatch(ctx context.Context, urls []model.URL) error
	GetUserURLs(ctx context.Context) ([]model.URL, error)
	Close() error
	PingContext(ctx context.Context) error
}
