package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"sync"
	"testing"

	"github.com/avitamin/go-shortener/internal/audit"
	"github.com/avitamin/go-shortener/internal/config"
	"github.com/avitamin/go-shortener/internal/model"
	"github.com/avitamin/go-shortener/internal/repository"
)

// init отключает логирование для бенчмарков
func init() {
	log.SetOutput(io.Discard)
}

// Глобальные переменные для переиспользования сервиса в бенчмарках
var (
	benchRepo    repository.Repository
	benchConfig  *config.Config
	benchService *ShortenerService
	benchOnce    sync.Once
)

// getBenchService возвращает singleton сервис для бенчмарков
func getBenchService() *ShortenerService {
	benchOnce.Do(func() {
		benchRepo = repository.NewInMemoryStorage()
		benchConfig = &config.Config{
			BaseURL: "http://localhost:8080",
		}
		benchService = NewShortenerService(benchRepo, benchConfig, audit.NewServiceFromConfig(benchConfig))
	})
	return benchService
}

// BenchmarkGenerateID измеряет скорость генерации ID
func BenchmarkGenerateID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateID()
	}
}

// BenchmarkGenerateIDParallel измеряет скорость параллельной генерации ID
func BenchmarkGenerateIDParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			generateID()
		}
	})
}

// BenchmarkGenerateIDCryptoRand сравнение с прямым использованием crypto/rand
func BenchmarkGenerateIDCryptoRand(b *testing.B) {
	b.Run("CurrentImplementation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			generateID()
		}
	})

	b.Run("DirectCryptoRand", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b := make([]byte, 6)
			io.ReadFull(rand.Reader, b)
			_ = base64.URLEncoding.EncodeToString(b)
		}
	})
}

// BenchmarkShorten измеряет скорость создания короткой ссылки
func BenchmarkShorten(b *testing.B) {
	service := getBenchService()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.Shorten(ctx, fmt.Sprintf("https://example.com/%d", i))
	}
}

// BenchmarkShortenWithContext измеряет скорость создания короткой ссылки с UserID в контексте
func BenchmarkShortenWithContext(b *testing.B) {
	service := getBenchService()
	ctx := context.WithValue(context.Background(), model.ContextUserID, "user123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.Shorten(ctx, fmt.Sprintf("https://example.com/%d", i))
	}
}

// BenchmarkResolve измеряет скорость получения оригинальной ссылки
func BenchmarkResolve(b *testing.B) {
	service := getBenchService()
	ctx := context.Background()

	// Предварительно создаем 1000 ссылок
	shorts := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		shortURL, _ := service.Shorten(ctx, fmt.Sprintf("https://example.com/resolve/%d", i))
		// Извлекаем короткий ID из полного URL
		shorts[i] = shortURL[len(benchConfig.BaseURL)+1:]
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.Resolve(shorts[i%1000])
	}
}

// BenchmarkShortenBatch измеряет скорость пакетного создания ссылок
func BenchmarkShortenBatch(b *testing.B) {
	batchSizes := []int{10, 100, 1000}

	for _, size := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize_%d", size), func(b *testing.B) {
			repo := repository.NewInMemoryStorage()
			cfg := &config.Config{
				BaseURL: "http://localhost:8080",
			}
			service := NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
			ctx := context.Background()

			urls := make([]string, size)
			for i := 0; i < size; i++ {
				urls[i] = fmt.Sprintf("https://example.com/%d", i)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				service.ShortenBatch(ctx, urls)
			}
		})
	}
}

// BenchmarkShortenBatchWithDuplicates измеряет скорость пакетного создания с дубликатами
func BenchmarkShortenBatchWithDuplicates(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}
	service := NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	ctx := context.Background()

	// Создаем батч с дубликатами (50% дубликатов)
	urls := make([]string, 100)
	for i := 0; i < 100; i++ {
		urls[i] = fmt.Sprintf("https://example.com/%d", i%50)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ShortenBatch(ctx, urls)
	}
}

// BenchmarkGetShort измеряет скорость получения существующей короткой ссылки
func BenchmarkGetShort(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}
	service := NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	ctx := context.Background()

	// Предварительно создаем ссылки
	for i := 0; i < 1000; i++ {
		service.Shorten(ctx, fmt.Sprintf("https://example.com/%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GetShort(ctx, fmt.Sprintf("https://example.com/%d", i%1000))
	}
}

// BenchmarkConcurrentShorten измеряет скорость конкурентного создания ссылок
func BenchmarkConcurrentShorten(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}
	service := NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			service.Shorten(ctx, fmt.Sprintf("https://example.com/%d", i))
			i++
		}
	})
}

// BenchmarkConcurrentResolve измеряет скорость конкурентного получения ссылок
func BenchmarkConcurrentResolve(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}
	service := NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	ctx := context.Background()

	// Предварительно создаем ссылки
	shorts := make([]string, 10000)
	for i := 0; i < 10000; i++ {
		shortURL, _ := service.Shorten(ctx, fmt.Sprintf("https://example.com/%d", i))
		shorts[i] = shortURL[len(cfg.BaseURL)+1:]
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			service.Resolve(shorts[i%10000])
			i++
		}
	})
}

// BenchmarkMixedOperations измеряет скорость смешанных операций
func BenchmarkMixedOperations(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}
	service := NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	ctx := context.Background()

	// Предварительно создаем ссылки
	shorts := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		shortURL, _ := service.Shorten(ctx, fmt.Sprintf("https://example.com/%d", i))
		shorts[i] = shortURL[len(cfg.BaseURL)+1:]
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// 70% Resolve, 30% Shorten
			if i%10 < 7 {
				service.Resolve(shorts[i%1000])
			} else {
				service.Shorten(ctx, fmt.Sprintf("https://example.com/new/%d", i))
			}
			i++
		}
	})
}

// BenchmarkGetUserURLs измеряет скорость получения URL пользователя
func BenchmarkGetUserURLs(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}
	service := NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	ctx := context.WithValue(context.Background(), model.ContextUserID, "user123")

	// Создаем 100 ссылок для пользователя
	for i := 0; i < 100; i++ {
		service.Shorten(ctx, fmt.Sprintf("https://example.com/%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GetUserURLs(ctx)
	}
}

// BenchmarkDeleteUserURLs измеряет скорость удаления URL пользователя
func BenchmarkDeleteUserURLs(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}
	service := NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	ctx := context.WithValue(context.Background(), model.ContextUserID, "user123")

	// Создаем ссылки для пользователя
	shorts := make([]string, 100)
	for i := 0; i < 100; i++ {
		shortURL, _ := service.Shorten(ctx, fmt.Sprintf("https://example.com/%d", i))
		shorts[i] = shortURL[len(cfg.BaseURL)+1:]
	}

	deleteURLs := shorts[:10]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.DeleteUserURLs(ctx, deleteURLs)
	}
}

// BenchmarkCreateModel измеряет скорость создания модели URL
func BenchmarkCreateModel(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}
	service := NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	ctx := context.WithValue(context.Background(), model.ContextUserID, "user123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.createModel(ctx, fmt.Sprintf("https://example.com/%d", i))
	}
}

// BenchmarkGetAbsoluteShortURL измеряет скорость формирования абсолютного URL
func BenchmarkGetAbsoluteShortURL(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}
	service := NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GetAbsoluteShortURL("abc123")
	}
}

// BenchmarkHighContentionScenario измеряет производительность при высокой конкуренции
func BenchmarkHighContentionScenario(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}
	service := NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	ctx := context.Background()

	// Создаем небольшой набор популярных URL для высокой конкуренции
	popularURLs := []string{
		"https://example.com/popular1",
		"https://example.com/popular2",
		"https://example.com/popular3",
		"https://example.com/popular4",
		"https://example.com/popular5",
	}

	for _, url := range popularURLs {
		service.Shorten(ctx, url)
	}

	b.ResetTimer()

	var wg sync.WaitGroup
	workers := 100
	opsPerWorker := b.N / workers

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				// Все потоки обращаются к одним и тем же URL
				url := popularURLs[i%len(popularURLs)]
				service.GetShort(ctx, url)
			}
		}(w)
	}
	wg.Wait()
}

// BenchmarkMemoryAllocation измеряет аллокации памяти
func BenchmarkMemoryAllocation(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}
	service := NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		service.Shorten(ctx, fmt.Sprintf("https://example.com/%d", i))
	}
}
