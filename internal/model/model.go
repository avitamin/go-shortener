package model

import (
	"context"
	"errors"
)

var ErrUserNotFound error = errors.New("user not found")

type ctxKey string

const ContextUserID ctxKey = "user_id"

type User struct {
	ID string
}

func UserFromContext(ctx context.Context) (User, error) {
	userID, ok := ctx.Value(ContextUserID).(string)
	if ok {
		return User{ID: userID}, nil
	}

	return User{}, ErrUserNotFound
}

func NewContextWithUser(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ContextUserID, userID)
}

type URL struct {
	UUID        string `json:"uuid"`
	Short       string `json:"short_url"`
	Original    string `json:"original_url"`
	UserID      string
	DeletedFlag bool
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
}

type BatchShortRequest struct {
	CorrelationID string `json:"correlation_id"`
	Original      string `json:"original_url"`
}

type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type UserURLsResponse struct {
	OriginalURL string `json:"original_url"`
	ShortURL    string `json:"short_url"`
}
