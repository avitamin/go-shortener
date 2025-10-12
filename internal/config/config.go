package config

import (
	"flag"
)

type Config struct {
	Address string
	BaseURL string
}

func New() *Config {
	addr := flag.String("a", "localhost:8080", "адрес сервера (например localhost:8080)")
	base := flag.String("b", "http://localhost:8080", "базовый URL (например http://localhost:8080)")

	return &Config{
		Address: *addr,
		BaseURL: *base,
	}
}
