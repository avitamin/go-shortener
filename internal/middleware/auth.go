package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/avitamin/go-shortener/internal/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const cookieName = "user_id"

var ErrCookieMissingID = errors.New("cookie missing user id")
var ErrSecretKeyReq = errors.New("auth secret is empty")

// Auth возвращает middleware; secret не должен быть пустым.
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
			if !verifySignature(userID, sig, secret) {
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

func computeHMAC(msg, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

func verifySignature(msg, sig, secret string) bool {
	expected := computeHMAC(msg, secret)
	return hmac.Equal([]byte(expected), []byte(sig))
}

func setSignedCookie(w http.ResponseWriter, userID, secret string) {
	sig := computeHMAC(userID, secret)
	cookie := &http.Cookie{
		Name:  cookieName,
		Value: fmt.Sprintf("%s|%s", userID, sig),
		Path:  "/",
		// Session-only: не устанавливаем MaxAge или Expires
		HttpOnly: true,
		// secure можно включать в зависимости от окружения
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}
