package config_test

import (
	"flag"
	"os"
	"testing"

	"github.com/avitamin/go-shortener/internal/config"
)

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flag.String("a", "localhost:8080", "адрес сервера")
	flag.String("b", "http://localhost:8080", "базовый URL")
}

func TestConfig(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		env         map[string]string
		wantAddress string
		wantBaseURL string
	}{
		{
			name:        "defaults only",
			args:        []string{"cmd"},
			wantAddress: "localhost:8080",
			wantBaseURL: "http://localhost:8080",
		},
		{
			name:        "flags override defaults",
			args:        []string{"cmd", "-a=127.0.0.1:9001", "-b=http://127.0.0.1:9001"},
			wantAddress: "127.0.0.1:9001",
			wantBaseURL: "http://127.0.0.1:9001",
		},
		{
			name: "env override defaults",
			env: map[string]string{
				"SERVER_ADDRESS": "0.0.0.0:9002",
				"BASE_URL":       "http://0.0.0.0:9002",
			},
			args:        []string{"cmd"},
			wantAddress: "0.0.0.0:9002",
			wantBaseURL: "http://0.0.0.0:9002",
		},
		{
			name: "env overrides flags",
			env: map[string]string{
				"SERVER_ADDRESS": "0.0.0.0:9999",
			},
			args:        []string{"cmd", "-a=127.0.0.1:9003", "-b=http://127.0.0.1:9003"},
			wantAddress: "0.0.0.0:9999",
			wantBaseURL: "http://127.0.0.1:9003",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags()
			os.Clearenv()

			// установить переменные окружения
			for k, v := range tt.env {
				os.Setenv(k, v)
			}
			defer os.Clearenv()

			os.Args = tt.args
			cfg := config.New(true)

			if cfg.Address != tt.wantAddress {
				t.Errorf("Address = %s, want %s", cfg.Address, tt.wantAddress)
			}
			if cfg.BaseURL != tt.wantBaseURL {
				t.Errorf("BaseURL = %s, want %s", cfg.BaseURL, tt.wantBaseURL)
			}
		})
	}
}
