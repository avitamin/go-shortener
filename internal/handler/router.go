package handler

import (
	"context"
	"io"
	"net/http"
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

	rtr.Post("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "text/plain" {
			http.Error(w, "Некорректный Content-Type", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)

		if err != nil || len(body) == 0 {
			http.Error(w, "Пустое тело запроса", http.StatusBadRequest)
		}

		defer r.Body.Close()

		short, err := service.Shorten(string(body))

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)

		w.Write([]byte(short))
	})

	rtr.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "Пустой ID", http.StatusBadRequest)
		}

		original, err := service.Resolve(id)

		if err != nil {
			if err == repository.ErrNotFound {
				http.Error(w, "URL не найден", http.StatusBadRequest)
				return
			}

			log.Error(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Location", original)
		w.WriteHeader(http.StatusTemporaryRedirect)
	})

	rtr.Post("/api/shorten", func(w http.ResponseWriter, r *http.Request) {
		var req model.ShortenRequest

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

		short, err := service.Shorten(req.URL)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := model.ShortenResponse{Result: short}
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
