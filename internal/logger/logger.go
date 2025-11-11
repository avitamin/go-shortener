package logger

import (
	"go.uber.org/zap"
)

func New() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}
