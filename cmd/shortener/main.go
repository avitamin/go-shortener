package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/avitamin/go-shortener/internal/audit"
	"github.com/avitamin/go-shortener/internal/config"
	"github.com/avitamin/go-shortener/internal/handler"
	"github.com/avitamin/go-shortener/internal/repository"
	"github.com/avitamin/go-shortener/internal/service"
)

var buildVersion string
var buildDate string
var buildCommit string

func main() {
	fmt.Printf("Build version: %s\n", valueOrNA(buildVersion))
	fmt.Printf("Build date: %s\n", valueOrNA(buildDate))
	fmt.Printf("Build commit: %s\n", valueOrNA(buildCommit))

	cfg, err := config.New(true)
	if err != nil {
		log.Fatalf("configuration creating error: %v", err)
	}
	var repo repository.Repository

	if cfg.DatabaseDsn != "" {
		db, err := sql.Open("pgx", cfg.DatabaseDsn)
		if err != nil {
			log.Fatalf("database connection opening error: %v", err)
		}
		defer db.Close()

		driver, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			log.Fatalf("database driver creating error: %v", err)
		}
		m, err := migrate.NewWithDatabaseInstance(
			"file://migrations",
			"postgres", driver)
		if err != nil {
			log.Fatalf("migrations applying error: %v", err)
		}
		m.Up()

		repo, err = repository.NewDataBaseRepository(db)
		if err != nil {
			log.Fatalf("repository creating error: %v", err)
		}

	} else if cfg.FileStoragePath != "" {
		repo, err = repository.NewFileStorageRepository(cfg.FileStoragePath)
		if err != nil {
			log.Fatalf("repository creating error: %v", err)
		}

	} else {
		repo = repository.NewInMemoryStorage()
	}
	defer repo.Close()

	svc := service.NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))

	rtr, err := handler.NewRouter(svc)
	if err != nil {
		log.Fatalf("router creating error: %v", err)
	}

	log.Printf("Запускаем сервер по адресу %s\n", cfg.Address)

	server := &http.Server{
		Addr:    cfg.Address,
		Handler: rtr,
	}

	go func() {
		if err := launchServer(server, cfg.EnableHTTPS); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server launching error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	serviceShutdownCtx, serviceCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer serviceCancel()

	if err := svc.Shutdown(serviceShutdownCtx); err != nil {
		log.Printf("service shutdown error: %v", err)
	}
}

func valueOrNA(value string) string {
	if value == "" {
		return "N/A"
	}

	return value
}
