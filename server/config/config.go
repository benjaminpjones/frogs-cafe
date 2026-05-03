package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL   string
	Port          string
	JWTSecret     string
	Environment   string
	BaseURL       string   // e.g. https://frogs.cafe — used for actor URIs and AP activities
	BIKPrivateKey string   // PEM-encoded Ed25519 private key for signing BIK tokens
	BIKPeers      []string // URLs of known peer servers for federation
}

func Load() *Config {
	var peers []string
	if p := os.Getenv("BIK_PEERS"); p != "" {
		for _, s := range strings.Split(p, ",") {
			if trimmed := strings.TrimSpace(s); trimmed != "" {
				peers = append(peers, trimmed)
			}
		}
	}

	cfg := &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/frogs_cafe?sslmode=disable"),
		Port:          getEnv("PORT", "8080"),
		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		Environment:   getEnv("ENVIRONMENT", "development"),
		BaseURL:       getEnv("BASE_URL", "http://localhost:8080"),
		BIKPrivateKey: getEnv("BIK_PRIVATE_KEY", ""),
		BIKPeers:      peers,
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
