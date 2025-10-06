package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL             string
	MaxConnections  int
	MaxIdleTime     time.Duration
	MaxLifetime     time.Duration
	ConnectTimeout  time.Duration
	QueryTimeout    time.Duration
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL      string
	Password string
	DB       int
	PoolSize int
	Timeout  time.Duration
}

// TracingConfig holds distributed tracing configuration
type TracingConfig struct {
	JaegerURL       string
	SamplingRate    float64
	ServiceName     string
	ServiceVersion  string
	Environment     string
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	JWTSecret           string
	JWTExpiration       time.Duration
	PasswordMinLength   int
	RateLimitPerMinute  int
	EnableCORS          bool
	AllowedOrigins      []string
	TLSEnabled          bool
	TLSCertFile         string
	TLSKeyFile          string
}

// ObservabilityConfig holds monitoring and logging configuration
type ObservabilityConfig struct {
	LogLevel            string
	LogFormat           string // json or text
	MetricsEnabled      bool
	HealthCheckInterval time.Duration
	ProfilingEnabled    bool
}

// BaseConfig contains common configuration for all services
type BaseConfig struct {
	ServiceName     string
	Port            string
	Environment     string
	Debug           bool
	Database        DatabaseConfig
	Redis           RedisConfig
	Tracing         TracingConfig
	Security        SecurityConfig
	Observability   ObservabilityConfig
}

// LoadBaseConfig loads base configuration from environment variables
func LoadBaseConfig(serviceName string) *BaseConfig {
	return &BaseConfig{
		ServiceName: getEnvOrDefault("SERVICE_NAME", serviceName),
		Port:        getEnvOrDefault("PORT", "8080"),
		Environment: getEnvOrDefault("ENVIRONMENT", "development"),
		Debug:       getBoolEnvOrDefault("DEBUG", false),
		
		Database: DatabaseConfig{
			URL:             getEnvOrDefault("DATABASE_URL", "postgres://postgres:password@localhost:5432/"+serviceName+"?sslmode=disable"),
			MaxConnections:  getIntEnvOrDefault("DB_MAX_CONNECTIONS", 25),
			MaxIdleTime:     getDurationEnvOrDefault("DB_MAX_IDLE_TIME", 15*time.Minute),
			MaxLifetime:     getDurationEnvOrDefault("DB_MAX_LIFETIME", 1*time.Hour),
			ConnectTimeout:  getDurationEnvOrDefault("DB_CONNECT_TIMEOUT", 10*time.Second),
			QueryTimeout:    getDurationEnvOrDefault("DB_QUERY_TIMEOUT", 30*time.Second),
		},
		
		Redis: RedisConfig{
			URL:      getEnvOrDefault("REDIS_URL", "redis:6379"),
			Password: getEnvOrDefault("REDIS_PASSWORD", ""),
			DB:       getIntEnvOrDefault("REDIS_DB", 0),
			PoolSize: getIntEnvOrDefault("REDIS_POOL_SIZE", 10),
			Timeout:  getDurationEnvOrDefault("REDIS_TIMEOUT", 5*time.Second),
		},
		
		Tracing: TracingConfig{
			JaegerURL:      getEnvOrDefault("JAEGER_URL", "http://jaeger:14268/api/traces"),
			SamplingRate:   getFloatEnvOrDefault("JAEGER_SAMPLING_RATE", 1.0),
			ServiceName:    serviceName,
			ServiceVersion: getEnvOrDefault("SERVICE_VERSION", "1.0.0"),
			Environment:    getEnvOrDefault("ENVIRONMENT", "development"),
		},
		
		Security: SecurityConfig{
			JWTSecret:          getEnvOrDefault("JWT_SECRET", "your-secret-key-change-in-production"),
			JWTExpiration:      getDurationEnvOrDefault("JWT_EXPIRATION", 24*time.Hour),
			PasswordMinLength:  getIntEnvOrDefault("PASSWORD_MIN_LENGTH", 8),
			RateLimitPerMinute: getIntEnvOrDefault("RATE_LIMIT_PER_MINUTE", 100),
			EnableCORS:         getBoolEnvOrDefault("ENABLE_CORS", true),
			AllowedOrigins:     getStringSliceEnvOrDefault("ALLOWED_ORIGINS", []string{"*"}),
			TLSEnabled:         getBoolEnvOrDefault("TLS_ENABLED", false),
			TLSCertFile:        getEnvOrDefault("TLS_CERT_FILE", ""),
			TLSKeyFile:         getEnvOrDefault("TLS_KEY_FILE", ""),
		},
		
		Observability: ObservabilityConfig{
			LogLevel:            getEnvOrDefault("LOG_LEVEL", "info"),
			LogFormat:           getEnvOrDefault("LOG_FORMAT", "json"),
			MetricsEnabled:      getBoolEnvOrDefault("METRICS_ENABLED", true),
			HealthCheckInterval: getDurationEnvOrDefault("HEALTH_CHECK_INTERVAL", 30*time.Second),
			ProfilingEnabled:    getBoolEnvOrDefault("PROFILING_ENABLED", false),
		},
	}
}

// Validate validates the configuration
func (c *BaseConfig) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}
	
	if c.Port == "" {
		return fmt.Errorf("port is required")
	}
	
	if c.Database.URL == "" {
		return fmt.Errorf("database URL is required")
	}
	
	if c.Security.JWTSecret == "your-secret-key-change-in-production" && c.Environment == "production" {
		return fmt.Errorf("JWT secret must be changed in production")
	}
	
	if c.Security.PasswordMinLength < 6 {
		return fmt.Errorf("password minimum length must be at least 6")
	}
	
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, c.Observability.LogLevel) {
		return fmt.Errorf("invalid log level: %s, must be one of %v", c.Observability.LogLevel, validLogLevels)
	}
	
	return nil
}

// IsDevelopment returns true if running in development environment
func (c *BaseConfig) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production environment
func (c *BaseConfig) IsProduction() bool {
	return c.Environment == "production"
}

// GetDatabaseURL returns the database URL with proper formatting
func (c *BaseConfig) GetDatabaseURL() string {
	return c.Database.URL
}

// GetRedisURL returns the Redis URL
func (c *BaseConfig) GetRedisURL() string {
	return c.Redis.URL
}

// GetJWTSecret returns the JWT secret
func (c *BaseConfig) GetJWTSecret() string {
	return c.Security.JWTSecret
}

// Helper functions for environment variable parsing

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnvOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getBoolEnvOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getFloatEnvOrDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

func getDurationEnvOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getStringSliceEnvOrDefault(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}