package config_test

import (
	"flag"
	"os"
	"path/filepath"
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
		wantError           string
		args                []string
		env                 map[string]string
		configJSON          string
		wantAddress         string
		wantEnableHTTPS     bool
		wantBaseURL         string
		wantFileStoragePath string
		wantDatabaseDsn     string
		wantSecretKey       string
		wantAuditFile       string
		wantAuditURL        string
		wantTrustedSubnet   string
	}{
		{
			name:                "defaults only",
			args:                []string{"cmd"},
			wantAddress:         "localhost:8080",
			wantEnableHTTPS:     false,
			wantBaseURL:         "http://localhost:8080",
			wantFileStoragePath: "./runtime/storage",
			wantDatabaseDsn:     "",
			wantSecretKey:       "secret_key",
			wantAuditFile:       "",
			wantAuditURL:        "",
		},
		{
			name: "defaults with secret key from env",
			args: []string{"cmd"},
			env: map[string]string{
				"SECRET_KEY": "secret_key",
			},
			wantAddress:         "localhost:8080",
			wantEnableHTTPS:     false,
			wantBaseURL:         "http://localhost:8080",
			wantFileStoragePath: "./runtime/storage",
			wantDatabaseDsn:     "",
			wantSecretKey:       "secret_key",
			wantAuditFile:       "",
			wantAuditURL:        "",
		},
		{
			name: "flags override defaults",
			args: []string{"cmd", "-a=127.0.0.1:9001", "-b=http://127.0.0.1:9001", "-f=./runtime/new-storage", "-d=postgres://postgres:postgres@db:5432/new_db", "-s=true"},
			env: map[string]string{
				"SECRET_KEY": "secret_key",
			},
			wantAddress:         "127.0.0.1:9001",
			wantEnableHTTPS:     true,
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
				"ENABLE_HTTPS":      "true",
				"BASE_URL":          "http://0.0.0.0:9002",
				"FILE_STORAGE_PATH": "./runtime/new-storage",
				"DATABASE_DSN":      "postgres://postgres:postgres@db:5432/env_db",
				"SECRET_KEY":        "secret_key",
			},
			args:                []string{"cmd"},
			wantAddress:         "0.0.0.0:9002",
			wantEnableHTTPS:     true,
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
				"ENABLE_HTTPS":      "false",
				"BASE_URL":          "http://127.0.0.1:9994",
				"FILE_STORAGE_PATH": "./runtime/env_storage",
				"DATABASE_DSN":      "postgres://postgres:postgres@db:5432/env_db",
				"SECRET_KEY":        "secret_key",
			},
			args:                []string{"cmd", "-a=127.0.0.1:9003", "-b=http://127.0.0.1:9003", "-f=./runtime/flag_storage", "-d=postgres://postgres:postgres@db:5432/flag_db", "-s=true"},
			wantAddress:         "0.0.0.0:9999",
			wantEnableHTTPS:     false,
			wantBaseURL:         "http://127.0.0.1:9994",
			wantFileStoragePath: "./runtime/env_storage",
			wantDatabaseDsn:     "postgres://postgres:postgres@db:5432/env_db",
			wantSecretKey:       "secret_key",
			wantAuditFile:       "",
			wantAuditURL:        "",
		},
		{
			name:       "config file applies when no env and flags",
			args:       []string{"cmd", "-c=config.json"},
			configJSON: `{"server_address":"127.0.0.1:7777","base_url":"http://127.0.0.1:7777","file_storage_path":"./runtime/from-file","database_dsn":"postgres://postgres:postgres@db:5432/from_file","enable_https":true}`,
			env: map[string]string{
				"SECRET_KEY": "secret_key",
			},
			wantAddress:         "127.0.0.1:7777",
			wantEnableHTTPS:     true,
			wantBaseURL:         "http://127.0.0.1:7777",
			wantFileStoragePath: "./runtime/from-file",
			wantDatabaseDsn:     "postgres://postgres:postgres@db:5432/from_file",
			wantSecretKey:       "secret_key",
			wantAuditFile:       "",
			wantAuditURL:        "",
		},
		{
			name:       "flags override config file",
			args:       []string{"cmd", "-c=config.json", "-a=127.0.0.1:9001", "-s=true"},
			configJSON: `{"server_address":"127.0.0.1:7777","base_url":"http://127.0.0.1:7777","file_storage_path":"./runtime/from-file","database_dsn":"postgres://postgres:postgres@db:5432/from_file","enable_https":false}`,
			env: map[string]string{
				"SECRET_KEY": "secret_key",
			},
			wantAddress:         "127.0.0.1:9001",
			wantEnableHTTPS:     true,
			wantBaseURL:         "http://127.0.0.1:7777",
			wantFileStoragePath: "./runtime/from-file",
			wantDatabaseDsn:     "postgres://postgres:postgres@db:5432/from_file",
			wantSecretKey:       "secret_key",
			wantAuditFile:       "",
			wantAuditURL:        "",
		},
		{
			name:       "env overrides config file and config env overrides config flag",
			args:       []string{"cmd", "-c=wrong.json"},
			configJSON: `{"server_address":"127.0.0.1:7777","base_url":"http://127.0.0.1:7777","file_storage_path":"./runtime/from-file","database_dsn":"postgres://postgres:postgres@db:5432/from_file","enable_https":false}`,
			env: map[string]string{
				"CONFIG":            "config.json",
				"SERVER_ADDRESS":    "0.0.0.0:9002",
				"ENABLE_HTTPS":      "true",
				"BASE_URL":          "http://0.0.0.0:9002",
				"FILE_STORAGE_PATH": "./runtime/new-storage",
				"DATABASE_DSN":      "postgres://postgres:postgres@db:5432/env_db",
				"SECRET_KEY":        "secret_key",
			},
			wantAddress:         "0.0.0.0:9002",
			wantEnableHTTPS:     true,
			wantBaseURL:         "http://0.0.0.0:9002",
			wantFileStoragePath: "./runtime/new-storage",
			wantDatabaseDsn:     "postgres://postgres:postgres@db:5432/env_db",
			wantSecretKey:       "secret_key",
			wantAuditFile:       "",
			wantAuditURL:        "",
		},
		{
			name:      "missing config file returns error",
			args:      []string{"cmd", "-c=missing.json"},
			wantError: "reading config file",
			env: map[string]string{
				"SECRET_KEY": "secret_key",
			},
		},
		{
			name:       "invalid config file returns error",
			args:       []string{"cmd", "-config=config.json"},
			configJSON: `{invalid json}`,
			wantError:  "parsing config file",
			env: map[string]string{
				"SECRET_KEY": "secret_key",
			},
		},
		{
			name: "audit flags set",
			args: []string{"cmd", "--audit-file=/var/log/audit.log", "--audit-url=http://audit-server:8080"},
			env: map[string]string{
				"SECRET_KEY": "secret_key",
			},
			wantAddress:         "localhost:8080",
			wantEnableHTTPS:     false,
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
			wantEnableHTTPS:     false,
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
			wantEnableHTTPS:     false,
			wantBaseURL:         "http://localhost:8080",
			wantFileStoragePath: "./runtime/storage",
			wantDatabaseDsn:     "",
			wantSecretKey:       "secret_key",
			wantAuditFile:       "/var/log/env_audit.log",
			wantAuditURL:        "http://env-audit-server:8080",
		},
		{
			name: "trusted subnet from flag",
			args: []string{"cmd", "-t=192.168.0.0/24"},
			env: map[string]string{
				"SECRET_KEY": "secret_key",
			},
			wantAddress:         "localhost:8080",
			wantEnableHTTPS:     false,
			wantBaseURL:         "http://localhost:8080",
			wantFileStoragePath: "./runtime/storage",
			wantDatabaseDsn:     "",
			wantSecretKey:       "secret_key",
			wantAuditFile:       "",
			wantAuditURL:        "",
			wantTrustedSubnet:   "192.168.0.0/24",
		},
		{
			name: "trusted subnet env overrides flag",
			args: []string{"cmd", "-t=192.168.0.0/24"},
			env: map[string]string{
				"SECRET_KEY":     "secret_key",
				"TRUSTED_SUBNET": "10.0.0.0/8",
			},
			wantAddress:         "localhost:8080",
			wantEnableHTTPS:     false,
			wantBaseURL:         "http://localhost:8080",
			wantFileStoragePath: "./runtime/storage",
			wantDatabaseDsn:     "",
			wantSecretKey:       "secret_key",
			wantAuditFile:       "",
			wantAuditURL:        "",
			wantTrustedSubnet:   "10.0.0.0/8",
		},
		{
			name:       "trusted subnet from config file",
			args:       []string{"cmd", "-c=config.json"},
			configJSON: `{"trusted_subnet":"172.16.0.0/12"}`,
			env: map[string]string{
				"SECRET_KEY": "secret_key",
			},
			wantAddress:         "localhost:8080",
			wantEnableHTTPS:     false,
			wantBaseURL:         "http://localhost:8080",
			wantFileStoragePath: "./runtime/storage",
			wantDatabaseDsn:     "",
			wantSecretKey:       "secret_key",
			wantAuditFile:       "",
			wantAuditURL:        "",
			wantTrustedSubnet:   "172.16.0.0/12",
		},
		{
			name:      "invalid trusted subnet returns error",
			args:      []string{"cmd", "-t=invalid"},
			wantError: "invalid trusted subnet",
			env: map[string]string{
				"SECRET_KEY": "secret_key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			defer os.Clearenv()

			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			if tt.configJSON != "" {
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, "config.json")
				err := os.WriteFile(configPath, []byte(tt.configJSON), 0o600)
				assert.NoError(t, err)

				for i, arg := range tt.args {
					if arg == "-c=config.json" || arg == "-config=config.json" {
						tt.args[i] = tt.args[i][:len(tt.args[i])-len("config.json")] + configPath
					}
					if arg == "-c=wrong.json" {
						tt.args[i] = tt.args[i][:len(tt.args[i])-len("wrong.json")] + filepath.Join(tempDir, "wrong.json")
					}
				}

				if configEnvPath, ok := tt.env["CONFIG"]; ok {
					if configEnvPath == "config.json" {
						os.Setenv("CONFIG", configPath)
					}
				}
			}

			os.Args = tt.args
			defer ClearFlags()

			cfg, err := config.New(true)
			if tt.wantError != "" {
				assert.Error(t, err)
				assert.ErrorContains(t, err, tt.wantError)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantAddress, cfg.Address)
			assert.Equal(t, tt.wantEnableHTTPS, cfg.EnableHTTPS)
			assert.Equal(t, tt.wantBaseURL, cfg.BaseURL)
			assert.Equal(t, tt.wantFileStoragePath, cfg.FileStoragePath)
			assert.Equal(t, tt.wantDatabaseDsn, cfg.DatabaseDsn)
			assert.Equal(t, tt.wantSecretKey, cfg.SecretKey)
			assert.Equal(t, tt.wantAuditFile, cfg.AuditFile)
			assert.Equal(t, tt.wantAuditURL, cfg.AuditURL)
			assert.Equal(t, tt.wantTrustedSubnet, cfg.TrustedSubnet)
		})
	}
}
