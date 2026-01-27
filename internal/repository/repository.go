// Package repository предоставляет интерфейсы и реализации для работы с хранилищем URL.
package repository

import (
	"context"
	"errors"

	"github.com/avitamin/go-shortener/internal/model"
)

// ErrNotFound возвращается, когда URL не найден в хранилище.
var ErrNotFound = errors.New("url не найден")

// ErrIsDeleted возвращается, когда URL помечен как удалённый.
var ErrIsDeleted = errors.New("url помечен как удалённый")

// ErrDBNotConfigured возвращается, когда подключение к БД не настроено.
var ErrDBNotConfigured = errors.New("подключение к БД не настроено")

// Repository определяет интерфейс для работы с хранилищем сокращенных URL.
// Реализации могут использовать базу данных, файловое хранилище или память.
type Repository interface {
	// Save сохраняет новый URL в хранилище.
	Save(ctx context.Context, url model.URL) error

	// Find находит URL по короткому идентификатору.
	// Возвращает ErrNotFound, если URL не найден.
	// Возвращает ErrIsDeleted, если URL помечен как удалённый.
	Find(short string) (model.URL, error)

	// GetShort проверяет существование короткого идентификатора для исходного URL.
	GetShort(ctx context.Context, orig string) (short string, ok bool)

	// SaveBatch сохраняет пакет URL в хранилище за одну операцию.
	SaveBatch(ctx context.Context, urls []model.URL) error

	// GetUserURLs возвращает все URL, созданные пользователем из контекста.
	GetUserURLs(ctx context.Context) ([]model.URL, error)

	// DeleteUserURLs помечает URL пользователя как удалённые.
	DeleteUserURLs(ctx context.Context, userID string, shortens []string) error

	// Close закрывает соединение с хранилищем.
	Close() error

	// PingContext проверяет доступность хранилища.
	PingContext(ctx context.Context) error
}
