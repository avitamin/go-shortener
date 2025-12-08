package config

import (
	"errors"
	"flag"
	"fmt"
	"net/url"

	"github.com/caarlos0/env/v6"
)

var ErrSecretKeyReq error = errors.New("secret key required")

const DefaultAddress = "localhost:8080"
const DefaultBaseURL = "http://localhost:8080"
const DefaultFileStoragePath = "./runtime/storage"

type Config struct {
	Address         string `env:"SERVER_ADDRESS"`
	BaseURL         string `env:"BASE_URL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DatabaseDsn     string `env:"DATABASE_DSN"`
	SecretKey       string `env:"SECRET_KEY"`
}

func New(withParse bool) (*Config, error) {
	var cfg Config

	flag.StringVar(&cfg.Address, "a", DefaultAddress, "адрес сервера (например localhost:8080)")
	flag.StringVar(&cfg.BaseURL, "b", DefaultBaseURL, "базовый URL (например http://localhost:8080)")
	flag.StringVar(&cfg.FileStoragePath, "f", DefaultFileStoragePath, "путь к файлу хранилища (например ./runtime/storage)")
	flag.StringVar(&cfg.DatabaseDsn, "d", "", "DSN (например postgres://postgres:postgres@db:5432/postgres)")

	if withParse {
		flag.Parse()
	}

	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) ParseFlags() {

	flag.Parse()
}

func (c *Config) Validate() error {
	if c.SecretKey == "" {
		return ErrSecretKeyReq
	}

	if _, err := url.ParseRequestURI(c.BaseURL); err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	return nil
}
