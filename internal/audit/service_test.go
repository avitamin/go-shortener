package audit

import (
	"os"
	"strings"
	"testing"
)

func TestAuditService(t *testing.T) {
	// Создаём временный файл для тестирования
	tmpFile := "/tmp/audit_test.log"
	defer os.Remove(tmpFile)

	// Создаём сервис аудита
	service := NewService()

	// Добавляем FileObserver
	fileObserver := NewFileObserver(tmpFile)
	service.AddObserver(fileObserver)

	// Логируем событие создания короткой ссылки
	service.LogShorten("user123", "https://example.com")
	// Завершаем сервис и дожидаемся записи события
	service.Close()

	// Читаем файл и проверяем содержимое
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read audit file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "shorten") {
		t.Errorf("Expected audit log to contain 'shorten', got: %s", content)
	}
	if !strings.Contains(content, "user123") {
		t.Errorf("Expected audit log to contain 'user123', got: %s", content)
	}
	if !strings.Contains(content, "https://example.com") {
		t.Errorf("Expected audit log to contain 'https://example.com', got: %s", content)
	}
}

func TestAuditFollowEvent(t *testing.T) {
	// Создаём временный файл для тестирования
	tmpFile := "/tmp/audit_follow_test.log"
	defer os.Remove(tmpFile)

	// Создаём сервис аудита
	service := NewService()

	// Добавляем FileObserver
	fileObserver := NewFileObserver(tmpFile)
	service.AddObserver(fileObserver)

	// Логируем событие прохождения по ссылке
	service.LogFollow("user456", "https://example.org")
	// Завершаем сервис и дожидаемся записи события
	service.Close()

	// Читаем файл и проверяем содержимое
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read audit file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "follow") {
		t.Errorf("Expected audit log to contain 'follow', got: %s", content)
	}
	if !strings.Contains(content, "user456") {
		t.Errorf("Expected audit log to contain 'user456', got: %s", content)
	}
	if !strings.Contains(content, "https://example.org") {
		t.Errorf("Expected audit log to contain 'https://example.org', got: %s", content)
	}
}
