# Distributed Microservices Platform

A scalable microservices architecture built with Go, gRPC, Docker, and Kubernetes, featuring distributed tracing with OpenTelemetry and service mesh integration using Istio.

## Architecture Overview

This platform consists of:
- **User Service**: Manages user authentication and profiles
- **Order Service**: Handles order processing and management
- **Product Service**: Manages product catalog
- **Payment Service**: Processes payments
- **Notification Service**: Handles notifications
- **API Gateway**: Routes requests and provides unified API

## Technologies

- **Go**: Primary programming language
- **gRPC**: Inter-service communication
- **Docker**: Containerization
- **Kubernetes**: Orchestration
- **OpenTelemetry**: Distributed tracing
- **Istio**: Service mesh for traffic management
- **Prometheus**: Metrics collection
- **Jaeger**: Tracing visualization

## Project Structure

```
microservices-platform/
├── services/
│   ├── user-service/
│   ├── order-service/
│   ├── product-service/
│   ├── payment-service/
│   └── notification-service/
├── api-gateway/
├── proto/
├── k8s/
├── istio/
├── monitoring/
└── docker-compose.yml
```

## Getting Started

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Kubernetes cluster
- kubectl
- Istio CLI

### Local Development

1. **Clone and setup**:
```bash
git clone <repository>
cd microservices-platform
```

2. **Start services with Docker Compose**:
```bash
docker-compose up -d
```

3. **Deploy to Kubernetes**:
```bash
kubectl apply -f k8s/
```

4. **Install Istio service mesh**:
```bash
istioctl install --set values.defaultRevision=default
kubectl apply -f istio/
```

### Service Endpoints

- API Gateway: `http://localhost:8080`
- User Service: `http://localhost:8081`
- Order Service: `http://localhost:8082`
- Product Service: `http://localhost:8083`
- Payment Service: `http://localhost:8084`
- Notification Service: `http://localhost:8085`

### Monitoring

- Prometheus: `http://localhost:9090`
- Jaeger: `http://localhost:16686`
- Grafana: `http://localhost:3000`

## Development

### Adding New Services

1. Create service directory in `services/`
2. Define gRPC proto in `proto/`
3. Generate Go code: `make proto-gen`
4. Implement service logic
5. Add Dockerfile and K8s manifests
6. Update docker-compose.yml

### Testing

```bash
# Unit tests
make test

# Integration tests
make test-integration

# End-to-end tests
make test-e2e
```

## Deployment

### Production Deployment

```bash
# Build and push images
make build-all
make push-all

# Deploy to production cluster
kubectl apply -f k8s/production/
```

### Monitoring & Observability

The platform includes comprehensive observability:

- **Metrics**: Prometheus scrapes metrics from all services
- **Tracing**: OpenTelemetry sends traces to Jaeger
- **Logging**: Structured logging with correlation IDs
- **Health Checks**: Kubernetes probes for service health

## Contributing

1. Fork the repository
2. Create feature branch
3. Make changes with tests
4. Submit pull request

## License

MIT License