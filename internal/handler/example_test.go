package handler_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/avitamin/go-shortener/internal/config"
	"github.com/avitamin/go-shortener/internal/handler"
	"github.com/avitamin/go-shortener/internal/repository"
	"github.com/avitamin/go-shortener/internal/service"
	"github.com/goccy/go-json"
)

// Вспомогательная функция для создания конфигурации без парсинга флагов
func makeTestConfig() *config.Config {
	return &config.Config{
		Address:         "localhost:8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "./runtime/storage",
		SecretKey:       "test_secret_key",
	}
}

// Example_createShortURL демонстрирует создание короткой ссылки через POST / (text/plain).
func Example_createShortURL() {
	// Инициализация сервиса
	cfg := makeTestConfig()

	repo := repository.NewInMemoryStorage()
	svc := service.NewShortenerService(repo, cfg)
	router, _ := handler.NewRouter(svc)

	// Создание тестового запроса
	requestBody := "https://example.com/very-long-url"
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(requestBody))
	req.Header.Set("Content-Type", "text/plain")

	// Выполнение запроса
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Short URL created: %s\n", string(body)[:len(cfg.BaseURL)])
	// Output:
	// Status: 201
	// Short URL created: http://localhost:8080
}

// Example_createShortURLJSON демонстрирует создание короткой ссылки через POST /api/shorten (JSON).
func Example_createShortURLJSON() {
	// Инициализация сервиса
	cfg := makeTestConfig()

	repo := repository.NewInMemoryStorage()
	svc := service.NewShortenerService(repo, cfg)
	router, _ := handler.NewRouter(svc)

	// Подготовка JSON запроса
	requestJSON := `{"url":"https://example.com/api-test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(requestJSON))
	req.Header.Set("Content-Type", "application/json")

	// Выполнение запроса
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var response map[string]string
	json.Unmarshal(body, &response)

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Response contains result: %v\n", response["result"] != "")
	// Output:
	// Status: 201
	// Response contains result: true
}

// Example_batchCreateShortURLs демонстрирует пакетное создание коротких ссылок через POST /api/shorten/batch.
func Example_batchCreateShortURLs() {
	// Инициализация сервиса
	cfg := makeTestConfig()

	repo := repository.NewInMemoryStorage()
	svc := service.NewShortenerService(repo, cfg)
	router, _ := handler.NewRouter(svc)

	// Подготовка батч-запроса
	requestJSON := `[
		{"correlation_id":"1","original_url":"https://example.com/page1"},
		{"correlation_id":"2","original_url":"https://example.com/page2"}
	]`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBufferString(requestJSON))
	req.Header.Set("Content-Type", "application/json")

	// Выполнение запроса
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var response []map[string]string
	json.Unmarshal(body, &response)

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Number of URLs created: %d\n", len(response))
	fmt.Printf("First correlation_id: %s\n", response[0]["correlation_id"])
	// Output:
	// Status: 201
	// Number of URLs created: 2
	// First correlation_id: 1
}

// Example_redirectShortURL демонстрирует переход по короткой ссылке через GET /{id}.
func Example_redirectShortURL() {
	// Инициализация сервиса
	cfg := makeTestConfig()

	repo := repository.NewInMemoryStorage()
	svc := service.NewShortenerService(repo, cfg)
	router, _ := handler.NewRouter(svc)

	// Сначала создаем короткую ссылку
	ctx := context.Background()
	shortURL, _ := svc.Shorten(ctx, "https://example.com/target")
	shortID := shortURL[len(cfg.BaseURL)+1:] // Извлекаем короткий ID

	// Переход по короткой ссылке
	req := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Location: %s\n", resp.Header.Get("Location"))
	// Output:
	// Status: 307
	// Location: https://example.com/target
}

// Example_getUserURLs демонстрирует получение списка URL пользователя через GET /api/user/urls.
// Примечание: in-memory хранилище не сохраняет ассоциацию URL с пользователем,
// поэтому возвращает 204 (No Content). Для работающего примера используйте database repository.
func Example_getUserURLs() {
	// Инициализация сервиса
	cfg := makeTestConfig()

	repo := repository.NewInMemoryStorage()
	svc := service.NewShortenerService(repo, cfg)
	router, _ := handler.NewRouter(svc)

	// Создаем несколько ссылок - используем POST запрос чтобы middleware установил cookie
	req1 := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("https://example.com/url1"))
	req1.Header.Set("Content-Type", "text/plain")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	// Получаем cookie из первого ответа
	resp1 := w1.Result()
	defer resp1.Body.Close()
	cookies := resp1.Cookies()

	// Создаем второй URL с тем же пользователем
	req2 := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("https://example.com/url2"))
	req2.Header.Set("Content-Type", "text/plain")
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	resp2 := w2.Result()
	defer resp2.Body.Close()

	// Получаем список URL пользователя с той же cookie
	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var urls []map[string]string
	json.Unmarshal(body, &urls)

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Number of URLs: %d\n", len(urls))
	// Output:
	// Status: 204
	// Number of URLs: 0
}

// Example_deleteUserURLs демонстрирует удаление URL пользователя через DELETE /api/user/urls.
func Example_deleteUserURLs() {
	// Инициализация сервиса
	cfg := makeTestConfig()

	repo := repository.NewInMemoryStorage()
	svc := service.NewShortenerService(repo, cfg)
	router, _ := handler.NewRouter(svc)

	// Подготовка запроса на удаление
	requestJSON := `["short1","short2"]`
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBufferString(requestJSON))
	req.Header.Set("Content-Type", "application/json")

	// Выполнение запроса
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
	// Output:
	// Status: 202
}

// Example_pingService демонстрирует проверку доступности сервиса через GET /ping.
// Примечание: in-memory хранилище не поддерживает ping, поэтому возвращается 500.
// Для работающего примера необходимо использовать database или file storage.
func Example_pingService() {
	// Инициализация сервиса
	cfg := makeTestConfig()

	repo := repository.NewInMemoryStorage()
	svc := service.NewShortenerService(repo, cfg)
	router, err := handler.NewRouter(svc)
	if err != nil {
		log.Fatal(err)
	}

	// Проверка доступности
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// In-memory storage не поддерживает ping
	fmt.Printf("Status: %d\n", resp.StatusCode)
	// Output:
	// Status: 500
}

// Example_duplicateURL демонстрирует поведение при попытке создания дубликата URL.
func Example_duplicateURL() {
	// Инициализация сервиса
	cfg := makeTestConfig()

	repo := repository.NewInMemoryStorage()
	svc := service.NewShortenerService(repo, cfg)
	router, _ := handler.NewRouter(svc)

	originalURL := "https://example.com/duplicate-test"

	// Первый запрос - создание
	req1 := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(originalURL))
	req1.Header.Set("Content-Type", "text/plain")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	// Второй запрос - дубликат
	req2 := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(originalURL))
	req2.Header.Set("Content-Type", "text/plain")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	resp1 := w1.Result()
	resp2 := w2.Result()
	defer resp1.Body.Close()
	defer resp2.Body.Close()

	body1, _ := io.ReadAll(resp1.Body)
	body2, _ := io.ReadAll(resp2.Body)

	fmt.Printf("First request status: %d\n", resp1.StatusCode)
	fmt.Printf("Second request status: %d\n", resp2.StatusCode)
	fmt.Printf("URLs are identical: %v\n", string(body1) == string(body2))
	// Output:
	// First request status: 201
	// Second request status: 409
	// URLs are identical: true
}
