package utils

import (
	"fmt"
	"os"
)

func BuildDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		GetEnv("DB_USER", "postgres"),
		GetEnv("DB_PASSWORD", "postgres"),
		GetEnv("DB_HOST", "localhost"),
		GetEnv("DB_PORT", "5432"),
		GetEnv("DB_NAME", "go_message_app"),
		GetEnv("SSL_MODE", "disable"),
	)
}

func GetEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
