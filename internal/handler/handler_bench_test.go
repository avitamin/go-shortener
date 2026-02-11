package handler

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avitamin/go-shortener/internal/audit"
	"github.com/avitamin/go-shortener/internal/config"
	"github.com/avitamin/go-shortener/internal/repository"
	"github.com/avitamin/go-shortener/internal/service"
)

// init отключает логирование для бенчмарков
func init() {
	log.SetOutput(io.Discard)
}

// newBenchRouter создает роутер без middleware для бенчмарков
func newBenchRouter(svc *service.ShortenerService) http.Handler {
	mux := http.NewServeMux()

	// POST / - создание короткой ссылки
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			return
		}
		if r.Header.Get("Content-Type") != "text/plain" {
			http.Error(w, "Некорректный Content-Type", http.StatusBadRequest)
			return
		}

		body := new(bytes.Buffer)
		body.ReadFrom(r.Body)
		defer r.Body.Close()

		orig := body.String()
		short, ok := svc.GetShort(r.Context(), orig)
		if ok {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(short))
			return
		}

		if err := validateOriginalURL(orig); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		short, err := svc.Shorten(r.Context(), orig)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(short))
	})

	return mux
}

// BenchmarkHandlerPostShorten измеряет скорость обработки POST / запроса
func BenchmarkHandlerPostShorten(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL:   "http://localhost:8080",
		SecretKey: "test-secret-key-32-bytes-long!!",
	}
	svc := service.NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	router := newBenchRouter(svc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		url := fmt.Sprintf("https://example.com/%d", i)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(url))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkHandlerGetRedirect измеряет скорость обработки GET /{id} запроса
func BenchmarkHandlerGetRedirect(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL:   "http://localhost:8080",
		SecretKey: "test-secret-key-32-bytes-long!!",
	}
	svc := service.NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	router, _ := NewRouter(svc)

	// Предварительно создаем ссылки
	shorts := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		url := fmt.Sprintf("https://example.com/%d", i)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(url))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		// Извлекаем короткий ID из ответа
		shorts[i] = w.Body.String()[len(cfg.BaseURL)+1:]
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/"+shorts[i%1000], nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkHandlerAPIShortenJSON измеряет скорость обработки POST /api/shorten
func BenchmarkHandlerAPIShortenJSON(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL:   "http://localhost:8080",
		SecretKey: "test-secret-key-32-bytes-long!!",
	}
	svc := service.NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	router, _ := NewRouter(svc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jsonBody := fmt.Sprintf(`{"url":"https://example.com/%d"}`, i)
		req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkHandlerBatchShorten измеряет скорость обработки POST /api/shorten/batch
func BenchmarkHandlerBatchShorten(b *testing.B) {
	batchSizes := []int{10, 100, 1000}

	for _, size := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize_%d", size), func(b *testing.B) {
			repo := repository.NewInMemoryStorage()
			cfg := &config.Config{
				BaseURL:   "http://localhost:8080",
				SecretKey: "test-secret-key-32-bytes-long!!",
			}
			svc := service.NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
			router, _ := NewRouter(svc)

			// Формируем JSON для батча
			var jsonBody bytes.Buffer
			jsonBody.WriteString("[")
			for i := 0; i < size; i++ {
				if i > 0 {
					jsonBody.WriteString(",")
				}
				jsonBody.WriteString(fmt.Sprintf(`{"correlation_id":"%d","original_url":"https://example.com/%d"}`, i, i))
			}
			jsonBody.WriteString("]")

			batchJSON := jsonBody.String()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBufferString(batchJSON))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
			}
		})
	}
}

// BenchmarkHandlerPing измеряет скорость обработки GET /ping
func BenchmarkHandlerPing(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL:   "http://localhost:8080",
		SecretKey: "test-secret-key-32-bytes-long!!",
	}
	svc := service.NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	router, _ := NewRouter(svc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkHandlerConcurrentRequests измеряет скорость конкурентных запросов
func BenchmarkHandlerConcurrentRequests(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL:   "http://localhost:8080",
		SecretKey: "test-secret-key-32-bytes-long!!",
	}
	svc := service.NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	router, _ := NewRouter(svc)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			url := fmt.Sprintf("https://example.com/%d", i)
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(url))
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			i++
		}
	})
}

// BenchmarkHandlerMixedOperations измеряет скорость смешанных операций
func BenchmarkHandlerMixedOperations(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL:   "http://localhost:8080",
		SecretKey: "test-secret-key-32-bytes-long!!",
	}
	svc := service.NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	router, _ := NewRouter(svc)

	// Предварительно создаем ссылки
	shorts := make([]string, 100)
	for i := 0; i < 100; i++ {
		url := fmt.Sprintf("https://example.com/%d", i)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(url))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		shorts[i] = w.Body.String()[len(cfg.BaseURL)+1:]
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// 70% GET, 30% POST
			if i%10 < 7 {
				req := httptest.NewRequest(http.MethodGet, "/"+shorts[i%100], nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
			} else {
				url := fmt.Sprintf("https://example.com/new/%d", i)
				req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(url))
				req.Header.Set("Content-Type", "text/plain")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
			}
			i++
		}
	})
}

// BenchmarkHandlerValidation измеряет скорость валидации URL
func BenchmarkHandlerValidation(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL:   "http://localhost:8080",
		SecretKey: "test-secret-key-32-bytes-long!!",
	}
	svc := service.NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	router, _ := NewRouter(svc)

	b.Run("ValidURL", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			url := fmt.Sprintf("https://example.com/%d", i)
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(url))
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("InvalidURL", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			url := fmt.Sprintf("invalid-url-%d", i)
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(url))
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// BenchmarkHandlerJSONMarshalUnmarshal измеряет скорость сериализации/десериализации JSON
func BenchmarkHandlerJSONMarshalUnmarshal(b *testing.B) {
	repo := repository.NewInMemoryStorage()
	cfg := &config.Config{
		BaseURL:   "http://localhost:8080",
		SecretKey: "test-secret-key-32-bytes-long!!",
	}
	svc := service.NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))
	router, _ := NewRouter(svc)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		jsonBody := fmt.Sprintf(`{"url":"https://example.com/%d"}`, i)
		req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkValidateOriginalURL измеряет скорость валидации URL
func BenchmarkValidateOriginalURL(b *testing.B) {
	b.Run("ValidHTTPS", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			validateOriginalURL("https://example.com")
		}
	})

	b.Run("ValidHTTP", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			validateOriginalURL("http://example.com")
		}
	})

	b.Run("Invalid", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			validateOriginalURL("invalid-url")
		}
	})
}
