package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	Env                string
	DatabaseURL        string
	JWTAccessSecret    string
	JWTRefreshSecret   string
	JWTAccessTTL       time.Duration
	JWTRefreshTTL      time.Duration
	LogLevel           string
	CORSAllowedOrigins []string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		Env:                getEnv("ENV", "dev"),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		JWTAccessSecret:    getEnv("JWT_ACCESS_SECRET", ""),
		JWTRefreshSecret:   getEnv("JWT_REFRESH_SECRET", ""),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		CORSAllowedOrigins: splitAndTrim(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:8081")),
	}

	accessTTL, err := time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL: %w", err)
	}
	cfg.JWTAccessTTL = accessTTL

	refreshTTL, err := time.ParseDuration(getEnv("JWT_REFRESH_TTL", "168h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TTL: %w", err)
	}
	cfg.JWTRefreshTTL = refreshTTL

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTAccessSecret == "" {
		return nil, fmt.Errorf("JWT_ACCESS_SECRET is required")
	}
	if cfg.JWTRefreshSecret == "" {
		return nil, fmt.Errorf("JWT_REFRESH_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func splitAndTrim(csv string) []string {
	parts := strings.Split(csv, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
