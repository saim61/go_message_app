package config

import (
	"fmt"
	"log"
	"os"
)

type Config struct {
	DSN          string
	KafkaBrokers []string
	JWTSecret    string
	Port         string
}

func Load() Config {
	get := func(key, dflt string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return dflt
	}

	dbUser := get("DB_USER", "postgres")
	dbPass := get("DB_PASSWORD", "postgres")
	dbHost := get("DB_HOST", "localhost")
	dbPort := get("DB_PORT", "5432")
	dbName := get("DB_NAME", "go_message_app")
	sslMode := get("SSL_MODE", "disable")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPass, dbHost, dbPort, dbName, sslMode)

	cfg := Config{
		DSN:          dsn,
		KafkaBrokers: []string{get("KAFKA_BROKERS", "localhost:9092")},
		JWTSecret:    get("JWT_SECRET", "dev_only_secret"),
		Port:         get("PORT", "8080"),
	}
	log.Printf("[config] %+v\n", cfg)
	return cfg
}
