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
	mock_repository "github.com/avitamin/go-shortener/internal/repository/mock"
	"github.com/avitamin/go-shortener/internal/service"

	"github.com/golang/mock/gomock"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
)

var (
	cfg *config.Config
)

func init() {
	var err error

	cfg, err = config.New(false)
	if err != nil {
		log.Fatal(err)
	}
}

func setupRepository(t *testing.T, repoType string) (repo repository.Repository, err error) {
	t.Helper()

	switch repoType {
	case "database":

	case "file":
		tmpDir := t.TempDir()
		tmpFilePath := filepath.Join(tmpDir, "test_storage.json")
		cfg.FileStoragePath = tmpFilePath

		repo, err = repository.NewFileStorageRepository(cfg.FileStoragePath)
		if err != nil {
			t.Fatal(err)
		}
	default:
		repo = repository.NewInMemoryStorage()
	}

	return repo, nil
}

func setupService(t *testing.T, repo repository.Repository) *service.ShortenerService {
	t.Helper()

	svc := service.NewShortenerService(repo, cfg.BaseURL)

	return svc
}

func setupRouter(t *testing.T, svc *service.ShortenerService) (rtr http.Handler, err error) {
	t.Helper()

	rtr, err = handler.NewRouter(svc)
	if err != nil {
		return nil, err
	}

	return rtr, nil
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mock_repository.NewMockRepository(ctrl)
			defer repo.Close()

			repo.EXPECT().PingContext(gomock.Any()).Return(tt.pingResult).AnyTimes()
			repo.EXPECT().Close().Return(nil).AnyTimes()

			svc := setupService(t, repo)

			router, err := setupRouter(t, svc)
			if err != nil {
				t.Fatalf("failed to setup router: %v", err)
			}

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
	repo, err := setupRepository(t, "inmemory")
	if err != nil {
		t.Fatalf("failed to setup repository: %v", err)
	}
	defer repo.Close()

	svc := setupService(t, repo)

	router, err := setupRouter(t, svc)
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

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
	repo, err := setupRepository(t, "inmemory")
	if err != nil {
		t.Fatalf("failed to setup repository: %v", err)
	}
	defer repo.Close()

	svc := setupService(t, repo)

	router, err := setupRouter(t, svc)
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPOST_EmptyBody(t *testing.T) {
	repo, err := setupRepository(t, "inmemory")
	if err != nil {
		t.Fatalf("failed to setup repository: %v", err)
	}
	defer repo.Close()

	svc := setupService(t, repo)

	router, err := setupRouter(t, svc)
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGET_Success(t *testing.T) {
	repo, err := setupRepository(t, "inmemory")
	if err != nil {
		t.Fatalf("failed to setup repository: %v", err)
	}
	defer repo.Close()

	svc := setupService(t, repo)

	router, err := setupRouter(t, svc)
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	_ = repo.Save(model.URL{Short: "xyz", Original: "https://ya.ru"})

	req := httptest.NewRequest(http.MethodGet, "/xyz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	assert.Equal(t, "https://ya.ru", resp.Header.Get("Location"))
}

func TestShortenURL_Success(t *testing.T) {
	repo, err := setupRepository(t, "inmemory")
	if err != nil {
		t.Fatalf("failed to setup repository: %v", err)
	}
	defer repo.Close()

	svc := setupService(t, repo)

	router, err := setupRouter(t, svc)
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

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
	repo, err := setupRepository(t, "inmemory")
	if err != nil {
		t.Fatalf("failed to setup repository: %v", err)
	}
	defer repo.Close()

	svc := setupService(t, repo)

	router, err := setupRouter(t, svc)
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

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
	repo, err := setupRepository(t, "inmemory")
	if err != nil {
		t.Fatalf("failed to setup repository: %v", err)
	}
	defer repo.Close()

	svc := setupService(t, repo)

	router, err := setupRouter(t, svc)
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

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
	repo, err := setupRepository(t, "inmemory")
	if err != nil {
		t.Fatalf("failed to setup repository: %v", err)
	}
	defer repo.Close()

	svc := setupService(t, repo)

	router, err := setupRouter(t, svc)
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

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
	repo, err := setupRepository(t, "inmemory")
	if err != nil {
		t.Fatalf("failed to setup repository: %v", err)
	}
	defer repo.Close()

	svc := setupService(t, repo)

	router, err := setupRouter(t, svc)
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestShortenBatch_Success(t *testing.T) {
	repo := repository.NewInMemoryStorage()
	svc := setupService(t, repo)
	router, err := setupRouter(t, svc)
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	payload := []model.BatchShortRequest{
		{CorrelationID: "1", Original: "https://a.example"},
		{CorrelationID: "2", Original: "https://b.example"},
		{CorrelationID: "3", Original: "https://a.example"}, // duplicate original -> same short
	}

	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var got []model.BatchShortenResponse
	json.Unmarshal(respBody, &got)

	assert.Equal(t, len(payload), len(got))
	// Проверим, что первые и третий short совпадают
	assert.Equal(t, got[0].ShortURL, got[2].ShortURL)
	// correlation ids preserved
	assert.Equal(t, "1", got[0].CorrelationID)
	assert.Equal(t, "2", got[1].CorrelationID)
	assert.Equal(t, "3", got[2].CorrelationID)
}

func TestShortenBatch_EmptyArray(t *testing.T) {
	repo := repository.NewInMemoryStorage()
	svc := setupService(t, repo)
	router, _ := setupRouter(t, svc)

	b := []byte("[]")
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestShortenBatch_ElementInvalid(t *testing.T) {
	repo := repository.NewInMemoryStorage()
	svc := setupService(t, repo)
	router, _ := setupRouter(t, svc)

	payload := []model.BatchShortRequest{
		{CorrelationID: "1", Original: ""},
	}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
