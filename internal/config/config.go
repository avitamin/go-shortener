package config

import "os"

type Config struct {
	Address string
	BaseURL string
}

func New() *Config {
	address := os.Getenv("SERVER_ADDRESS")

	if address == "" {
		address = "localhost:8080"
	}

	base := os.Getenv("BASE_URL")

	if base == "" {
		base = "http://localhost:8080"
	}

	return &Config{
		Address: address,
		BaseURL: base,
	}
}
