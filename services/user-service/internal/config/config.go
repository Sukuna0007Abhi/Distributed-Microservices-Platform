package config

import (
	"os"
)

// Config holds application configuration
type Config struct {
	ServiceName  string
	Port         string
	DatabaseURL  string
	JWTSecret    string
	Environment  string
	LogLevel     string
	JaegerURL    string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		ServiceName: getEnv("SERVICE_NAME", "user-service"),
		Port:        getEnv("PORT", "8081"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:password@postgres:5432/userdb?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key"),
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		JaegerURL:   getEnv("JAEGER_URL", "http://jaeger:14268/api/traces"),
	}
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}