package main

import (
	"database/sql"
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

	db, err := sql.Open("pgx", cfg.DatabaseDsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo, err := repository.NewFileStorageRepository(cfg.FileStoragePath)
	if err != nil {
		log.Fatal(err)
	}

	svc := service.NewShortenerService(repo, cfg.BaseURL)
	svc.AttachDB(db)

	rtr, err := handler.NewRouter(svc)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Запускаем сервер по адресу %s\n", cfg.Address)

	if err := http.ListenAndServe(cfg.Address, rtr); err != nil {
		log.Fatal(err)
	}
}
