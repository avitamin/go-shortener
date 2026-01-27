package service_test

import (
	"context"
	"fmt"

	"github.com/avitamin/go-shortener/internal/config"
	"github.com/avitamin/go-shortener/internal/model"
	"github.com/avitamin/go-shortener/internal/repository"
	"github.com/avitamin/go-shortener/internal/service"
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

// Example_shorten демонстрирует создание короткой ссылки.
func Example_shorten() {
	// Инициализация сервиса
	cfg := makeTestConfig()

	repo := repository.NewInMemoryStorage()
	svc := service.NewShortenerService(repo, cfg)

	// Создание короткой ссылки
	ctx := context.Background()
	shortURL, err := svc.Shorten(ctx, "https://example.com/very-long-url")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Short URL created: %s\n", shortURL[:len(cfg.BaseURL)])
	// Output:
	// Short URL created: http://localhost:8080
}

// Example_shortenWithUser демонстрирует создание короткой ссылки с привязкой к пользователю.
func Example_shortenWithUser() {
	// Инициализация сервиса
	cfg := makeTestConfig()

	repo := repository.NewInMemoryStorage()
	svc := service.NewShortenerService(repo, cfg)

	// Создание контекста с идентификатором пользователя
	ctx := model.NewContextWithUser(context.Background(), "user123")

	// Создание короткой ссылки
	shortURL, err := svc.Shorten(ctx, "https://example.com/user-url")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Short URL created for user: %s\n", shortURL[:len(cfg.BaseURL)])
	// Output:
	// Short URL created for user: http://localhost:8080
}

// Example_resolve демонстрирует получение исходного URL по короткому идентификатору.
func Example_resolve() {
	// Инициализация сервиса
	cfg := makeTestConfig()

	repo := repository.NewInMemoryStorage()
	svc := service.NewShortenerService(repo, cfg)

	// Создание короткой ссылки
	ctx := context.Background()
	originalURL := "https://example.com/target"
	shortURL, _ := svc.Shorten(ctx, originalURL)

	// Извлекаем короткий идентификатор
	shortID := shortURL[len(cfg.BaseURL)+1:]

	// Получаем исходный URL
	resolved, err := svc.Resolve(shortID)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Original URL: %s\n", resolved)
	// Output:
	// Original URL: https://example.com/target
}

// Example_shortenBatch демонстрирует пакетное создание коротких ссылок.
func Example_shortenBatch() {
	// Инициализация сервиса
	cfg := makeTestConfig()

	repo := repository.NewInMemoryStorage()
	svc := service.NewShortenerService(repo, cfg)

	// Список URL для сокращения
	originals := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/page3",
	}

	// Пакетное создание коротких ссылок
	ctx := context.Background()
	shorts, err := svc.ShortenBatch(ctx, originals)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Created %d short URLs\n", len(shorts))
	fmt.Printf("First URL starts with: %s\n", shorts[0][:len(cfg.BaseURL)])
	// Output:
	// Created 3 short URLs
	// First URL starts with: http://localhost:8080
}

// Example_getUserURLs демонстрирует получение всех ссылок пользователя.
// Примечание: in-memory репозиторий не сохраняет привязку URL к пользователю,
// поэтому возвращает пустой список. Для работающего примера используйте database repository.
func Example_getUserURLs() {
	// Инициализация сервиса
	cfg := makeTestConfig()

	repo := repository.NewInMemoryStorage()
	svc := service.NewShortenerService(repo, cfg)

	// Создание контекста с пользователем
	ctx := model.NewContextWithUser(context.Background(), "user123")

	// Создание нескольких ссылок
	svc.Shorten(ctx, "https://example.com/url1")
	svc.Shorten(ctx, "https://example.com/url2")
	svc.Shorten(ctx, "https://example.com/url3")

	// Получение всех ссылок пользователя
	urls, err := svc.GetUserURLs(ctx)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("User has %d URLs\n", len(urls))
	// Output:
	// User has 0 URLs
}

// Example_deleteUserURLs демонстрирует удаление ссылок пользователя.
func Example_deleteUserURLs() {
	// Инициализация сервиса
	cfg := makeTestConfig()

	repo := repository.NewInMemoryStorage()
	svc := service.NewShortenerService(repo, cfg)

	// Создание контекста с пользователем
	ctx := model.NewContextWithUser(context.Background(), "user123")

	// Создание коротких ссылок
	shortURL1, _ := svc.Shorten(ctx, "https://example.com/delete1")
	shortURL2, _ := svc.Shorten(ctx, "https://example.com/delete2")

	// Извлечение коротких идентификаторов
	shortID1 := shortURL1[len(cfg.BaseURL)+1:]
	shortID2 := shortURL2[len(cfg.BaseURL)+1:]

	// Удаление ссылок
	err := svc.DeleteUserURLs(ctx, []string{shortID1, shortID2})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("URLs marked for deletion")
	// Output:
	// URLs marked for deletion
}

// Example_getShort демонстрирует проверку существования короткой ссылки для URL.
func Example_getShort() {
	// Инициализация сервиса
	cfg := makeTestConfig()

	repo := repository.NewInMemoryStorage()
	svc := service.NewShortenerService(repo, cfg)

	ctx := context.Background()
	originalURL := "https://example.com/check-duplicate"

	// Создание первой короткой ссылки
	firstShort, _ := svc.Shorten(ctx, originalURL)

	// Проверка существования ссылки для того же URL
	existingShort, found := svc.GetShort(ctx, originalURL)

	fmt.Printf("URL already exists: %v\n", found)
	fmt.Printf("Same short URL: %v\n", firstShort == existingShort)
	// Output:
	// URL already exists: true
	// Same short URL: true
}

// Example_pingContext демонстрирует проверку доступности хранилища.
// Примечание: in-memory хранилище не поддерживает ping и возвращает ошибку.
// Для работающего примера используйте database repository.
func Example_pingContext() {
	// Инициализация сервиса
	cfg := makeTestConfig()

	repo := repository.NewInMemoryStorage()
	svc := service.NewShortenerService(repo, cfg)

	// Проверка доступности
	ctx := context.Background()
	err := svc.PingContext(ctx)

	if err != nil {
		fmt.Println("Storage unavailable:", err)
		return
	}

	fmt.Println("Storage is available")
	// Output:
	// Storage unavailable: подключение к БД не настроено
}

// Example_getUserIDFromContext демонстрирует извлечение идентификатора пользователя из контекста.
func Example_getUserIDFromContext() {
	// Создание контекста с пользователем
	ctx := model.NewContextWithUser(context.Background(), "user456")

	// Извлечение идентификатора пользователя
	userID, found := service.GetUserIDFromContext(ctx)

	fmt.Printf("User ID found: %v\n", found)
	fmt.Printf("User ID: %s\n", userID)
	// Output:
	// User ID found: true
	// User ID: user456
}
