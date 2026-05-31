package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config содержит всю конфигурацию приложения.
type Config struct {
	ServerPort  string
	DatabaseURL string
	DBMaxConns  int
	DBMinConns  int
	LogLevel    string
}

// Load читает конфигурацию из файла config.env и переменных окружения.
func Load() (*Config, error) {
	_ = godotenv.Load("config.env")

	cfg := &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),
		DBMaxConns: getEnvInt("DB_MAX_CONNS", 50),
		DBMinConns: getEnvInt("DB_MIN_CONNS", 10),
		LogLevel:   getEnv("LOG_LEVEL", "info"),
	}

	host := getEnv("DB_HOST", "")
	port := getEnv("DB_PORT", "")
	user := getEnv("DB_USER", "")
	password := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "")
	sslMode := getEnv("DB_SSLMODE", "disable")

	// Проверка обязательных полей базы данных
	var missing []string
	if host == "" {
		missing = append(missing, "DB_HOST")
	}
	if port == "" {
		missing = append(missing, "DB_PORT")
	}
	if user == "" {
		missing = append(missing, "DB_USER")
	}
	if password == "" {
		missing = append(missing, "DB_PASSWORD")
	}
	if dbName == "" {
		missing = append(missing, "DB_NAME")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s",
			strings.Join(missing, ", "))
	}

	cfg.DatabaseURL = fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbName, sslMode,
	)

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}