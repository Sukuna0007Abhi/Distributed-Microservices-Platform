package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"microservices-platform/pkg/resilience"
)

// ServiceConfig defines configuration for a service
type ServiceConfig struct {
	Name        string
	URL         string
	HealthPath  string
	Timeout     time.Duration
	Retries     int
	CircuitBreaker *resilience.CircuitBreaker
}

// Gateway represents the API Gateway with reverse proxy capabilities
type Gateway struct {
	services map[string]*ServiceConfig
	tracer   trace.Tracer
}

// NewGateway creates a new API Gateway
func NewGateway() *Gateway {
	return &Gateway{
		services: make(map[string]*ServiceConfig),
		tracer:   otel.Tracer("api-gateway"),
	}
}

// RegisterService registers a service with the gateway
func (g *Gateway) RegisterService(service *ServiceConfig) {
	if service.CircuitBreaker == nil {
		service.CircuitBreaker = resilience.NewCircuitBreaker(resilience.DefaultSettings())
	}
	if service.Timeout == 0 {
		service.Timeout = 30 * time.Second
	}
	g.services[service.Name] = service
	log.Printf("Registered service: %s -> %s", service.Name, service.URL)
}

// ProxyHandler creates a gin handler that proxies requests to the specified service
func (g *Gateway) ProxyHandler(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		service, exists := g.services[serviceName]
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
			return
		}

		ctx, span := g.tracer.Start(c.Request.Context(), "gateway.proxy",
			trace.WithAttributes(
				attribute.String("service.name", serviceName),
				attribute.String("service.url", service.URL),
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.path", c.Request.URL.Path),
			),
		)
		defer span.End()

		// Execute request with circuit breaker
		err := service.CircuitBreaker.Execute(ctx, func() error {
			return g.proxyRequest(c, service)
		})

		if err != nil {
			span.RecordError(err)
			log.Printf("Proxy error for service %s: %v", serviceName, err)
			
			if err.Error() == "circuit breaker is open" {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error": "Service temporarily unavailable",
					"service": serviceName,
				})
			} else {
				c.JSON(http.StatusBadGateway, gin.H{
					"error": "Bad gateway",
					"service": serviceName,
				})
			}
		}
	}
}

// proxyRequest proxies the request to the target service
func (g *Gateway) proxyRequest(c *gin.Context, service *ServiceConfig) error {
	// Parse target URL
	targetURL, err := url.Parse(service.URL)
	if err != nil {
		return fmt.Errorf("invalid target URL: %v", err)
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	
	// Custom director to modify the request
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		
		// Add tracing headers
		if span := trace.SpanFromContext(req.Context()); span.SpanContext().IsValid() {
			req.Header.Set("X-Trace-ID", span.SpanContext().TraceID().String())
			req.Header.Set("X-Span-ID", span.SpanContext().SpanID().String())
		}
		
		// Add gateway headers
		req.Header.Set("X-Forwarded-By", "api-gateway")
		req.Header.Set("X-Gateway-Service", service.Name)
	}

	// Custom error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error": "Bad gateway"}`))
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), service.Timeout)
	defer cancel()
	c.Request = c.Request.WithContext(ctx)

	// Execute proxy
	proxy.ServeHTTP(c.Writer, c.Request)
	return nil
}

// HealthCheckHandler checks the health of all registered services
func (g *Gateway) HealthCheckHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		results := make(map[string]interface{})
		overallHealthy := true

		for name, service := range g.services {
			healthy, details := g.checkServiceHealth(service)
			results[name] = map[string]interface{}{
				"healthy": healthy,
				"details": details,
				"circuit_breaker": service.CircuitBreaker.GetStats(),
			}
			
			if !healthy {
				overallHealthy = false
			}
		}

		status := http.StatusOK
		if !overallHealthy {
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, gin.H{
			"status": map[string]bool{"healthy": overallHealthy},
			"services": results,
			"timestamp": time.Now().UTC(),
		})
	}
}

// checkServiceHealth checks if a service is healthy
func (g *Gateway) checkServiceHealth(service *ServiceConfig) (bool, string) {
	if service.HealthPath == "" {
		return true, "No health check configured"
	}

	healthURL := strings.TrimSuffix(service.URL, "/") + "/" + strings.TrimPrefix(service.HealthPath, "/")
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return false, fmt.Sprintf("Failed to create health check request: %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, "Health check passed"
	}

	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Sprintf("Health check failed with status %d: %s", resp.StatusCode, string(body))
}

// LoadBalancer interface for different load balancing strategies
type LoadBalancer interface {
	SelectService(services []*ServiceConfig) *ServiceConfig
}

// RoundRobinBalancer implements round-robin load balancing
type RoundRobinBalancer struct {
	current int
}

// SelectService selects the next service using round-robin
func (rb *RoundRobinBalancer) SelectService(services []*ServiceConfig) *ServiceConfig {
	if len(services) == 0 {
		return nil
	}
	
	service := services[rb.current%len(services)]
	rb.current++
	return service
}

// RequestLoggingHandler logs all requests
func RequestLoggingHandler() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %d %s %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
		)
	})
}