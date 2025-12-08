package config_test

import (
	"flag"
	"os"
	"testing"

	"github.com/avitamin/go-shortener/internal/config"
	"github.com/stretchr/testify/assert"
)

// clear flags for testing purposes
func ClearFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func TestConfig(t *testing.T) {
	tests := []struct {
		name                string
		wantError           error
		args                []string
		env                 map[string]string
		wantAddress         string
		wantBaseURL         string
		wantFileStoragePath string
		wantDatabaseDsn     string
		wantSecretKey       string
	}{
		{
			name:                "defaults only",
			wantError:           config.ErrSecretKeyReq,
			args:                []string{"cmd"},
			wantAddress:         "localhost:8080",
			wantBaseURL:         "http://localhost:8080",
			wantFileStoragePath: "./runtime/storage",
			wantDatabaseDsn:     "postgres://postgres:postgres@db:5432/postgres?sslmode=disable",
		}, {
			name: "defaults with secret key from env",
			args: []string{"cmd"},
			env: map[string]string{
				"SECRET_KEY": "secret_key",
			},
			wantAddress:         "localhost:8080",
			wantBaseURL:         "http://localhost:8080",
			wantFileStoragePath: "./runtime/storage",
			wantDatabaseDsn:     "postgres://postgres:postgres@db:5432/postgres?sslmode=disable",
			wantSecretKey:       "secret_key",
		},
		{
			name: "flags override defaults",
			args: []string{"cmd", "-a=127.0.0.1:9001", "-b=http://127.0.0.1:9001", "-f=./runtime/new-storage", "-d=postgres://postgres:postgres@db:5432/new_db"},
			env: map[string]string{
				"SECRET_KEY": "secret_key",
			},
			wantAddress:         "127.0.0.1:9001",
			wantBaseURL:         "http://127.0.0.1:9001",
			wantFileStoragePath: "./runtime/new-storage",
			wantDatabaseDsn:     "postgres://postgres:postgres@db:5432/new_db",
			wantSecretKey:       "secret_key",
		},
		{
			name: "env override defaults",
			env: map[string]string{
				"SERVER_ADDRESS":    "0.0.0.0:9002",
				"BASE_URL":          "http://0.0.0.0:9002",
				"FILE_STORAGE_PATH": "./runtime/new-storage",
				"DATABASE_DSN":      "postgres://postgres:postgres@db:5432/env_db",
				"SECRET_KEY":        "secret_key",
			},
			args:                []string{"cmd"},
			wantAddress:         "0.0.0.0:9002",
			wantBaseURL:         "http://0.0.0.0:9002",
			wantFileStoragePath: "./runtime/new-storage",
			wantDatabaseDsn:     "postgres://postgres:postgres@db:5432/env_db",
			wantSecretKey:       "secret_key",
		},
		{
			name: "env overrides flags",
			env: map[string]string{
				"SERVER_ADDRESS":    "0.0.0.0:9999",
				"BASE_URL":          "http://127.0.0.1:9994",
				"FILE_STORAGE_PATH": "./runtime/env_storage",
				"DATABASE_DSN":      "postgres://postgres:postgres@db:5432/env_db",
				"SECRET_KEY":        "secret_key",
			},
			args:                []string{"cmd", "-a=127.0.0.1:9003", "-b=http://127.0.0.1:9003", "-f=./runtime/flag_storage", "-d=postgres://postgres:postgres@db:5432/flag_db"},
			wantAddress:         "0.0.0.0:9999",
			wantBaseURL:         "http://127.0.0.1:9994",
			wantFileStoragePath: "./runtime/env_storage",
			wantDatabaseDsn:     "postgres://postgres:postgres@db:5432/env_db",
			wantSecretKey:       "secret_key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// установить переменные окружения
			for k, v := range tt.env {
				os.Setenv(k, v)
			}
			defer os.Clearenv()

			os.Args = tt.args
			defer ClearFlags()

			cfg, err := config.New(true)
			if err != nil {
				if tt.wantError != nil {
					assert.ErrorIs(t, err, tt.wantError)
					return
				} else {
					t.Fatal(err)
				}
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
