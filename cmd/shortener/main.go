package main

import (
	"log"
	"net/http"

	"github.com/avitamin/go-shortener/internal/config"
	"github.com/avitamin/go-shortener/internal/handler"
	"github.com/avitamin/go-shortener/internal/repository"
	"github.com/avitamin/go-shortener/internal/service"
)

func main() {

	config := config.New()

	repo := repository.NewInMemoryRepository()
	svc := service.NewShortenerService(repo, config.BaseURL)
	mux := handler.NewRouter(svc)

	log.Printf("Запускаем сервер по адресу %s\n", config.Address)

	if error := http.ListenAndServe(config.Address, mux); error != nil {
		log.Fatal(error)
	}
}
