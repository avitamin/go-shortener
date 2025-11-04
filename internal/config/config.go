package config

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	Address         string `env:"SERVER_ADDRESS"`
	BaseURL         string `env:"BASE_URL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
}

var (
	addr    string
	base    string
	storage string
)

func init() {
	flag.StringVar(&addr, "a", "localhost:8080", "адрес сервера (например localhost:8080)")
	flag.StringVar(&base, "b", "http://localhost:8080", "базовый URL (например http://localhost:8080)")
	flag.StringVar(&storage, "f", "./runtime/storage", "путь к файлу хранилища (например ./runtime/storage)")
}

func New(withParse bool) *Config {

	if withParse {
		flag.Parse()
	}

	cfg := &Config{
		Address:         addr,
		BaseURL:         base,
		FileStoragePath: filepath.Clean(storage),
	}

	if err := env.Parse(cfg); err != nil {
		log.Fatal(err)
	}

	return cfg
}
