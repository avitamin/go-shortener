package repository

import (
	"context"
	"database/sql"
	"errors"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/avitamin/go-shortener/internal/config"
	"github.com/avitamin/go-shortener/internal/model"
)

var ErrNotFound = errors.New("url не найден")

type Repository interface {
	Save(url model.URL) error
	Find(id string) (model.URL, error)
	Close() error
	PingContext(ctx context.Context) error
}

func New(cfg *config.Config) (Repository, error) {
	var repo Repository
	var err error

	if cfg.DatabaseDsn != "" {
		db, err := sql.Open("pgx", cfg.DatabaseDsn)
		if err != nil {
			return nil, err
		}
		defer db.Close()

		repo, err = NewDataBaseRepository(db)
		if err != nil {
			return nil, err
		}

	} else if cfg.FileStoragePath != "" {
		repo, err = NewFileStorageRepository(cfg.FileStoragePath)
		if err != nil {
			return nil, err
		}

	} else {
		repo = NewInMemoryStorage()
	}
	return repo, nil
}
