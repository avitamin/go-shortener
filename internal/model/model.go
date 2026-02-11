// Package model содержит основные модели данных для сервиса сокращения URL.
package model

import (
	"context"
	"errors"
)

// ErrUserNotFound возвращается, когда пользователь не найден в контексте.
var ErrUserNotFound error = errors.New("user not found")

type ctxKey string

// ContextUserID — ключ для хранения идентификатора пользователя в контексте запроса.
const ContextUserID ctxKey = "user_id"

// User представляет пользователя в системе.
type User struct {
	// ID — уникальный идентификатор пользователя.
	ID string
}

// UserFromContext извлекает пользователя из контекста запроса.
// Возвращает User и nil, если пользователь найден, иначе пустой User и ErrUserNotFound.
func UserFromContext(ctx context.Context) (User, error) {
	userID, ok := ctx.Value(ContextUserID).(string)
	if ok {
		return User{ID: userID}, nil
	}

	return User{}, ErrUserNotFound
}

// NewContextWithUser создает новый контекст с добавленным идентификатором пользователя.
func NewContextWithUser(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ContextUserID, userID)
}

// URL представляет сокращенную ссылку в системе.
type URL struct {
	// UUID — уникальный идентификатор записи.
	UUID string `json:"uuid"`
	// Short — короткий идентификатор URL (часть после базового адреса).
	Short string `json:"short_url"`
	// Original — исходный полный URL.
	Original string `json:"original_url"`
	// UserID — идентификатор пользователя, создавшего ссылку.
	UserID string
	// DeletedFlag указывает, была ли ссылка удалена.
	DeletedFlag bool
}

// ShortenRequest представляет запрос на создание короткой ссылки.
type ShortenRequest struct {
	// URL — исходный URL для сокращения.
	URL string `json:"url"`
}

// ShortenResponse представляет ответ с созданной короткой ссылкой.
type ShortenResponse struct {
	// Result — полный URL короткой ссылки.
	Result string `json:"result"`
}

// BatchShortRequest представляет один элемент в пакетном запросе на создание коротких ссылок.
type BatchShortRequest struct {
	// CorrelationID — идентификатор для сопоставления запроса и ответа.
	CorrelationID string `json:"correlation_id"`
	// Original — исходный URL для сокращения.
	Original string `json:"original_url"`
}

// BatchShortenResponse представляет один элемент в ответе на пакетный запрос.
type BatchShortenResponse struct {
	// CorrelationID — идентификатор из запроса для сопоставления.
	CorrelationID string `json:"correlation_id"`
	// ShortURL — созданная короткая ссылка.
	ShortURL string `json:"short_url"`
}

// UserURLsResponse представляет одну ссылку пользователя в ответе.
type UserURLsResponse struct {
	// OriginalURL — исходный полный URL.
	OriginalURL string `json:"original_url"`
	// ShortURL — короткая ссылка.
	ShortURL string `json:"short_url"`
}
