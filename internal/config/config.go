package config

import (
	"flag"
)

type Config struct {
	Address string
	BaseURL string
}

var (
	addr string
	base string
)

func init() {
	flag.StringVar(&addr, "a", "localhost:8080", "адрес сервера (например localhost:8080)")
	flag.StringVar(&base, "b", "http://localhost:8080", "базовый URL (например http://localhost:8080)")
}

func New(withParse bool) *Config {

	if withParse {
		flag.Parse()
	}

	return &Config{
		Address: addr,
		BaseURL: base,
	}
}
