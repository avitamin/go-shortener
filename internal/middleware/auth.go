// Package middleware содержит HTTP middleware для обработки запросов.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	authn "github.com/avitamin/go-shortener/internal/auth"
	"github.com/avitamin/go-shortener/internal/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const cookieName = "user_id"

// ErrCookieMissingID возвращается, когда cookie не содержит идентификатор пользователя.
var ErrCookieMissingID = errors.New("cookie missing user id")

// ErrSecretKeyReq возвращается, когда секретный ключ для аутентификации пустой.
var ErrSecretKeyReq = errors.New("auth secret is empty")

// Auth возвращает middleware для аутентификации пользователей через подписанные cookies.
// Если cookie отсутствует или подпись невалидна, создается новый идентификатор пользователя.
// Идентификатор пользователя сохраняется в контексте запроса.
// Параметр secret используется для HMAC-подписи cookies и не должен быть пустым.
func Auth(secret string, logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(cookieName)
			if err != nil {
				if err != http.ErrNoCookie {
					// прочие ошибки получения cookie — логируем и создаём новую cookie
					logger.Warn("failed to read cookie, creating new one", zap.Error(err))
				}

				// Нет cookie — создаём новую и продолжаем
				newID := uuid.New().String()
				setSignedCookie(w, newID, secret)
				ctx := model.NewContextWithUser(r.Context(), newID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// cookie есть — ожидаем формат "uuid|hexsig"
			parts := strings.SplitN(c.Value, "|", 2)
			if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
				// cookie присутствует, но не содержит ID -> 401 per requirements
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			userID := parts[0]
			sig := parts[1]
			if !authn.VerifySignature(userID, sig, secret) {
				// подпись некорректна -> сгенерируем новый userID и установим cookie
				logger.Info("invalid cookie signature, issuing new cookie", zap.String("cookie_value", c.Value))
				newID := uuid.New().String()
				setSignedCookie(w, newID, secret)
				ctx := context.WithValue(r.Context(), model.ContextUserID, newID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// подпись валидна -> передаём userID в контекст
			ctx := context.WithValue(r.Context(), model.ContextUserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func setSignedCookie(w http.ResponseWriter, userID, secret string) {
	cookie := &http.Cookie{
		Name:  cookieName,
		Value: authn.BuildSignedUserID(userID, secret),
		Path:  "/",
		// Session-only: не устанавливаем MaxAge или Expires
		HttpOnly: true,
		// secure можно включать в зависимости от окружения
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}
