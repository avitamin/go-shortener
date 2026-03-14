package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/avitamin/go-shortener/internal/audit"
	"github.com/avitamin/go-shortener/internal/config"
	grpcserver "github.com/avitamin/go-shortener/internal/grpc"
	pb "github.com/avitamin/go-shortener/internal/grpc/pb"
	"github.com/avitamin/go-shortener/internal/handler"
	"github.com/avitamin/go-shortener/internal/repository"
	"github.com/avitamin/go-shortener/internal/service"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc"
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

	repo := initRepository(cfg)
	defer repo.Close()

	svc := service.NewShortenerService(repo, cfg, audit.NewServiceFromConfig(cfg))

	rtr, err := handler.NewRouter(svc)
	if err != nil {
		log.Fatalf("router creating error: %v", err)
	}

	httpServer := &http.Server{
		Addr:    cfg.Address,
		Handler: rtr,
	}

	grpcTransport := grpcserver.NewServer(svc)
	grpcSrv := grpc.NewServer(grpc.UnaryInterceptor(grpcserver.UnaryAuthInterceptor(cfg.SecretKey)))
	pb.RegisterShortenerServiceServer(grpcSrv, grpcTransport)

	log.Printf("Запускаем HTTP сервер по адресу %s\n", cfg.Address)
	log.Printf("Запускаем gRPC сервер по адресу %s\n", cfg.GRPCAddress)

	go func() {
		if err := launchServer(httpServer, cfg.EnableHTTPS); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server launching error: %v", err)
		}
	}()

	go func() {
		if err := launchGRPCServer(grpcSrv, cfg.GRPCAddress, cfg.EnableHTTPS); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Fatalf("gRPC server launching error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-stop

	httpShutdownCtx, httpCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer httpCancel()

	if err := httpServer.Shutdown(httpShutdownCtx); err != nil {
		log.Printf("http server shutdown error: %v", err)
	}

	shutdownGRPCServer(grpcSrv, 10*time.Second)

	serviceShutdownCtx, serviceCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer serviceCancel()

	if err := svc.Shutdown(serviceShutdownCtx); err != nil {
		log.Printf("service shutdown error: %v", err)
	}
}

func initRepository(cfg *config.Config) repository.Repository {
	if cfg.DatabaseDsn != "" {
		db, err := sql.Open("pgx", cfg.DatabaseDsn)
		if err != nil {
			log.Fatalf("database connection opening error: %v", err)
		}

		driver, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			_ = db.Close()
			log.Fatalf("database driver creating error: %v", err)
		}

		m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
		if err != nil {
			_ = db.Close()
			log.Fatalf("migrations applying error: %v", err)
		}
		_ = m.Up()

		repo, err := repository.NewDataBaseRepository(db)
		if err != nil {
			_ = db.Close()
			log.Fatalf("repository creating error: %v", err)
		}

		return repo
	}

	if cfg.FileStoragePath != "" {
		repo, err := repository.NewFileStorageRepository(cfg.FileStoragePath)
		if err != nil {
			log.Fatalf("repository creating error: %v", err)
		}

		return repo
	}

	return repository.NewInMemoryStorage()
}

func shutdownGRPCServer(server *grpc.Server, timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(done)
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-done:
	case <-timer.C:
		server.Stop()
	}
}

func valueOrNA(value string) string {
	if value == "" {
		return "N/A"
	}

	return value
}
