package config

import (
	"flag"
	"path/filepath"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	Address         string `env:"SERVER_ADDRESS"`
	BaseURL         string `env:"BASE_URL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DatabaseDsn     string `env:"DATABASE_DSN"`
}

var (
	addr        string
	base        string
	storage     string
	databaseDsn string
)

func init() {
	flag.StringVar(&addr, "a", "localhost:8080", "адрес сервера (например localhost:8080)")
	flag.StringVar(&base, "b", "http://localhost:8080", "базовый URL (например http://localhost:8080)")
	flag.StringVar(&storage, "f", "./runtime/storage", "путь к файлу хранилища (например ./runtime/storage)")
	flag.StringVar(&databaseDsn, "d", "postgres://postgres:postgres@db:5432/postgres?sslmode=disable", "путь к файлу хранилища (например postgres://postgres:postgres@db:5432/postgres)")
}

func New(withParse bool) (*Config, error) {

	if withParse {
		flag.Parse()
	}

	cfg := &Config{
		Address:         addr,
		BaseURL:         base,
		FileStoragePath: filepath.Clean(storage),
		DatabaseDsn:     databaseDsn,
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
