package config

import (
	"log"
	"os"
)

type Config struct {
	DatabaseURL    string
	Port           string
	JWTSecret      string
	Environment    string
	BaseURL        string // e.g. https://frogs.cafe — used for actor URIs and AP activities
	BIKPrivateKey  string // PEM-encoded Ed25519 private key for signing BIK tokens
}

func Load() *Config {
	cfg := &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/frogs_cafe?sslmode=disable"),
		Port:          getEnv("PORT", "8080"),
		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		Environment:   getEnv("ENVIRONMENT", "development"),
		BaseURL:       getEnv("BASE_URL", "http://localhost:8080"),
		BIKPrivateKey: getEnv("BIK_PRIVATE_KEY", ""),
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
