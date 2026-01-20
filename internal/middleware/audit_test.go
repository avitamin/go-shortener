package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/avitamin/go-shortener/internal/config"
	"github.com/avitamin/go-shortener/internal/repository/mock"
	"github.com/avitamin/go-shortener/internal/service"
	"github.com/golang/mock/gomock"
	"go.uber.org/zap"
)

func TestAuditMiddleware(t *testing.T) {
	// Создаём временный файл для аудита
	tmpFile := "/tmp/audit_middleware_test.log"
	defer os.Remove(tmpFile)

	// Создаём конфигурацию с файлом аудита
	cfg := &config.Config{
		Address:         "localhost:8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "/tmp/test_storage",
		SecretKey:       "test_secret",
		AuditFile:       tmpFile,
	}

	// Создаём репозиторий и сервис
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mock.NewMockRepository(ctrl)
	svc := service.NewShortenerService(repo, cfg)

	// Создаём logger
	log, _ := zap.NewDevelopment()

	// Создаём тестовый handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Симулируем успешное создание короткой ссылки
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("http://localhost:8080/abc123"))

		// Сохраняем данные для аудита
		SetAuditData(w, "shorten", "https://example.com")
	})

	// Оборачиваем handler middleware
	wrappedHandler := AuditMiddleware(svc, log)(handler)

	// Создаём тестовый запрос
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	// Выполняем запрос
	wrappedHandler.ServeHTTP(w, req)

	// Даём время на асинхронную запись
	time.Sleep(200 * time.Millisecond)

	// Проверяем что файл аудита создан и содержит нужные данные
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read audit file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "shorten") {
		t.Errorf("Expected audit log to contain 'shorten', got: %s", content)
	}
	if !strings.Contains(content, "https://example.com") {
		t.Errorf("Expected audit log to contain 'https://example.com', got: %s", content)
	}
}

func TestAuditMiddleware_NoAuditOnError(t *testing.T) {
	// Создаём временный файл для аудита
	tmpFile := "/tmp/audit_middleware_error_test.log"
	defer os.Remove(tmpFile)

	// Создаём конфигурацию с файлом аудита
	cfg := &config.Config{
		Address:         "localhost:8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "/tmp/test_storage",
		SecretKey:       "test_secret",
		AuditFile:       tmpFile,
	}

	// Создаём репозиторий и сервис
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mock.NewMockRepository(ctrl)
	svc := service.NewShortenerService(repo, cfg)

	// Создаём logger
	log, _ := zap.NewDevelopment()

	// Создаём тестовый handler который возвращает ошибку
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))

		// Сохраняем данные для аудита
		SetAuditData(w, "shorten", "https://example.com")
	})

	// Оборачиваем handler middleware
	wrappedHandler := AuditMiddleware(svc, log)(handler)

	// Создаём тестовый запрос
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	// Выполняем запрос
	wrappedHandler.ServeHTTP(w, req)

	// Даём время на асинхронную запись
	time.Sleep(200 * time.Millisecond)

	// Проверяем что файл аудита пустой (не должен логировать ошибки)
	data, err := os.ReadFile(tmpFile)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to read audit file: %v", err)
	}

	if len(data) > 0 {
		t.Errorf("Expected empty audit log for error response, got: %s", string(data))
	}
}

func TestSetAuditData(t *testing.T) {
	// Создаём auditResponseWriter
	w := &auditResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
	}

	// Устанавливаем данные аудита
	SetAuditData(w, "follow", "https://test.com")

	// Проверяем что данные сохранены
	if w.auditAction != "follow" {
		t.Errorf("Expected auditAction to be 'follow', got: %s", w.auditAction)
	}
	if w.auditURL != "https://test.com" {
		t.Errorf("Expected auditURL to be 'https://test.com', got: %s", w.auditURL)
	}
}

func TestSetAuditData_NotAuditResponseWriter(t *testing.T) {
	// Создаём обычный ResponseWriter
	w := httptest.NewRecorder()

	// Пытаемся установить данные аудита (не должно паниковать)
	SetAuditData(w, "follow", "https://test.com")

	// Тест проходит если нет паники
}

func TestIsSuccessStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   bool
	}{
		{"200 OK", http.StatusOK, true},
		{"201 Created", http.StatusCreated, true},
		{"307 Redirect", http.StatusTemporaryRedirect, true},
		{"400 Bad Request", http.StatusBadRequest, false},
		{"404 Not Found", http.StatusNotFound, false},
		{"500 Internal Server Error", http.StatusInternalServerError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSuccessStatus(tt.status); got != tt.want {
				t.Errorf("isSuccessStatus(%d) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
