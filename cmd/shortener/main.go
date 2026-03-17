package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/avitamin/go-shortener/internal/audit"
	"github.com/avitamin/go-shortener/internal/config"
	grpcserver "github.com/avitamin/go-shortener/internal/grpc"
	pb "github.com/avitamin/go-shortener/internal/grpc/pb"
	"github.com/avitamin/go-shortener/internal/handler"
	projectlogger "github.com/avitamin/go-shortener/internal/logger"
	"github.com/avitamin/go-shortener/internal/repository"
	"github.com/avitamin/go-shortener/internal/service"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var buildVersion string
var buildDate string
var buildCommit string

const gracefulShutdownTimeout = 30 * time.Second

func main() {
	appLogger, err := projectlogger.New()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = appLogger.Sync()
	}()

	appLogger.Info("build info")
	appLogger.Info("version", zap.String("value", valueOrNA(buildVersion)))
	appLogger.Info("date", zap.String("value", valueOrNA(buildDate)))
	appLogger.Info("commit", zap.String("value", valueOrNA(buildCommit)))

	cfg, err := config.New(true)
	if err != nil {
		appLogger.Fatal("configuration creating error", zap.Error(err))
	}

	repo := initRepository(cfg, appLogger)
	defer repo.Close()

	svc := service.NewShortenerServiceWithLogger(repo, cfg, audit.NewServiceFromConfig(cfg), appLogger)

	rtr, err := handler.NewRouter(svc)
	if err != nil {
		appLogger.Fatal("router creating error", zap.Error(err))
	}

	httpServer := &http.Server{
		Addr:    cfg.Address,
		Handler: rtr,
	}

	grpcTransport := grpcserver.NewServer(svc)
	grpcSrv := grpc.NewServer(grpc.UnaryInterceptor(grpcserver.UnaryAuthInterceptor(cfg.SecretKey)))
	pb.RegisterShortenerServiceServer(grpcSrv, grpcTransport)

	appLogger.Info("starting HTTP server", zap.String("address", cfg.Address))
	appLogger.Info("starting gRPC server", zap.String("address", cfg.GRPCAddress))

	go func() {
		if err := launchServer(httpServer, cfg.EnableHTTPS); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLogger.Fatal("http server launching error", zap.Error(err))
		}
	}()

	go func() {
		if err := launchGRPCServer(grpcSrv, cfg.GRPCAddress, cfg.EnableHTTPS); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			appLogger.Fatal("gRPC server launching error", zap.Error(err))
		}
	}()

	baseShutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	<-baseShutdownCtx.Done()
	stop()
	shutdownStartedAt := time.Now()
	appLogger.Info("shutdown signal received, starting graceful shutdown")

	httpShutdownCtx, httpCancel := context.WithTimeout(context.WithoutCancel(baseShutdownCtx), gracefulShutdownTimeout)
	defer httpCancel()

	appLogger.Info("shutting down HTTP server", zap.Duration("timeout", gracefulShutdownTimeout))
	if err := httpServer.Shutdown(httpShutdownCtx); err != nil {
		appLogger.Error("http server shutdown error", zap.Error(err))
	} else {
		appLogger.Info("HTTP server shutdown completed")
	}

	appLogger.Info("shutting down gRPC server", zap.Duration("timeout", gracefulShutdownTimeout))
	shutdownGRPCServer(grpcSrv, gracefulShutdownTimeout)
	appLogger.Info("gRPC server shutdown completed")

	serviceShutdownCtx, serviceCancel := context.WithTimeout(context.WithoutCancel(baseShutdownCtx), gracefulShutdownTimeout)
	defer serviceCancel()

	appLogger.Info("shutting down service background workers", zap.Duration("timeout", gracefulShutdownTimeout))
	if err := svc.Shutdown(serviceShutdownCtx); err != nil {
		appLogger.Error("service shutdown error", zap.Error(err))
	} else {
		appLogger.Info("service shutdown completed")
	}

	appLogger.Info("graceful shutdown completed", zap.Duration("duration", time.Since(shutdownStartedAt)))
}

func initRepository(cfg *config.Config, appLogger *zap.Logger) repository.Repository {
	if cfg.DatabaseDsn != "" {
		db, err := sql.Open("pgx", cfg.DatabaseDsn)
		if err != nil {
			appLogger.Fatal("database connection opening error", zap.Error(err))
		}

		driver, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			_ = db.Close()
			appLogger.Fatal("database driver creating error", zap.Error(err))
		}

		m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
		if err != nil {
			_ = db.Close()
			appLogger.Fatal("migrations applying error", zap.Error(err))
		}
		_ = m.Up()

		repo, err := repository.NewDataBaseRepository(db)
		if err != nil {
			_ = db.Close()
			appLogger.Fatal("repository creating error", zap.Error(err))
		}

		return repo
	}

	if cfg.FileStoragePath != "" {
		repo, err := repository.NewFileStorageRepository(cfg.FileStoragePath)
		if err != nil {
			appLogger.Fatal("repository creating error", zap.Error(err))
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
