package config

import "os"

type Config struct {
	Address string
	BaseUrl string
}

func New() *Config {
	address := os.Getenv("SERVER_ADDRESS")

	if address == "" {
		address = "localhost:8099"
	}

	base := os.Getenv("BASE_URL")

	if base == "" {
		base = "http://localhost:8099"
	}

	return &Config{
		Address: address,
		BaseUrl: base,
	}
}
