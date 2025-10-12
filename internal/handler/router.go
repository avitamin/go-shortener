package handler

import (
	"io"
	"net/http"

	"github.com/avitamin/go-shortener/internal/repository"
	"github.com/avitamin/go-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(service *service.ShortenerService) http.Handler {
	rtr := chi.NewRouter()

	rtr.Use(middleware.RequestID)
	rtr.Use(middleware.Logger)
	rtr.Use(middleware.Recoverer)

	rtr.Post("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "text/plain" {
			http.Error(w, "Некорректный Content-Type", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)

		if err != nil || len(body) == 0 {
			http.Error(w, "Пустое тело запроса", http.StatusBadRequest)
		}

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

			http.Error(w, "внутренняя ошибка", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Location", original)
		w.WriteHeader(http.StatusTemporaryRedirect)
	})

	return rtr
}
