package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/avitamin/go-shortener/internal/repository"
	"github.com/avitamin/go-shortener/internal/service"
)

func NewRouter(service *service.ShortenerService) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {

		case http.MethodPost:
			handleShorten(w, r, service)

		case http.MethodGet:
			handleResolve(w, r, service)

		default:
			http.Error(w, "Метод недоступен", http.StatusMethodNotAllowed)
		}
	})

	return mux
}

func handleShorten(w http.ResponseWriter, r *http.Request, service *service.ShortenerService) {
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

	// todo write body
}

func handleResolve(w http.ResponseWriter, r *http.Request, service *service.ShortenerService) {
	id := strings.TrimPrefix(r.URL.Path, "/")
	if id == "" {
		http.Error(w, "Пустой ID", http.StatusBadRequest)
	}

	original, err := service.Resolve(id)

	if err != nil {
		if err == repository.ErrNotFound {
			http.Error(w, "URL Не найден", http.StatusBadRequest)
			return
		}

		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", original)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
