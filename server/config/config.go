package config

import (
	"log"
	"os"
)

type Config struct {
	DatabaseURL string
	Port        string
	JWTSecret   string
	Environment string
}

func Load() *Config {
	cfg := &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/frogs_cafe?sslmode=disable"),
		Port:        getEnv("PORT", "8080"),
		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}

	log.Printf("Configuration loaded - running in %s mode", cfg.Environment)

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
