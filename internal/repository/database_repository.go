package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"

	"github.com/avitamin/go-shortener/internal/model"
	"github.com/lib/pq"
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

func (r *DataBaseRepository) Find(short string) (model.URL, error) {
	return r.storage.Find(short)
}

func (r *DataBaseRepository) GetShort(ctx context.Context, orig string) (short string, ok bool) {
	return r.storage.GetShort(ctx, orig)
}

func (r *DataBaseRepository) Save(ctx context.Context, url model.URL) error {
	err := r.storage.Save(ctx, url)
	if err != nil {
		return err
	}

	_, err = r.db.Exec("INSERT INTO urls (short, original, user_id) VALUES ($1, $2, $3)", url.Short, url.Original, url.UserID)
	if err != nil {
		return err
	}

	return nil

}

func (r *DataBaseRepository) SaveBatch(ctx context.Context, urls []model.URL) error {
	if len(urls) == 0 {
		return nil
	}

	// Блокируем in-memory storage на время операции, чтобы избежать гонок между
	// локальной памятью и записью в БД.
	r.mu.Lock()
	defer r.mu.Unlock()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Подготавливаем statement для вставки
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO urls (short, original) VALUES ($1, $2)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, u := range urls {
		if _, err := stmt.ExecContext(ctx, u.Short, u.Original); err != nil {
			tx.Rollback()
			return err
		}
		// Записываем в in-memory storage
		if err := r.storage.Save(ctx, u); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
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
	rows, err := r.db.QueryContext(ctx, "SELECT short, original, user_id, is_deleted FROM urls")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var url model.URL
		if err := rows.Scan(&url.Short, &url.Original, &url.UserID, &url.DeletedFlag); err != nil {
			return err
		}
		if err := r.storage.Save(ctx, url); err != nil {
			return err
		}
	}

	return rows.Err()
}

func (r *DataBaseRepository) GetUserURLs(ctx context.Context) ([]model.URL, error) {
	result := make([]model.URL, 0)

	r.mu.Lock()
	defer r.mu.Unlock()

	user, err := model.UserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, "SELECT short, original, user_id FROM urls WHERE user_id = $1", user.ID)
	if err != nil {
		return nil, fmt.Errorf("selection query error: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var url model.URL
		if err := rows.Scan(&url.Short, &url.Original, &url.UserID); err != nil {
			return nil, err
		}
		result = append(result, url)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *DataBaseRepository) DeleteUserURLs(ctx context.Context, userID string, shortens []string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// подготавливаем statement для пометки на удаление
	stmt, err := tx.PrepareContext(ctx, "UPDATE urls SET is_deleted = TRUE WHERE user_id = $1 AND short = ANY($2)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	if _, err := stmt.ExecContext(ctx, userID, pq.Array(shortens)); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Println("Mark URLs as deleted in DB for user:", userID, "shortens:", shortens)

	// Обновляем in-memory storage
	r.storage.DeleteUserURLs(ctx, userID, shortens)

	return nil
}
