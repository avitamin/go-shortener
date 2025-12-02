package config

import (
	"flag"
	"sync"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	Address         string `env:"SERVER_ADDRESS"`
	BaseURL         string `env:"BASE_URL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DatabaseDsn     string `env:"DATABASE_DSN"`
	SecretKey       string `env:"SECRET_KEY"`
}

func Set(c *Config) {
	once.Do(func() {
		cfg = c
	})
}

func Get() *Config {
	return cfg
}

var (
	addr        string
	base        string
	filePath    string
	databaseDsn string

	cfg  *Config
	once sync.Once
)

func init() {
	flag.StringVar(&addr, "a", "localhost:8080", "адрес сервера (например localhost:8080)")
	flag.StringVar(&base, "b", "http://localhost:8080", "базовый URL (например http://localhost:8080)")
	flag.StringVar(&filePath, "f", "./runtime/storage", "путь к файлу хранилища (например ./runtime/storage)")
	flag.StringVar(&databaseDsn, "d", "", "DSN (например postgres://postgres:postgres@db:5432/postgres)")
}

func New(withParse bool) (*Config, error) {

	if withParse {
		flag.Parse()
	}

	cfg := &Config{
		Address:         addr,
		BaseURL:         base,
		FileStoragePath: filePath,
		DatabaseDsn:     databaseDsn,
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	// если секретный ключ не задан, генерируем его
	if cfg.SecretKey == "" {
		cfg.SecretKey = "default_secret_key"
	}

	return cfg, nil
}
