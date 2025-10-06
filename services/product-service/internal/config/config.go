package config

import (
	"os"
	"time"
)

// Config holds application configuration
type Config struct {
	ServiceName        string
	Port               string
	DatabaseURL        string
	RedisURL           string
	Environment        string
	LogLevel           string
	JaegerURL          string
	CacheEnabled       bool
	CacheTTL           time.Duration
}

// Load loads configuration from environment variables
func Load() *Config {
	cacheTTL, _ := time.ParseDuration(getEnv("CACHE_TTL", "5m"))
	
	return &Config{
		ServiceName:  getEnv("SERVICE_NAME", "product-service"),
		Port:         getEnv("PORT", "8083"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://postgres:password@postgres:5432/productdb?sslmode=disable"),
		RedisURL:     getEnv("REDIS_URL", "redis:6379"),
		Environment:  getEnv("ENVIRONMENT", "development"),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		JaegerURL:    getEnv("JAEGER_URL", "http://jaeger:14268/api/traces"),
		CacheEnabled: getEnv("CACHE_ENABLED", "true") == "true",
		CacheTTL:     cacheTTL,
	}
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}