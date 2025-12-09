package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/avitamin/go-shortener/internal/logger"
	"github.com/avitamin/go-shortener/internal/repository"
	"github.com/avitamin/go-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/goccy/go-json"

	lclmw "github.com/avitamin/go-shortener/internal/middleware"
	"github.com/avitamin/go-shortener/internal/model"
)

func NewRouter(service *service.ShortenerService) (http.Handler, error) {
	log, err := logger.New()
	if err != nil {
		return nil, err
	}

	rtr := chi.NewRouter()

	rtr.Use(middleware.RequestID)
	rtr.Use(middleware.Logger)
	rtr.Use(middleware.Recoverer)

	rtr.Use(lclmw.ZapLogger(log))
	rtr.Use(lclmw.GzipRequest(log))
	rtr.Use(lclmw.GzipResponse)
	rtr.Use(lclmw.Auth(service.Config.SecretKey, log))

	rtr.Post("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "text/plain" {
			http.Error(w, "Некорректный Content-Type", http.StatusBadRequest)
			return
		}

		var statusCode int

		body, err := io.ReadAll(r.Body)

		if err != nil || len(body) == 0 {
			http.Error(w, "Пустое тело запроса", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		orig := strings.TrimSpace(string(body))
		short, ok := service.GetShort(r.Context(), orig)
		if ok {
			statusCode = http.StatusConflict
		} else {
			err := validateOriginalURL(orig)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			statusCode = http.StatusCreated

			short, err = service.Shorten(r.Context(), orig)
			if err != nil {
				log.Error(err.Error())
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(statusCode)

		w.Write([]byte(short))
	})

	rtr.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "Пустой ID", http.StatusBadRequest)
		}

		original, err := service.Resolve(id)
		if err != nil {
			log.Error(err.Error())

			switch {
			case errors.Is(err, repository.ErrIsDeleted):
				http.Error(w, "URL был удалён", http.StatusGone)
				return
			case errors.Is(err, repository.ErrNotFound):
				http.Error(w, "URL не найден", http.StatusBadRequest)
				return
			}

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Location", original)
		w.WriteHeader(http.StatusTemporaryRedirect)
	})

	rtr.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
		var req model.ShortenRequest
		var statusCode int

		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Некорректный Content-Type", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)

		if err != nil || len(body) == 0 {
			http.Error(w, "Пустое тело запроса", http.StatusBadRequest)
		}
		defer r.Body.Close()

		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Некорректный JSON", http.StatusBadRequest)
			return
		}

		orig := strings.TrimSpace(req.URL)
		short, ok := service.GetShort(r.Context(), orig)
		if ok {
			statusCode = http.StatusConflict
		} else {
			err := validateOriginalURL(orig)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			statusCode = http.StatusCreated
			short, err = service.Shorten(r.Context(), orig)
			if err != nil {
				log.Error(err.Error())
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		}

		resp := model.ShortenResponse{Result: short}
		respBytes, err := json.Marshal(resp)
		if err != nil {
			log.Error(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		w.Write(respBytes)
	})

	rtr.Post("/api/shorten/batch", func(w http.ResponseWriter, r *http.Request) {
		var req []model.BatchShortRequest

		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Некорректный Content-Type", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			http.Error(w, "Пустое тело запроса", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		if err := json.Unmarshal(body, &req); err != nil {
			log.Error(err.Error())
			http.Error(w, "Некорректный JSON", http.StatusBadRequest)
			return
		}

		// не отправляем пустые батчи
		if len(req) == 0 {
			http.Error(w, "Пустой массив", http.StatusBadRequest)
			return
		}

		// лимит 1000
		if len(req) > 1000 {
			http.Error(w, "Слишком большой батч", http.StatusRequestEntityTooLarge)
			return
		}

		// Валидация: каждый original непустой
		for _, it := range req {
			if strings.TrimSpace(it.Original) == "" {
				log.Error("Пустой original_url в батче, correlation_id: " + it.CorrelationID)
				http.Error(w, "Некорректный элемент в батче", http.StatusBadRequest)
				return
			}
		}

		// Собираем оригиналы в порядке запроса
		originals := make([]string, 0, len(req))
		for _, it := range req {
			originals = append(originals, it.Original)
		}

		// Контекст с таймаутом 30s
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		shorts, err := service.ShortenBatch(ctx, originals)
		if err != nil {
			log.Error(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// Формируем ответ — соответствие correlation_id -> short_url
		resp := make([]model.BatchShortenResponse, 0, len(req))
		for i, it := range req {
			resp = append(resp, model.BatchShortenResponse{
				CorrelationID: it.CorrelationID,
				ShortURL:      shorts[i],
			})
		}

		respBytes, err := json.Marshal(resp)
		if err != nil {
			log.Error(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(respBytes)
	})

	rtr.Delete("/api/user/urls", func(w http.ResponseWriter, r *http.Request) {
		var req []string

		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Некорректный Content-Type", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			http.Error(w, "Пустое тело запроса", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		if err := json.Unmarshal(body, &req); err != nil {
			log.Error(err.Error())
			http.Error(w, "Некорректный JSON", http.StatusBadRequest)
			return
		}

		if len(req) == 0 {
			http.Error(w, "Пустой массив", http.StatusBadRequest)
			return
		}

		err = service.DeleteUserURLs(r.Context(), req)
		if err != nil {
			log.Error(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	})

	rtr.Get("/api/user/urls", func(w http.ResponseWriter, r *http.Request) {

		// Контекст с таймаутом 30s
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		urls, err := service.GetUserURLs(ctx)
		if err != nil {
			log.Error(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if len(urls) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		resp := make([]model.UserURLsResponse, 0, len(urls))
		for _, it := range urls {
			resp = append(resp, model.UserURLsResponse{
				OriginalURL: it.Original,
				ShortURL:    it.Short,
			})
		}

		respBytes, err := json.Marshal(resp)
		if err != nil {
			log.Error(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(respBytes)
	})

	rtr.Get("/ping", func(w http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()

		if err := service.PingContext(ctx); err != nil {
			log.Error(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	return rtr, nil
}

var ErrInvalidURL = errors.New("некорректный url")

func validateOriginalURL(orig string) error {

	if !strings.HasPrefix(orig, "http://") && !strings.HasPrefix(orig, "https://") {
		return ErrInvalidURL
	}

	return nil
}
