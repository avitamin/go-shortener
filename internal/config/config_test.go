package config_test

import (
	"os"
	"testing"

	"github.com/avitamin/go-shortener/internal/config"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name                string
		args                []string
		env                 map[string]string
		wantAddress         string
		wantBaseURL         string
		wantFileStoragePath string
	}{
		{
			name:                "defaults only",
			args:                []string{"cmd"},
			wantAddress:         "localhost:8080",
			wantBaseURL:         "http://localhost:8080",
			wantFileStoragePath: "./runtime/storage",
		},
		{
			name:                "flags override defaults",
			args:                []string{"cmd", "-a=127.0.0.1:9001", "-b=http://127.0.0.1:9001", "-f=./runtime/new-storage"},
			wantAddress:         "127.0.0.1:9001",
			wantBaseURL:         "http://127.0.0.1:9001",
			wantFileStoragePath: "./runtime/new-storage",
		},
		{
			name: "env override defaults",
			env: map[string]string{
				"SERVER_ADDRESS":    "0.0.0.0:9002",
				"BASE_URL":          "http://0.0.0.0:9002",
				"FILE_STORAGE_PATH": "./runtime/new-storage",
			},
			args:                []string{"cmd"},
			wantAddress:         "0.0.0.0:9002",
			wantBaseURL:         "http://0.0.0.0:9002",
			wantFileStoragePath: "./runtime/new-storage",
		},
		{
			name: "env overrides flags",
			env: map[string]string{
				"SERVER_ADDRESS": "0.0.0.0:9999",
			},
			args:                []string{"cmd", "-a=127.0.0.1:9003", "-b=http://127.0.0.1:9003"},
			wantAddress:         "0.0.0.0:9999",
			wantBaseURL:         "http://127.0.0.1:9003",
			wantFileStoragePath: "./runtime/storage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
