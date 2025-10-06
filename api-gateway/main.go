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
)

type Config struct {
	Port              string
	UserServiceURL    string
	OrderServiceURL   string
	ProductServiceURL string
	PaymentServiceURL string
	NotificationServiceURL string
}

func loadConfig() *Config {
	return &Config{
		Port:                   getEnv("PORT", "8080"),
		UserServiceURL:         getEnv("USER_SERVICE_URL", "user-service:8081"),
		OrderServiceURL:        getEnv("ORDER_SERVICE_URL", "order-service:8082"),
		ProductServiceURL:      getEnv("PRODUCT_SERVICE_URL", "product-service:8083"),
		PaymentServiceURL:      getEnv("PAYMENT_SERVICE_URL", "payment-service:8084"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "notification-service:8085"),
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

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	
	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(tracingMiddleware())
	router.Use(metricsMiddleware())

	// Health check
	router.GET("/health", healthCheck)

	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes
	setupRoutes(router, cfg)

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("API Gateway starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down API Gateway...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("API Gateway forced to shutdown: %v", err)
	}

	log.Println("API Gateway stopped")
}

func setupRoutes(router *gin.Engine, cfg *Config) {
	api := router.Group("/api/v1")
	
	// User service routes
	userGroup := api.Group("/users")
	{
		userGroup.POST("", proxyToService(cfg.UserServiceURL, "/api/v1/users"))
		userGroup.GET("/:id", proxyToService(cfg.UserServiceURL, "/api/v1/users"))
		userGroup.PUT("/:id", proxyToService(cfg.UserServiceURL, "/api/v1/users"))
		userGroup.DELETE("/:id", proxyToService(cfg.UserServiceURL, "/api/v1/users"))
		userGroup.GET("", proxyToService(cfg.UserServiceURL, "/api/v1/users"))
	}

	// Auth routes
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/login", proxyToService(cfg.UserServiceURL, "/api/v1/auth/login"))
	}

	// Order service routes
	orderGroup := api.Group("/orders")
	{
		orderGroup.POST("", proxyToService(cfg.OrderServiceURL, "/api/v1/orders"))
		orderGroup.GET("/:id", proxyToService(cfg.OrderServiceURL, "/api/v1/orders"))
		orderGroup.PUT("/:id/status", proxyToService(cfg.OrderServiceURL, "/api/v1/orders"))
		orderGroup.POST("/:id/cancel", proxyToService(cfg.OrderServiceURL, "/api/v1/orders"))
		orderGroup.GET("", proxyToService(cfg.OrderServiceURL, "/api/v1/orders"))
	}

	// Product service routes
	productGroup := api.Group("/products")
	{
		productGroup.POST("", proxyToService(cfg.ProductServiceURL, "/api/v1/products"))
		productGroup.GET("/:id", proxyToService(cfg.ProductServiceURL, "/api/v1/products"))
		productGroup.PUT("/:id", proxyToService(cfg.ProductServiceURL, "/api/v1/products"))
		productGroup.DELETE("/:id", proxyToService(cfg.ProductServiceURL, "/api/v1/products"))
		productGroup.GET("", proxyToService(cfg.ProductServiceURL, "/api/v1/products"))
		productGroup.GET("/search", proxyToService(cfg.ProductServiceURL, "/api/v1/products/search"))
		productGroup.PUT("/:id/inventory", proxyToService(cfg.ProductServiceURL, "/api/v1/products"))
	}

	// Payment service routes
	paymentGroup := api.Group("/payments")
	{
		paymentGroup.POST("", proxyToService(cfg.PaymentServiceURL, "/api/v1/payments"))
		paymentGroup.GET("/:id", proxyToService(cfg.PaymentServiceURL, "/api/v1/payments"))
		paymentGroup.POST("/:id/refund", proxyToService(cfg.PaymentServiceURL, "/api/v1/payments"))
		paymentGroup.GET("", proxyToService(cfg.PaymentServiceURL, "/api/v1/payments"))
		paymentGroup.POST("/webhook", proxyToService(cfg.PaymentServiceURL, "/api/v1/payments/webhook"))
	}

	// Notification service routes
	notificationGroup := api.Group("/notifications")
	{
		notificationGroup.POST("", proxyToService(cfg.NotificationServiceURL, "/api/v1/notifications"))
		notificationGroup.GET("/:id", proxyToService(cfg.NotificationServiceURL, "/api/v1/notifications"))
		notificationGroup.GET("", proxyToService(cfg.NotificationServiceURL, "/api/v1/notifications"))
		notificationGroup.PUT("/:id/read", proxyToService(cfg.NotificationServiceURL, "/api/v1/notifications"))
		notificationGroup.DELETE("/:id", proxyToService(cfg.NotificationServiceURL, "/api/v1/notifications"))
		notificationGroup.POST("/subscribe", proxyToService(cfg.NotificationServiceURL, "/api/v1/notifications/subscribe"))
	}
}

func proxyToService(serviceURL, basePath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// This is a simplified proxy - in production you'd use a proper reverse proxy
		c.JSON(http.StatusOK, gin.H{
			"message":     "Request would be proxied to " + serviceURL,
			"path":        c.Request.URL.Path,
			"method":      c.Request.Method,
			"service_url": serviceURL,
		})
	}
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "api-gateway",
		"time":    time.Now().UTC(),
	})
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func tracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tracer := otel.Tracer("api-gateway")
		ctx, span := tracer.Start(c.Request.Context(), c.FullPath())
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		
		// Here you would record metrics like request duration, status code, etc.
		log.Printf("Request: %s %s - Status: %d - Duration: %v", 
			c.Request.Method, c.Request.URL.Path, c.Writer.Status(), duration)
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