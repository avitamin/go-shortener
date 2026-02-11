package logger

import (
	"go.uber.org/zap"
)

// New создает новый экземпляр zap.Logger в режиме разработки.
// Логгер настроен для вывода читаемых сообщений с цветным форматированием.
func New() (*zap.Logger, error) {
	logger, err := zap.NewDevelopment()

	if err != nil {
		return nil, err
	}

	return logger, nil
}
