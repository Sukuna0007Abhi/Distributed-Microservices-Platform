package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	"microservices-platform/pkg/middleware"
	"microservices-platform/pkg/proxy"
	"microservices-platform/pkg/resilience"
)

type Config struct {
	Port                   string
	UserServiceURL         string
	OrderServiceURL        string
	ProductServiceURL      string
	PaymentServiceURL      string
	NotificationServiceURL string
	JWTSecret              string
	Environment            string
}

func loadConfig() *Config {
	return &Config{
		Port:                   getEnv("PORT", "8080"),
		UserServiceURL:         getEnv("USER_SERVICE_URL", "user-service:8081"),
		OrderServiceURL:        getEnv("ORDER_SERVICE_URL", "order-service:8082"),
		ProductServiceURL:      getEnv("PRODUCT_SERVICE_URL", "product-service:8083"),
		PaymentServiceURL:      getEnv("PAYMENT_SERVICE_URL", "payment-service:8084"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "notification-service:8085"),
		JWTSecret:              getEnv("JWT_SECRET", "your-jwt-secret-key"),
		Environment:            getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func main() {
	// Initialize OpenTelemetry
	tp, err := initTracer("api-gateway")
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	// Load configuration
	cfg := loadConfig()

	// Initialize gateway with services
	gateway := setupGateway(cfg)

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	
	// Global middleware
	router.Use(proxy.RequestLoggingHandler())
	router.Use(gin.Recovery())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.TracingMiddleware("api-gateway"))
	router.Use(middleware.MetricsMiddleware())
	router.Use(middleware.RequestIDMiddleware())

	// Health checks
	router.GET("/health", gateway.HealthCheckHandler())
	router.GET("/health/:service", serviceHealthHandler(gateway))

	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes with proper authentication and authorization
	setupAPIRoutes(router, gateway, cfg)

	// Create HTTP server with timeouts
	srv := &http.Server{
		Addr:           ":" + cfg.Port,
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🚀 API Gateway starting on port %s (environment: %s)", cfg.Port, cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down API Gateway...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("API Gateway forced to shutdown: %v", err)
	}

	log.Println("✅ API Gateway stopped gracefully")
}

// setupGateway configures the gateway with all microservices
func setupGateway(cfg *Config) *proxy.Gateway {
	gateway := proxy.NewGateway()

	// Register services with circuit breakers and health checks
	services := []*proxy.ServiceConfig{
		{
			Name:       "user-service",
			URL:        "http://" + cfg.UserServiceURL,
			HealthPath: "/health",
			Timeout:    30 * time.Second,
			CircuitBreaker: resilience.NewCircuitBreaker(resilience.CircuitBreakerSettings{
				MaxFailures:      5,
				ResetTimeout:     60 * time.Second,
				SuccessThreshold: 3,
				Timeout:          30 * time.Second,
			}),
		},
		{
			Name:       "order-service",
			URL:        "http://" + cfg.OrderServiceURL,
			HealthPath: "/health",
			Timeout:    30 * time.Second,
			CircuitBreaker: resilience.NewCircuitBreaker(resilience.DefaultSettings()),
		},
		{
			Name:       "product-service",
			URL:        "http://" + cfg.ProductServiceURL,
			HealthPath: "/health",
			Timeout:    30 * time.Second,
			CircuitBreaker: resilience.NewCircuitBreaker(resilience.DefaultSettings()),
		},
		{
			Name:       "payment-service",
			URL:        "http://" + cfg.PaymentServiceURL,
			HealthPath: "/health",
			Timeout:    30 * time.Second,
			CircuitBreaker: resilience.NewCircuitBreaker(resilience.DefaultSettings()),
		},
		{
			Name:       "notification-service",
			URL:        "http://" + cfg.NotificationServiceURL,
			HealthPath: "/health",
			Timeout:    30 * time.Second,
			CircuitBreaker: resilience.NewCircuitBreaker(resilience.DefaultSettings()),
		},
	}

	for _, service := range services {
		gateway.RegisterService(service)
	}

	return gateway
}

// setupAPIRoutes configures API routes with proper authentication
func setupAPIRoutes(router *gin.Engine, gateway *proxy.Gateway, cfg *Config) {
	api := router.Group("/api/v1")
	
	// Public routes (no authentication required)
	public := api.Group("/")
	{
		// Authentication endpoint
		public.POST("/auth/login", gateway.ProxyHandler("user-service"))
		
		// Public product endpoints
		public.GET("/products", gateway.ProxyHandler("product-service"))
		public.GET("/products/:id", gateway.ProxyHandler("product-service"))
		public.GET("/products/search", gateway.ProxyHandler("product-service"))
	}

	// Protected routes (authentication required)
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		// User management
		userGroup := protected.Group("/users")
		{
			userGroup.POST("", gateway.ProxyHandler("user-service"))
			userGroup.GET("/:id", gateway.ProxyHandler("user-service"))
			userGroup.PUT("/:id", gateway.ProxyHandler("user-service"))
			userGroup.DELETE("/:id", gateway.ProxyHandler("user-service"))
			userGroup.GET("", gateway.ProxyHandler("user-service"))
		}

		// Order management
		orderGroup := protected.Group("/orders")
		{
			orderGroup.POST("", gateway.ProxyHandler("order-service"))
			orderGroup.GET("/:id", gateway.ProxyHandler("order-service"))
			orderGroup.PUT("/:id/status", gateway.ProxyHandler("order-service"))
			orderGroup.POST("/:id/cancel", gateway.ProxyHandler("order-service"))
			orderGroup.GET("", gateway.ProxyHandler("order-service"))
		}

		// Payment management
		paymentGroup := protected.Group("/payments")
		{
			paymentGroup.POST("", gateway.ProxyHandler("payment-service"))
			paymentGroup.GET("/:id", gateway.ProxyHandler("payment-service"))
			paymentGroup.POST("/:id/refund", gateway.ProxyHandler("payment-service"))
			paymentGroup.GET("", gateway.ProxyHandler("payment-service"))
		}

		// Notification management
		notificationGroup := protected.Group("/notifications")
		{
			notificationGroup.POST("", gateway.ProxyHandler("notification-service"))
			notificationGroup.GET("/:id", gateway.ProxyHandler("notification-service"))
			notificationGroup.GET("", gateway.ProxyHandler("notification-service"))
			notificationGroup.PUT("/:id/read", gateway.ProxyHandler("notification-service"))
			notificationGroup.DELETE("/:id", gateway.ProxyHandler("notification-service"))
			notificationGroup.POST("/subscribe", gateway.ProxyHandler("notification-service"))
		}
	}

	// Admin routes (admin authentication required)
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	// TODO: Add admin role validation
	{
		// Product management (admin only)
		adminProductGroup := admin.Group("/products")
		{
			adminProductGroup.POST("", gateway.ProxyHandler("product-service"))
			adminProductGroup.PUT("/:id", gateway.ProxyHandler("product-service"))
			adminProductGroup.DELETE("/:id", gateway.ProxyHandler("product-service"))
			adminProductGroup.PUT("/:id/inventory", gateway.ProxyHandler("product-service"))
		}
	}

	// Webhook endpoints (no authentication, but should validate signatures)
	webhooks := api.Group("/webhooks")
	{
		webhooks.POST("/payments/:provider", gateway.ProxyHandler("payment-service"))
	}
}

// serviceHealthHandler returns health status for a specific service
func serviceHealthHandler(gateway *proxy.Gateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.Param("service")
		
		// This would require exposing service health checking from the gateway
		c.JSON(http.StatusOK, gin.H{
			"service": serviceName,
			"message": "Service health check endpoint - implementation needed",
		})
	}
}



func initTracer(serviceName string) (*tracesdk.TracerProvider, error) {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://jaeger:14268/api/traces")))
	if err != nil {
		return nil, err
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}