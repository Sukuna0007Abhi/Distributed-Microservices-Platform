package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	"microservices-platform/services/order-service/internal/config"
	"microservices-platform/services/order-service/internal/database"
	"microservices-platform/services/order-service/internal/handler"
	"microservices-platform/services/order-service/internal/repository"
	"microservices-platform/services/order-service/internal/service"
	pb "microservices-platform/pkg/proto/order/v1"
)

func main() {
	// Initialize configuration
	cfg := config.Load()

	// Initialize OpenTelemetry
	tp, err := initTracer(cfg.ServiceName)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	// Initialize database
	db, err := database.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize repository
	orderRepo := repository.NewOrderRepository(db)

	// Initialize service
	orderService := service.NewOrderService(orderRepo, cfg)

	// Initialize gRPC handler
	orderHandler := handler.NewOrderHandler(orderService)

	// Create gRPC server with OpenTelemetry interceptors
	server := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
	)

	// Register service
	pb.RegisterOrderServiceServer(server, orderHandler)

	// Enable reflection for debugging
	reflection.Register(server)

	// Create listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Port))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.Port, err)
	}

	// Graceful shutdown
	go func() {
		log.Printf("Order service starting on port %s", cfg.Port)
		if err := server.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down order service...")
	server.GracefulStop()
	log.Println("Order service stopped")
}

// initTracer creates and configures OpenTelemetry tracer
func initTracer(serviceName string) (*tracesdk.TracerProvider, error) {
	// Create Jaeger exporter
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