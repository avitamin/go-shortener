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
		wantDatabaseDsn     string
	}{
		{
			name:                "defaults only",
			args:                []string{"cmd"},
			wantAddress:         "localhost:8080",
			wantBaseURL:         "http://localhost:8080",
			wantFileStoragePath: "./runtime/storage",
			wantDatabaseDsn:     "postgres://postgres:postgres@db:5432/postgres?sslmode=disable",
		},
		{
			name:                "flags override defaults",
			args:                []string{"cmd", "-a=127.0.0.1:9001", "-b=http://127.0.0.1:9001", "-f=./runtime/new-storage", "-d=postgres://postgres:postgres@db:5432/new_db"},
			wantAddress:         "127.0.0.1:9001",
			wantBaseURL:         "http://127.0.0.1:9001",
			wantFileStoragePath: "./runtime/new-storage",
			wantDatabaseDsn:     "postgres://postgres:postgres@db:5432/new_db",
		},
		{
			name: "env override defaults",
			env: map[string]string{
				"SERVER_ADDRESS":    "0.0.0.0:9002",
				"BASE_URL":          "http://0.0.0.0:9002",
				"FILE_STORAGE_PATH": "./runtime/new-storage",
				"DATABASE_DSN":      "postgres://postgres:postgres@db:5432/env_db",
			},
			args:                []string{"cmd"},
			wantAddress:         "0.0.0.0:9002",
			wantBaseURL:         "http://0.0.0.0:9002",
			wantFileStoragePath: "./runtime/new-storage",
			wantDatabaseDsn:     "postgres://postgres:postgres@db:5432/env_db",
		},
		{
			name: "env overrides flags",
			env: map[string]string{
				"SERVER_ADDRESS":    "0.0.0.0:9999",
				"BASE_URL":          "http://127.0.0.1:9994",
				"FILE_STORAGE_PATH": "./runtime/env_storage",
				"DATABASE_DSN":      "postgres://postgres:postgres@db:5432/env_db",
			},
			args:                []string{"cmd", "-a=127.0.0.1:9003", "-b=http://127.0.0.1:9003", "-f=./runtime/flag_storage", "-d=postgres://postgres:postgres@db:5432/flag_db"},
			wantAddress:         "0.0.0.0:9999",
			wantBaseURL:         "http://127.0.0.1:9994",
			wantFileStoragePath: "./runtime/env_storage",
			wantDatabaseDsn:     "postgres://postgres:postgres@db:5432/env_db",
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

			cfg, err := config.New(true)
			if err != nil {
				t.Fatal(err)
			}

			if cfg.Address != tt.wantAddress {
				t.Errorf("Address = %s, want %s", cfg.Address, tt.wantAddress)
			}
			if cfg.BaseURL != tt.wantBaseURL {
				t.Errorf("BaseURL = %s, want %s", cfg.BaseURL, tt.wantBaseURL)
			}
		})
	}
}
