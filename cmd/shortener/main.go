package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	svc.Audit.Close()
}

func valueOrNA(value string) string {
	if value == "" {
		return "N/A"
	}

	return value
}

func launchServer(server *http.Server, enableHTTPS bool) error {
	if !enableHTTPS {
		return server.ListenAndServe()
	}

	cert, err := generateSelfSignedCertificate(server.Addr)
	if err != nil {
		return fmt.Errorf("generating TLS certificate: %w", err)
	}

	listener, err := tls.Listen("tcp", server.Addr, &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	})
	if err != nil {
		return fmt.Errorf("creating TLS listener: %w", err)
	}

	log.Printf("HTTPS включен для адреса %s\n", server.Addr)
	return server.Serve(listener)
}

func generateSelfSignedCertificate(addr string) (tls.Certificate, error) {
	hosts := []string{"localhost"}

	host, _, err := net.SplitHostPort(addr)
	if err == nil && host != "" {
		hosts = append(hosts, strings.Trim(host, "[]"))
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "go-shortener",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
			continue
		}

		template.DNSNames = append(template.DNSNames, h)
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	return tls.X509KeyPair(certPEM, keyPEM)
}
