package repository

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"testing"

	"github.com/avitamin/go-shortener/internal/model"
)

// init отключает логирование для бенчмарков
func init() {
	log.SetOutput(io.Discard)
}

// BenchmarkInMemorySave измеряет скорость сохранения одной записи
func BenchmarkInMemorySave(b *testing.B) {
	repo := NewInMemoryStorage()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		url := model.URL{
			Short:    fmt.Sprintf("short%d", i),
			Original: fmt.Sprintf("https://example.com/%d", i),
			UserID:   "user1",
		}
		repo.Save(ctx, url)
	}
}

// BenchmarkInMemoryFind измеряет скорость поиска записи
func BenchmarkInMemoryFind(b *testing.B) {
	repo := NewInMemoryStorage()
	ctx := context.Background()

	// Предварительно заполняем хранилище
	for i := 0; i < 1000; i++ {
		url := model.URL{
			Short:    fmt.Sprintf("short%d", i),
			Original: fmt.Sprintf("https://example.com/%d", i),
			UserID:   "user1",
		}
		repo.Save(ctx, url)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.Find(fmt.Sprintf("short%d", i%1000))
	}
}

// BenchmarkInMemoryGetShort измеряет скорость получения короткой ссылки по оригинальной
func BenchmarkInMemoryGetShort(b *testing.B) {
	repo := NewInMemoryStorage()
	ctx := context.Background()

	// Предварительно заполняем хранилище
	for i := 0; i < 1000; i++ {
		url := model.URL{
			Short:    fmt.Sprintf("short%d", i),
			Original: fmt.Sprintf("https://example.com/%d", i),
			UserID:   "user1",
		}
		repo.Save(ctx, url)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.GetShort(ctx, fmt.Sprintf("https://example.com/%d", i%1000))
	}
}

// BenchmarkInMemorySaveBatch измеряет скорость пакетного сохранения
func BenchmarkInMemorySaveBatch(b *testing.B) {
	batchSizes := []int{10, 100, 1000}

	for _, size := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize_%d", size), func(b *testing.B) {
			repo := NewInMemoryStorage()
			ctx := context.Background()

			urls := make([]model.URL, size)
			for i := 0; i < size; i++ {
				urls[i] = model.URL{
					Short:    fmt.Sprintf("short%d", i),
					Original: fmt.Sprintf("https://example.com/%d", i),
					UserID:   "user1",
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				repo.SaveBatch(ctx, urls)
			}
		})
	}
}

// BenchmarkInMemoryConcurrentSave измеряет скорость конкурентного сохранения
func BenchmarkInMemoryConcurrentSave(b *testing.B) {
	concurrencyLevels := []int{1, 10, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			repo := NewInMemoryStorage()
			ctx := context.Background()

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					url := model.URL{
						Short:    fmt.Sprintf("short%d", i),
						Original: fmt.Sprintf("https://example.com/%d", i),
						UserID:   "user1",
					}
					repo.Save(ctx, url)
					i++
				}
			})
		})
	}
}

// BenchmarkInMemoryConcurrentRead измеряет скорость конкурентного чтения
func BenchmarkInMemoryConcurrentRead(b *testing.B) {
	repo := NewInMemoryStorage()
	ctx := context.Background()

	// Предварительно заполняем хранилище
	for i := 0; i < 10000; i++ {
		url := model.URL{
			Short:    fmt.Sprintf("short%d", i),
			Original: fmt.Sprintf("https://example.com/%d", i),
			UserID:   "user1",
		}
		repo.Save(ctx, url)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			repo.Find(fmt.Sprintf("short%d", i%10000))
			i++
		}
	})
}

// BenchmarkInMemoryMixedOperations измеряет скорость смешанных операций
func BenchmarkInMemoryMixedOperations(b *testing.B) {
	repo := NewInMemoryStorage()
	ctx := context.Background()

	// Предварительно заполняем хранилище
	for i := 0; i < 1000; i++ {
		url := model.URL{
			Short:    fmt.Sprintf("short%d", i),
			Original: fmt.Sprintf("https://example.com/%d", i),
			UserID:   "user1",
		}
		repo.Save(ctx, url)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// 80% чтения, 20% записи
			if i%5 == 0 {
				url := model.URL{
					Short:    fmt.Sprintf("short%d", i),
					Original: fmt.Sprintf("https://example.com/%d", i),
					UserID:   "user1",
				}
				repo.Save(ctx, url)
			} else {
				repo.Find(fmt.Sprintf("short%d", i%1000))
			}
			i++
		}
	})
}

// BenchmarkInMemoryDeleteUserURLs измеряет скорость удаления URL пользователя
func BenchmarkInMemoryDeleteUserURLs(b *testing.B) {
	repo := NewInMemoryStorage()
	ctx := context.Background()

	// Предварительно заполняем хранилище
	for i := 0; i < 1000; i++ {
		url := model.URL{
			Short:    fmt.Sprintf("short%d", i),
			Original: fmt.Sprintf("https://example.com/%d", i),
			UserID:   "user1",
		}
		repo.Save(ctx, url)
	}

	deleteURLs := []string{"short0", "short1", "short2", "short3", "short4"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.DeleteUserURLs(ctx, "user1", deleteURLs)
	}
}

// BenchmarkInMemoryLockContention измеряет конкуренцию за блокировку
func BenchmarkInMemoryLockContention(b *testing.B) {
	repo := NewInMemoryStorage()
	ctx := context.Background()

	// Предварительно заполняем хранилище
	for i := 0; i < 1000; i++ {
		url := model.URL{
			Short:    fmt.Sprintf("short%d", i),
			Original: fmt.Sprintf("https://example.com/%d", i),
			UserID:   "user1",
		}
		repo.Save(ctx, url)
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
				if i%2 == 0 {
					repo.Find(fmt.Sprintf("short%d", i%1000))
				} else {
					url := model.URL{
						Short:    fmt.Sprintf("new_short%d_%d", workerID, i),
						Original: fmt.Sprintf("https://example.com/new/%d/%d", workerID, i),
						UserID:   fmt.Sprintf("user%d", workerID),
					}
					repo.Save(ctx, url)
				}
			}
		}(w)
	}
	wg.Wait()
}

// BenchmarkInMemoryScalability измеряет масштабируемость с разными размерами данных
func BenchmarkInMemoryScalability(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			repo := NewInMemoryStorage()
			ctx := context.Background()

			// Заполняем хранилище
			for i := 0; i < size; i++ {
				url := model.URL{
					Short:    fmt.Sprintf("short%d", i),
					Original: fmt.Sprintf("https://example.com/%d", i),
					UserID:   "user1",
				}
				repo.Save(ctx, url)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				repo.Find(fmt.Sprintf("short%d", i%size))
			}
		})
	}
}
