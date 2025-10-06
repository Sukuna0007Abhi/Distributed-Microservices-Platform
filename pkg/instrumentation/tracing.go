// OpenTelemetry instrumentation for Go services
package instrumentation

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// TracerProvider holds the tracer provider
type TracerProvider struct {
	provider *tracesdk.TracerProvider
}

// NewTracerProvider creates a new tracer provider
func NewTracerProvider(serviceName, jaegerURL string) (*TracerProvider, error) {
	// Create Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerURL)))
	if err != nil {
		return nil, err
	}

	// Create tracer provider
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("1.0.0"),
		)),
	)

	// Register as global tracer provider
	otel.SetTracerProvider(tp)
	
	// Set global propagator to tracecontext (the default is no-op)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return &TracerProvider{provider: tp}, nil
}

// Shutdown shuts down the tracer provider
func (t *TracerProvider) Shutdown(ctx context.Context) error {
	return t.provider.Shutdown(ctx)
}

// GetTracer returns a tracer for the given name
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// SetupMetrics sets up Prometheus metrics for the service
func SetupMetrics(serviceName string) error {
	// This would setup Prometheus metrics
	// Implementation depends on specific metrics requirements
	log.Printf("Setting up metrics for service: %s", serviceName)
	return nil
}