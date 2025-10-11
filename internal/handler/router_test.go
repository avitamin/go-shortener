package handler_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avitamin/go-shortener/internal/config"
	"github.com/avitamin/go-shortener/internal/handler"
	"github.com/avitamin/go-shortener/internal/model"
	"github.com/avitamin/go-shortener/internal/repository"
	"github.com/avitamin/go-shortener/internal/service"

	"github.com/stretchr/testify/assert"
)

func setupRouter() http.Handler {
	cfg := config.New()
	repo := repository.NewInMemoryRepository()
	svc := service.NewShortenerService(repo, cfg.BaseURL)

	return handler.NewRouter(svc)
}

func TestPOST_Success(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Contains(t, string(body), "http://localhost:8080/")
}

func TestPOST_InvalidContentType(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	resp := w.Result()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPOST_EmptyBody(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	resp := w.Result()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGET_Success(t *testing.T) {
	repo := repository.NewInMemoryRepository()
	_ = repo.Save(model.URL{ID: "xyz", Original: "https://ya.ru"})
	svc := service.NewShortenerService(repo, "http://localhost:8080")
	router := handler.NewRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/xyz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	resp := w.Result()

	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	assert.Equal(t, "https://ya.ru", resp.Header.Get("Location"))
}

func TestGET_NotFound(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	resp := w.Result()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
