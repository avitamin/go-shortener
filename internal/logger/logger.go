package logger

import (
	"go.uber.org/zap"
)

func New() *zap.Logger {
	// Development — человекочитаемый формат, удобно для локальной разработки.
	// Для продакшена можно заменить на zap.NewProduction().
	logger, _ := zap.NewDevelopment()
	return logger
}
