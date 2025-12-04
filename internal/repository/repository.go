package repository

import (
	"context"
	"errors"

	"github.com/avitamin/go-shortener/internal/model"
)

var ErrNotFound = errors.New("url не найден")
var ErrIsDeleted = errors.New("url помечен как удалённый")
var ErrDBNotConfigured = errors.New("подключение к БД не настроено")

type Repository interface {
	Save(ctx context.Context, url model.URL) error
	Find(short string) (model.URL, error)
	GetShort(ctx context.Context, orig string) (short string, ok bool)
	SaveBatch(ctx context.Context, urls []model.URL) error
	GetUserURLs(ctx context.Context) ([]model.URL, error)
	DeleteUserURLs(ctx context.Context, userID string, shortens []string) error
	Close() error
	PingContext(ctx context.Context) error
}
