package repository

import (
	"context"
	"errors"

	"github.com/avitamin/go-shortener/internal/model"
)

var ErrNotFound = errors.New("url не найден")

type Repository interface {
	Save(url model.URL) error
	Find(id string) (model.URL, error)
	Close() error
	PingContext(ctx context.Context) error
}
