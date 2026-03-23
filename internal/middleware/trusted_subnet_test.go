package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avitamin/go-shortener/internal/middleware"
)

func TestRequireTrustedSubnet(t *testing.T) {
	t.Run("forbidden when trusted subnet empty", func(t *testing.T) {
		h := middleware.RequireTrustedSubnet("")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
		req.Header.Set("X-Real-IP", "192.168.0.10")
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected %d, got %d", http.StatusForbidden, w.Code)
		}
	})

	t.Run("forbidden when x-real-ip missing", func(t *testing.T) {
		h := middleware.RequireTrustedSubnet("192.168.0.0/24")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected %d, got %d", http.StatusForbidden, w.Code)
		}
	})

	t.Run("forbidden when x-real-ip invalid", func(t *testing.T) {
		h := middleware.RequireTrustedSubnet("192.168.0.0/24")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
		req.Header.Set("X-Real-IP", "not-an-ip")
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected %d, got %d", http.StatusForbidden, w.Code)
		}
	})

	t.Run("forbidden when x-real-ip outside subnet", func(t *testing.T) {
		h := middleware.RequireTrustedSubnet("192.168.0.0/24")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
		req.Header.Set("X-Real-IP", "10.0.0.1")
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected %d, got %d", http.StatusForbidden, w.Code)
		}
	})

	t.Run("success when x-real-ip in subnet", func(t *testing.T) {
		h := middleware.RequireTrustedSubnet("192.168.0.0/24")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
		req.Header.Set("X-Real-IP", "192.168.0.10")
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})
}
