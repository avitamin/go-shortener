package repository

import (
	"context"
	"database/sql"
	"sync"

	"github.com/avitamin/go-shortener/internal/model"
)

type DataBaseRepository struct {
	mu      sync.Mutex
	storage *inMemoryStorage
	db      *sql.DB
}

func NewDataBaseRepository(db *sql.DB) (Repository, error) {
	r := &DataBaseRepository{
		db:      db,
		storage: NewInMemoryStorage(),
	}

	if err := r.queryUrls(context.Background()); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *DataBaseRepository) Find(id string) (model.URL, error) {
	return r.storage.Find(id)
}

func (r *DataBaseRepository) Save(url model.URL) error {
	err := r.storage.Save(url)
	if err != nil {
		return err
	}

	_, err = r.db.Exec("INSERT INTO urls (short, original) VALUES ($1, $2)", url.Short, url.Original)
	if err != nil {
		return err
	}

	return nil

}

func (r *DataBaseRepository) Close() error {
	return nil
}

func (r *DataBaseRepository) PingContext(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *DataBaseRepository) queryUrls(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, "SELECT short, original FROM urls")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var url model.URL
		if err := rows.Scan(&url.Short, &url.Original); err != nil {
			return err
		}
		if err := r.storage.Save(url); err != nil {
			return err
		}
	}

	return rows.Err()
}
