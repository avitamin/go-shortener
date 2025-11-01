package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avitamin/go-shortener/internal/middleware"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap/zaptest"
)

func TestZapLogger(t *testing.T) {
	logger := zaptest.NewLogger(t)
	r := chi.NewRouter()
	r.Use(middleware.ZapLogger(logger))
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}
