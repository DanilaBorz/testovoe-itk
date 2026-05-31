package main

import (
	"go.uber.org/zap"

	"https://github.com/DanilaBorz/testovoe-itk/internal/config"
	"https://github.com/DanilaBorz/testovoe-itk/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		// Используем стандартный zap-логгер до загрузки конфигурации
		logger, _ := zap.NewProduction()
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// Инициализация логгера
	logLevel := zap.NewAtomicLevel()
	if err := logLevel.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		logLevel.SetLevel(zap.InfoLevel)
	}

	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level = logLevel
	logger, err := loggerConfig.Build()
	if err != nil {
		// Запасной вариант — стандартный логгер
		logger, _ = zap.NewProduction()
	}
	defer logger.Sync() //nolint:errcheck

	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Fatal("failed to create server", zap.Error(err))
	}

	if err := srv.Start(); err != nil {
		logger.Fatal("server error", zap.Error(err))
	}
}
