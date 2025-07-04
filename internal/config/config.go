package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port              string
	DatabaseURL       string
	JWTSecret         string
	TokenExpiration   time.Duration
	RefreshExpiration time.Duration
	LogLevel          string
}

func Load() *Config {
	tokenExp, _ := strconv.Atoi(getEnv("TOKEN_EXPIRATION_MINUTES", "60"))
	refreshExp, _ := strconv.Atoi(getEnv("REFRESH_EXPIRATION_HOURS", "168")) // 7 дней

	return &Config{
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://user:password@localhost:5432/oauth2_db?sslmode=disable"),
		JWTSecret:         getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production"),
		TokenExpiration:   time.Duration(tokenExp) * time.Minute,
		RefreshExpiration: time.Duration(refreshExp) * time.Hour,
		LogLevel:          getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
