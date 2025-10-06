package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for the microservices platform
var (
	// HTTP metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"service", "method", "endpoint", "status_code"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "endpoint"},
	)

	// gRPC metrics
	GRPCRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"service", "method", "status_code"},
	)

	GRPCRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_request_duration_seconds",
			Help:    "gRPC request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method"},
	)

	// Database metrics
	DatabaseConnectionsActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_connections_active",
			Help: "Number of active database connections",
		},
		[]string{"service", "database"},
	)

	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "operation", "table"},
	)

	DatabaseQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"service", "operation", "table", "status"},
	)

	// Cache metrics
	CacheHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"service", "cache_name"},
	)

	CacheMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"service", "cache_name"},
	)

	// Business metrics
	UsersTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "users_total",
			Help: "Total number of users",
		},
		[]string{"status"},
	)

	OrdersTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orders_total",
			Help: "Total number of orders",
		},
		[]string{"status"},
	)

	OrderValue = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "order_value_dollars",
			Help:    "Order value in dollars",
			Buckets: []float64{10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		},
		[]string{"status"},
	)

	PaymentsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payments_total",
			Help: "Total number of payments",
		},
		[]string{"method", "status"},
	)

	PaymentAmount = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "payment_amount_dollars",
			Help:    "Payment amount in dollars",
			Buckets: []float64{10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		},
		[]string{"method", "status"},
	)

	// Event metrics
	EventsPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_published_total",
			Help: "Total number of events published",
		},
		[]string{"service", "event_type"},
	)

	EventsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_processed_total",
			Help: "Total number of events processed",
		},
		[]string{"service", "event_type", "status"},
	)

	EventProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "event_processing_duration_seconds",
			Help:    "Event processing duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "event_type"},
	)

	// Circuit breaker metrics
	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=half-open, 2=open)",
		},
		[]string{"service", "circuit_name"},
	)

	CircuitBreakerRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_requests_total",
			Help: "Total number of circuit breaker requests",
		},
		[]string{"service", "circuit_name", "result"},
	)
)

// RecordHTTPRequest records an HTTP request metric
func RecordHTTPRequest(service, method, endpoint, statusCode string, duration time.Duration) {
	HTTPRequestsTotal.WithLabelValues(service, method, endpoint, statusCode).Inc()
	HTTPRequestDuration.WithLabelValues(service, method, endpoint).Observe(duration.Seconds())
}

// RecordGRPCRequest records a gRPC request metric
func RecordGRPCRequest(service, method, statusCode string, duration time.Duration) {
	GRPCRequestsTotal.WithLabelValues(service, method, statusCode).Inc()
	GRPCRequestDuration.WithLabelValues(service, method).Observe(duration.Seconds())
}

// RecordDatabaseQuery records a database query metric
func RecordDatabaseQuery(service, operation, table, status string, duration time.Duration) {
	DatabaseQueriesTotal.WithLabelValues(service, operation, table, status).Inc()
	DatabaseQueryDuration.WithLabelValues(service, operation, table).Observe(duration.Seconds())
}

// RecordCacheHit records a cache hit
func RecordCacheHit(service, cacheName string) {
	CacheHitsTotal.WithLabelValues(service, cacheName).Inc()
}

// RecordCacheMiss records a cache miss
func RecordCacheMiss(service, cacheName string) {
	CacheMissesTotal.WithLabelValues(service, cacheName).Inc()
}

// RecordOrder records an order metric
func RecordOrder(status string, value float64) {
	OrdersTotal.WithLabelValues(status).Inc()
	OrderValue.WithLabelValues(status).Observe(value)
}

// RecordPayment records a payment metric
func RecordPayment(method, status string, amount float64) {
	PaymentsTotal.WithLabelValues(method, status).Inc()
	PaymentAmount.WithLabelValues(method, status).Observe(amount)
}

// RecordEvent records event metrics
func RecordEventPublished(service, eventType string) {
	EventsPublished.WithLabelValues(service, eventType).Inc()
}

func RecordEventProcessed(service, eventType, status string, duration time.Duration) {
	EventsProcessed.WithLabelValues(service, eventType, status).Inc()
	EventProcessingDuration.WithLabelValues(service, eventType).Observe(duration.Seconds())
}

// UpdateCircuitBreakerState updates circuit breaker state metric
func UpdateCircuitBreakerState(service, circuitName string, state int) {
	CircuitBreakerState.WithLabelValues(service, circuitName).Set(float64(state))
}

// RecordCircuitBreakerRequest records a circuit breaker request
func RecordCircuitBreakerRequest(service, circuitName, result string) {
	CircuitBreakerRequests.WithLabelValues(service, circuitName, result).Inc()
}