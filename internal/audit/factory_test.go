package audit

import (
	"path/filepath"
	"testing"

	"github.com/avitamin/go-shortener/internal/config"
)

func TestNewServiceFromConfig(t *testing.T) {
	nilCfgSvc := NewServiceFromConfig(nil)
	defer nilCfgSvc.Close()
	if len(nilCfgSvc.observers) != 0 {
		t.Fatalf("expected no observers for nil cfg, got %d", len(nilCfgSvc.observers))
	}

	cfg := &config.Config{
		AuditFile: filepath.Join(t.TempDir(), "audit.log"),
		AuditURL:  "http://example.com/audit",
	}
	svc := NewServiceFromConfig(cfg)
	defer svc.Close()

	if len(svc.observers) != 2 {
		t.Fatalf("expected 2 observers, got %d", len(svc.observers))
	}
}
