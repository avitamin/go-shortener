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

// DataBaseRepository — реализация Repository с использованием PostgreSQL.
// Использует комбинацию базы данных и in-memory кеша для оптимизации производительности.
type DataBaseRepository struct {
	mu      sync.Mutex
	storage *inMemoryStorage
	db      *sql.DB
}

// NewDataBaseRepository создает новый репозиторий с использованием PostgreSQL для хранения данных.
// Загружает существующие URL из базы данных в память для быстрого доступа.
// Возвращает ошибку, если не удается загрузить данные из БД.
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

// Find находит URL по короткому идентификатору в in-memory кеше.
// Возвращает ErrNotFound, если URL не найден.
// Возвращает ErrIsDeleted, если URL помечен как удаленный.
func (r *DataBaseRepository) Find(short string) (model.URL, error) {
	return r.storage.Find(short)
}

// GetShort проверяет существование короткого идентификатора для исходного URL в кеше.
// Возвращает короткий идентификатор и true, если найден.
func (r *DataBaseRepository) GetShort(ctx context.Context, orig string) (short string, ok bool) {
	return r.storage.GetShort(ctx, orig)
}

// Save сохраняет URL в базу данных и обновляет in-memory кеш.
// Сначала сохраняет в кеш, затем в базу данных.
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

// SaveBatch сохраняет пакет URL в базу данных в рамках одной транзакции.
// Атомарно обновляет как базу данных, так и in-memory кеш.
// В случае ошибки откатывает всю транзакцию.
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

// Close закрывает соединение с базой данных.
// В текущей реализации не выполняет действий, так как соединение управляется извне.
func (r *DataBaseRepository) Close() error {
	return nil
}

// PingContext проверяет доступность соединения с базой данных.
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

// GetUserURLs возвращает все URL, созданные пользователем из контекста.
// Извлекает данные напрямую из базы данных.
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

// DeleteUserURLs помечает URL пользователя как удаленные в базе данных.
// Обновляет флаг is_deleted для указанных коротких идентификаторов.
// Также обновляет состояние in-memory кеша.
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
