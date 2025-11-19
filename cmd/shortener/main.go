package main

import (
	"database/sql"
	"log"
	"net/http"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

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
	var repo repository.Repository

	if cfg.DatabaseDsn != "" {
		db, err := sql.Open("pgx", cfg.DatabaseDsn)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		driver, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			log.Fatal(err)
		}
		m, err := migrate.NewWithDatabaseInstance(
			"file://migrations",
			"postgres", driver)
		if err != nil {
			log.Fatal(err)
		}
		m.Up()

		repo, err = repository.NewDataBaseRepository(db)
		if err != nil {
			log.Fatal(err)
		}

	} else if cfg.FileStoragePath != "" {
		repo, err = repository.NewFileStorageRepository(cfg.FileStoragePath)
		if err != nil {
			log.Fatal(err)
		}

	} else {
		repo = repository.NewInMemoryStorage()
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
