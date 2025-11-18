package main

import (
	"log"
	"net/http"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/avitamin/go-shortener/internal/config"
	"github.com/avitamin/go-shortener/internal/handler"
	"github.com/avitamin/go-shortener/internal/repository"
	"github.com/avitamin/go-shortener/internal/service"
)

func main() {

	cfg, err := config.New(true)
	if err != nil {
		log.Fatal(err)
	}

	repo, err := repository.NewRepository(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Close()

	svc := service.NewShortenerService(repo, cfg.BaseURL)

	rtr, err := handler.NewRouter(svc)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Запускаем сервер по адресу %s\n", cfg.Address)

	if err := http.ListenAndServe(cfg.Address, rtr); err != nil {
		log.Fatal(err)
	}
}
