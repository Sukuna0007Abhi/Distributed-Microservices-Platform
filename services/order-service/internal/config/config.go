package config

import (
	"os"
)

// Config holds application configuration
type Config struct {
	ServiceName        string
	Port               string
	DatabaseURL        string
	Environment        string
	LogLevel           string
	JaegerURL          string
	UserServiceURL     string
	ProductServiceURL  string
	PaymentServiceURL  string
	NotificationServiceURL string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		ServiceName:            getEnv("SERVICE_NAME", "order-service"),
		Port:                   getEnv("PORT", "8082"),
		DatabaseURL:            getEnv("DATABASE_URL", "postgres://postgres:password@postgres:5432/orderdb?sslmode=disable"),
		Environment:            getEnv("ENVIRONMENT", "development"),
		LogLevel:               getEnv("LOG_LEVEL", "info"),
		JaegerURL:              getEnv("JAEGER_URL", "http://jaeger:14268/api/traces"),
		UserServiceURL:         getEnv("USER_SERVICE_URL", "user-service:8081"),
		ProductServiceURL:      getEnv("PRODUCT_SERVICE_URL", "product-service:8083"),
		PaymentServiceURL:      getEnv("PAYMENT_SERVICE_URL", "payment-service:8084"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "notification-service:8085"),
	}
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}