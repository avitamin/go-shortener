package handler_test

import (
	"compress/gzip"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/avitamin/go-shortener/internal/config"
	"github.com/avitamin/go-shortener/internal/handler"
	"github.com/avitamin/go-shortener/internal/model"
	"github.com/avitamin/go-shortener/internal/repository"
	"github.com/avitamin/go-shortener/internal/service"
	"github.com/avitamin/go-shortener/internal/service/mock"

	"github.com/golang/mock/gomock"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
)

var (
	cfg *config.Config
)

func init() {
	config, err := config.New(false)
	if err != nil {
		log.Fatal(err)
	}

	cfg = config
}

func setupRouter(t *testing.T) (http.Handler, repository.Repository, *service.ShortenerService) {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFilePath := filepath.Join(tmpDir, "test_storage.json")
	cfg.FileStoragePath = tmpFilePath

	repo, err := repository.NewFileStorageRepository(cfg.FileStoragePath)
	if err != nil {
		t.Fatal(err)
	}

	svc := service.NewShortenerService(repo, cfg.BaseURL)

	handler, err := handler.NewRouter(svc)
	if err != nil {
		t.Fatal(err)
	}

	return handler, repo, svc
}

func TestPing(t *testing.T) {
	tests := []struct {
		name       string
		pingResult error
		wantStatus int
	}{
		{
			name:       "success",
			pingResult: nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "fails if timeout",
			pingResult: errors.New("истекло время ожидания"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, repo, svc := setupRouter(t)
			defer repo.Close()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock.NewMockPinger(ctrl)
			m.EXPECT().PingContext(gomock.Any()).Return(tt.pingResult)

			svc.AttachDB(m)

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}

}

func TestPOST_Success(t *testing.T) {
	router, repo, _ := setupRouter(t)
	defer repo.Close()

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
	router, repo, _ := setupRouter(t)
	defer repo.Close()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPOST_EmptyBody(t *testing.T) {
	router, repo, _ := setupRouter(t)
	defer repo.Close()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGET_Success(t *testing.T) {
	router, repo, _ := setupRouter(t)
	defer repo.Close()

	_ = repo.Save(model.URL{ID: "xyz", Original: "https://ya.ru"})

	req := httptest.NewRequest(http.MethodGet, "/xyz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	assert.Equal(t, "https://ya.ru", resp.Header.Get("Location"))
}

func TestShortenURL_Success(t *testing.T) {
	router, repo, _ := setupRouter(t)
	defer repo.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(`{"url":"https://practicum.yandex.ru"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	if contentType := resp.Header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var data map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}

	if _, ok := data["result"]; !ok {
		t.Error("expected result field in response")
	}
}

func TestShortenURL_BadRequest(t *testing.T) {
	router, repo, _ := setupRouter(t)
	defer repo.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(`{"bad":"request"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestGzipRequest_Success(t *testing.T) {
	router, repo, _ := setupRouter(t)
	defer repo.Close()

	var buf strings.Builder
	gzWriter := gzip.NewWriter(&buf)
	gzWriter.Write([]byte("https://example.com"))
	gzWriter.Close()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(buf.String()))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)

	assert.Contains(t, string(body), "http://localhost:8080/")
}

func TestGzipResponse_Success(t *testing.T) {
	router, repo, _ := setupRouter(t)
	defer repo.Close()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))

	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	body, _ := io.ReadAll(gzReader)

	assert.Contains(t, string(body), "http://localhost:8080/")
}

func TestGET_NotFound(t *testing.T) {
	router, repo, _ := setupRouter(t)
	defer repo.Close()

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
