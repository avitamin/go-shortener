package audit

import "github.com/avitamin/go-shortener/internal/config"

// NewServiceFromConfig создает сервис аудита и подключает наблюдателей из конфигурации.
func NewServiceFromConfig(cfg *config.Config) *Service {
	svc := NewService()

	if cfg == nil {
		return svc
	}

	if cfg.AuditFile != "" {
		svc.AddObserver(NewFileObserver(cfg.AuditFile))
	}

	if cfg.AuditURL != "" {
		svc.AddObserver(NewRemoteObserver(cfg.AuditURL))
	}

	return svc
}
