package config

import (
	"os"
)

type Config struct {
	DatabaseURL string
	Port        string
	JWTSecret   string
	Environment string
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/frogs_cafe?sslmode=disable"),
		Port:        getEnv("PORT", "8080"),
		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
