# Deployment Guide

## Prerequisites

Before deploying the Distributed Microservices Platform, ensure you have the following installed:

### Local Development
- Go 1.21 or later
- Docker and Docker Compose
- Make (optional, for using Makefile)

### Kubernetes Deployment
- Kubernetes cluster (1.24+)
- kubectl configured to access your cluster
- Docker registry access
- Helm 3 (optional)

### Service Mesh (Optional)
- Istio 1.17+ for service mesh functionality
- istioctl CLI tool

## Local Development Setup

### 1. Clone and Setup

```bash
git clone <repository-url>
cd microservices-platform
```

### 2. Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install development tools (optional)
make install-tools
```

### 3. Generate Protobuf Files

```bash
# Generate gRPC code from proto files
make proto-gen
```

### 4. Start Infrastructure Services

```bash
# Start PostgreSQL, Redis, Jaeger, and Prometheus
docker-compose up -d postgres redis jaeger prometheus grafana
```

### 5. Build and Run Services

Option A: Using Docker Compose (Recommended)
```bash
# Build and start all services
docker-compose up --build
```

Option B: Run individually
```bash
# Build all services
make build

# Run each service in separate terminals
cd services/user-service && ./bin/user-service
cd services/order-service && ./bin/order-service
cd services/product-service && ./bin/product-service
cd services/payment-service && ./bin/payment-service
cd services/notification-service && ./bin/notification-service
cd api-gateway && go run main.go
```

### 6. Verify Installation

```bash
# Check service health
curl http://localhost:8080/health

# Check Prometheus metrics
curl http://localhost:9090

# Check Jaeger UI
open http://localhost:16686

# Check Grafana
open http://localhost:3000
# Login: admin/admin
```

## Production Deployment

### 1. Prepare Container Images

```bash
# Set your registry
export DOCKER_REGISTRY=your-registry.com

# Build and push images
make docker-build
make docker-push

# Or use the build script
./scripts/build.sh
export PUSH_IMAGES=true ./scripts/build.sh
```

### 2. Configure Kubernetes Secrets

Create required secrets:

```bash
# Database secrets
kubectl create secret generic database-secret \
  --from-literal=user-db-url="postgres://user:pass@postgres:5432/userdb?sslmode=disable" \
  --from-literal=order-db-url="postgres://user:pass@postgres:5432/orderdb?sslmode=disable" \
  --from-literal=product-db-url="postgres://user:pass@postgres:5432/productdb?sslmode=disable" \
  --from-literal=payment-db-url="postgres://user:pass@postgres:5432/paymentdb?sslmode=disable" \
  --from-literal=notification-db-url="postgres://user:pass@postgres:5432/notificationdb?sslmode=disable" \
  -n microservices

# Application secrets
kubectl create secret generic app-secrets \
  --from-literal=jwt-secret="your-jwt-secret-key" \
  --from-literal=encryption-key="your-encryption-key" \
  -n microservices
```

### 3. Deploy to Kubernetes

Option A: Using deployment script
```bash
./scripts/deploy.sh
```

Option B: Manual deployment
```bash
# Deploy namespace
kubectl apply -f k8s/namespace.yaml

# Deploy monitoring stack
kubectl apply -f monitoring/monitoring-stack.yaml

# Deploy services
kubectl apply -f k8s/services/
kubectl apply -f k8s/deployments/
```

### 4. Configure Istio Service Mesh (Optional)

```bash
# Install Istio
istioctl install --set values.defaultRevision=default

# Enable injection for namespace
kubectl label namespace microservices istio-injection=enabled

# Apply Istio configurations
kubectl apply -f istio/

# Restart deployments to inject sidecars
kubectl rollout restart deployment -n microservices
```

## Environment Configuration

### Environment Variables

Each service can be configured using environment variables:

#### API Gateway
```bash
PORT=8080
USER_SERVICE_URL=user-service:8081
ORDER_SERVICE_URL=order-service:8082
PRODUCT_SERVICE_URL=product-service:8083
PAYMENT_SERVICE_URL=payment-service:8084
NOTIFICATION_SERVICE_URL=notification-service:8085
```

#### User Service
```bash
PORT=8081
DATABASE_URL=postgres://user:pass@postgres:5432/userdb?sslmode=disable
JWT_SECRET=your-jwt-secret-key
JAEGER_URL=http://jaeger:14268/api/traces
```

#### Order Service
```bash
PORT=8082
DATABASE_URL=postgres://user:pass@postgres:5432/orderdb?sslmode=disable
USER_SERVICE_URL=user-service:8081
PRODUCT_SERVICE_URL=product-service:8083
PAYMENT_SERVICE_URL=payment-service:8084
JAEGER_URL=http://jaeger:14268/api/traces
```

### Configuration Files

You can also use configuration files:

```yaml
# config/config.yaml
server:
  port: 8081
  
database:
  url: "postgres://user:pass@postgres:5432/userdb?sslmode=disable"
  max_connections: 100
  
security:
  jwt_secret: "your-jwt-secret-key"
  
observability:
  jaeger_url: "http://jaeger:14268/api/traces"
  metrics_enabled: true
  
logging:
  level: "info"
  format: "json"
```

## Database Setup

### PostgreSQL Configuration

1. **Create databases** (done automatically by init script):
```sql
CREATE DATABASE userdb;
CREATE DATABASE orderdb;
CREATE DATABASE productdb;
CREATE DATABASE paymentdb;
CREATE DATABASE notificationdb;
```

2. **Connection pooling** (recommended for production):
```bash
# Use PgBouncer or similar
docker run -d \
  --name pgbouncer \
  -e DATABASES_HOST=postgres \
  -e DATABASES_PORT=5432 \
  -e DATABASES_USER=postgres \
  -e DATABASES_PASSWORD=password \
  -e DATABASES_DBNAME=postgres \
  pgbouncer/pgbouncer:latest
```

### Redis Configuration

For caching and session storage:
```bash
# Redis with persistence
docker run -d \
  --name redis \
  -p 6379:6379 \
  -v redis_data:/data \
  redis:7-alpine redis-server --appendonly yes
```

## Monitoring and Observability

### Prometheus Configuration

Monitoring stack includes:
- **Prometheus**: Metrics collection
- **Grafana**: Visualization dashboards
- **Jaeger**: Distributed tracing
- **AlertManager**: Alert handling

### Custom Dashboards

Import Grafana dashboards:
1. Open Grafana at http://localhost:3000
2. Login with admin/admin
3. Import dashboards from `monitoring/grafana/dashboards/`

### Alerts

Configure alerts in `monitoring/alert-rules.yml`:
```yaml
groups:
  - name: microservices
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
```

## Scaling and Performance

### Horizontal Pod Autoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: user-service-hpa
  namespace: microservices
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: user-service
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### Database Scaling

1. **Read Replicas**: Configure PostgreSQL read replicas
2. **Connection Pooling**: Use PgBouncer
3. **Caching**: Implement Redis caching layers

## Security Considerations

### TLS Configuration

1. **Enable TLS for all services**:
```yaml
spec:
  tls:
    - hosts:
      - api.yourcompany.com
      secretName: tls-secret
```

2. **mTLS with Istio**:
```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
spec:
  mtls:
    mode: STRICT
```

### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all
  namespace: microservices
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
```

## Backup and Recovery

### Database Backup

```bash
# Automated backup script
#!/bin/bash
kubectl exec postgres-pod -- pg_dumpall -U postgres | gzip > backup-$(date +%Y%m%d).sql.gz
```

### Configuration Backup

```bash
# Backup Kubernetes configurations
kubectl get all -n microservices -o yaml > k8s-backup.yaml
```

## Troubleshooting

### Common Issues

1. **Service Discovery Issues**:
```bash
# Check DNS resolution
kubectl exec -it pod-name -- nslookup service-name
```

2. **Database Connection Issues**:
```bash
# Check database connectivity
kubectl exec -it user-service-pod -- nc -zv postgres 5432
```

3. **Memory Issues**:
```bash
# Check resource usage
kubectl top pods -n microservices
```

### Debugging Commands

```bash
# Check pod logs
kubectl logs -f deployment/user-service -n microservices

# Check service endpoints
kubectl get endpoints -n microservices

# Check Istio proxy status
istioctl proxy-status

# Check distributed traces
# Open Jaeger UI and search for traces
```

## Health Checks

### Kubernetes Probes

Each service includes:
- **Liveness Probe**: Restarts unhealthy pods
- **Readiness Probe**: Routes traffic only to ready pods

### Custom Health Endpoints

Services expose `/health` endpoints:
```bash
# Check service health
curl http://service-name:port/health
```

## Maintenance

### Rolling Updates

```bash
# Update service image
kubectl set image deployment/user-service user-service=registry/user-service:v2 -n microservices

# Check rollout status
kubectl rollout status deployment/user-service -n microservices

# Rollback if needed
kubectl rollout undo deployment/user-service -n microservices
```

### Database Migrations

```bash
# Run migrations (example)
kubectl create job migrate-userdb --from=cronjob/db-migration -n microservices
```

This completes the comprehensive deployment guide for the distributed microservices platform.