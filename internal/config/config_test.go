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
		wantAuditFile       string
		wantAuditURL        string
	}{
		{
			name:                "defaults only",
			args:                []string{"cmd"},
			wantAddress:         "localhost:8080",
			wantBaseURL:         "http://localhost:8080",
			wantFileStoragePath: "./runtime/storage",
			wantDatabaseDsn:     "",
			wantSecretKey:       "secret_key",
			wantAuditFile:       "",
			wantAuditURL:        "",
		}, {
			name: "defaults with secret key from env",
			args: []string{"cmd"},
			env: map[string]string{
				"SECRET_KEY": "secret_key",
			},
			wantAddress:         "localhost:8080",
			wantBaseURL:         "http://localhost:8080",
			wantFileStoragePath: "./runtime/storage",
			wantDatabaseDsn:     "",
			wantSecretKey:       "secret_key",
			wantAuditFile:       "",
			wantAuditURL:        "",
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
			wantAuditFile:       "",
			wantAuditURL:        "",
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
			wantAuditFile:       "",
			wantAuditURL:        "",
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
			wantAuditFile:       "",
			wantAuditURL:        "",
		},
		{
			name: "audit flags set",
			args: []string{"cmd", "--audit-file=/var/log/audit.log", "--audit-url=http://audit-server:8080"},
			env: map[string]string{
				"SECRET_KEY": "secret_key",
			},
			wantAddress:         "localhost:8080",
			wantBaseURL:         "http://localhost:8080",
			wantFileStoragePath: "./runtime/storage",
			wantDatabaseDsn:     "",
			wantSecretKey:       "secret_key",
			wantAuditFile:       "/var/log/audit.log",
			wantAuditURL:        "http://audit-server:8080",
		},
		{
			name: "audit env set",
			args: []string{"cmd"},
			env: map[string]string{
				"SECRET_KEY": "secret_key",
				"AUDIT_FILE": "/var/log/audit_env.log",
				"AUDIT_URL":  "http://audit-env-server:8080",
			},
			wantAddress:         "localhost:8080",
			wantBaseURL:         "http://localhost:8080",
			wantFileStoragePath: "./runtime/storage",
			wantDatabaseDsn:     "",
			wantSecretKey:       "secret_key",
			wantAuditFile:       "/var/log/audit_env.log",
			wantAuditURL:        "http://audit-env-server:8080",
		},
		{
			name: "audit env overrides flags",
			args: []string{"cmd", "--audit-file=/var/log/flag_audit.log", "--audit-url=http://flag-audit-server:8080"},
			env: map[string]string{
				"SECRET_KEY": "secret_key",
				"AUDIT_FILE": "/var/log/env_audit.log",
				"AUDIT_URL":  "http://env-audit-server:8080",
			},
			wantAddress:         "localhost:8080",
			wantBaseURL:         "http://localhost:8080",
			wantFileStoragePath: "./runtime/storage",
			wantDatabaseDsn:     "",
			wantSecretKey:       "secret_key",
			wantAuditFile:       "/var/log/env_audit.log",
			wantAuditURL:        "http://env-audit-server:8080",
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
			if tt.wantError != nil {
				assert.ErrorIs(t, err, tt.wantError)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantAddress, cfg.Address)
			assert.Equal(t, tt.wantBaseURL, cfg.BaseURL)
			assert.Equal(t, tt.wantFileStoragePath, cfg.FileStoragePath)
			assert.Equal(t, tt.wantDatabaseDsn, cfg.DatabaseDsn)
			assert.Equal(t, tt.wantSecretKey, cfg.SecretKey)
			assert.Equal(t, tt.wantAuditFile, cfg.AuditFile)
			assert.Equal(t, tt.wantAuditURL, cfg.AuditURL)
		})
	}
}
