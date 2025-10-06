# 🚀 Distributed Microservices Platform - Enhanced Makefile

# Build variables
DOCKER_REGISTRY ?= localhost:5000
VERSION ?= latest
SERVICES = user-service order-service product-service payment-service notification-service api-gateway
GO_VERSION = 1.21
PROJECT_NAME = microservices-platform

# Go variables
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod
GOFMT = gofmt
GOLINT = golangci-lint

# Proto variables
PROTO_DIR = proto
PROTO_OUT_DIR = pkg/proto

# Build flags
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"
BUILD_FLAGS = -a -installsuffix cgo $(LDFLAGS)

# Test variables
COVERAGE_FILE = coverage.out
COVERAGE_HTML = coverage.html
TEST_TIMEOUT = 10m

# Colors for output
RED = \033[0;31m
GREEN = \033[0;32m
YELLOW = \033[0;33m
BLUE = \033[0;34m
NC = \033[0m # No Color

.PHONY: all build clean test deps proto-gen docker-build docker-push k8s-deploy help

# Default target
all: deps proto-gen build test

# 📦 Dependencies and Setup
deps:
	@echo "$(BLUE)📦 Installing Go dependencies...$(NC)"
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "$(GREEN)✅ Dependencies installed$(NC)"

deps-update:
	@echo "$(BLUE)📦 Updating Go dependencies...$(NC)"
	$(GOMOD) get -u ./...
	$(GOMOD) tidy
	@echo "$(GREEN)✅ Dependencies updated$(NC)"

install-tools:
	@echo "$(BLUE)🛠️  Installing development tools...$(NC)"
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/sast-scan/cmd/sast-scan@latest
	@echo "$(GREEN)✅ Development tools installed$(NC)"

# 🔧 Code Generation
proto-gen:
	@echo "$(BLUE)🔧 Generating protobuf files...$(NC)"
	@mkdir -p $(PROTO_OUT_DIR)
	@for proto in $(PROTO_DIR)/*.proto; do \
		echo "Generating $$proto..."; \
		protoc --go_out=$(PROTO_OUT_DIR) --go_opt=paths=source_relative \
			--go-grpc_out=$(PROTO_OUT_DIR) --go-grpc_opt=paths=source_relative \
			--grpc-gateway_out=$(PROTO_OUT_DIR) --grpc-gateway_opt=paths=source_relative \
			--openapiv2_out=$(PROTO_OUT_DIR) \
			$$proto; \
	done
	@echo "$(GREEN)✅ Protobuf files generated$(NC)"

# 🏗️  Building
build:
	@echo "$(BLUE)🏗️  Building all services...$(NC)"
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		if [ "$$service" = "api-gateway" ]; then \
			cd api-gateway && $(GOBUILD) $(BUILD_FLAGS) -o bin/$$service main.go && cd ..; \
		else \
			cd services/$$service && $(GOBUILD) $(BUILD_FLAGS) -o bin/$$service ./cmd/main.go && cd ../..; \
		fi; \
	done
	@echo "$(GREEN)✅ All services built successfully$(NC)"

build-%:
	@echo "$(BLUE)🏗️  Building $*...$(NC)"
	@if [ "$*" = "api-gateway" ]; then \
		cd api-gateway && $(GOBUILD) $(BUILD_FLAGS) -o bin/$* main.go; \
	else \
		cd services/$* && $(GOBUILD) $(BUILD_FLAGS) -o bin/$* ./cmd/main.go; \
	fi
	@echo "$(GREEN)✅ $* built successfully$(NC)"

build-linux:
	@echo "$(BLUE)🐧 Building all services for Linux...$(NC)"
	@for service in $(SERVICES); do \
		echo "Building $$service for Linux..."; \
		if [ "$$service" = "api-gateway" ]; then \
			cd api-gateway && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o bin/$$service main.go && cd ..; \
		else \
			cd services/$$service && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o bin/$$service ./cmd/main.go && cd ../..; \
		fi; \
	done
	@echo "$(GREEN)✅ All services built for Linux$(NC)"

# 🧹 Cleaning
clean:
	@echo "$(BLUE)🧹 Cleaning build artifacts...$(NC)"
	@for service in $(SERVICES); do \
		if [ "$$service" = "api-gateway" ]; then \
			rm -rf api-gateway/bin; \
		else \
			rm -rf services/$$service/bin; \
		fi; \
	done
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	$(GOCLEAN)
	@echo "$(GREEN)✅ Cleanup completed$(NC)"

# 🧪 Testing
test:
	@echo "$(BLUE)🧪 Running unit tests...$(NC)"
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) -race ./...
	@echo "$(GREEN)✅ Unit tests completed$(NC)"

test-coverage:
	@echo "$(BLUE)🧪 Running tests with coverage...$(NC)"
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) -race -coverprofile=$(COVERAGE_FILE) ./...
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	$(GOCMD) tool cover -func=$(COVERAGE_FILE)
	@echo "$(GREEN)✅ Coverage report generated: $(COVERAGE_HTML)$(NC)"

test-integration:
	@echo "$(BLUE)🧪 Running integration tests...$(NC)"
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) -tags=integration ./tests/integration/...
	@echo "$(GREEN)✅ Integration tests completed$(NC)"

test-e2e:
	@echo "$(BLUE)🧪 Running end-to-end tests...$(NC)"
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) -tags=e2e ./tests/e2e/...
	@echo "$(GREEN)✅ End-to-end tests completed$(NC)"

test-all: test test-integration test-e2e
	@echo "$(GREEN)✅ All tests completed$(NC)"

bench:
	@echo "$(BLUE)⚡ Running benchmarks...$(NC)"
	$(GOTEST) -bench=. -benchmem ./...
	@echo "$(GREEN)✅ Benchmarks completed$(NC)"

# 🔍 Code Quality
fmt:
	@echo "$(BLUE)🔍 Formatting Go code...$(NC)"
	$(GOFMT) -s -w .
	@echo "$(GREEN)✅ Code formatted$(NC)"

lint:
	@echo "$(BLUE)🔍 Running linters...$(NC)"
	$(GOLINT) run --timeout 5m
	@echo "$(GREEN)✅ Linting completed$(NC)"

vet:
	@echo "$(BLUE)🔍 Running go vet...$(NC)"
	$(GOCMD) vet ./...
	@echo "$(GREEN)✅ Vet completed$(NC)"

security-scan:
	@echo "$(BLUE)🔒 Running security scan...$(NC)"
	gosec ./...
	@echo "$(GREEN)✅ Security scan completed$(NC)"

code-quality: fmt vet lint security-scan
	@echo "$(GREEN)✅ All code quality checks completed$(NC)"

# 🐳 Docker Operations
docker-build:
	@echo "$(BLUE)🐳 Building Docker images...$(NC)"
	@for service in $(SERVICES); do \
		echo "Building Docker image for $$service..."; \
		if [ "$$service" = "api-gateway" ]; then \
			docker build -t $(DOCKER_REGISTRY)/$$service:$(VERSION) -f api-gateway/Dockerfile .; \
		else \
			docker build -t $(DOCKER_REGISTRY)/$$service:$(VERSION) -f services/$$service/Dockerfile .; \
		fi; \
	done
	@echo "$(GREEN)✅ All Docker images built$(NC)"

docker-build-%:
	@echo "$(BLUE)🐳 Building Docker image for $*...$(NC)"
	@if [ "$*" = "api-gateway" ]; then \
		docker build -t $(DOCKER_REGISTRY)/$*:$(VERSION) -f api-gateway/Dockerfile .; \
	else \
		docker build -t $(DOCKER_REGISTRY)/$*:$(VERSION) -f services/$*/Dockerfile .; \
	fi
	@echo "$(GREEN)✅ Docker image for $* built$(NC)"

docker-push:
	@echo "$(BLUE)🐳 Pushing Docker images...$(NC)"
	@for service in $(SERVICES); do \
		echo "Pushing $$service..."; \
		docker push $(DOCKER_REGISTRY)/$$service:$(VERSION); \
	done
	@echo "$(GREEN)✅ All Docker images pushed$(NC)"

docker-push-%:
	@echo "$(BLUE)🐳 Pushing Docker image for $*...$(NC)"
	docker push $(DOCKER_REGISTRY)/$*:$(VERSION)
	@echo "$(GREEN)✅ Docker image for $* pushed$(NC)"

docker-scan:
	@echo "$(BLUE)🔒 Scanning Docker images for vulnerabilities...$(NC)"
	@for service in $(SERVICES); do \
		echo "Scanning $$service..."; \
		docker scan $(DOCKER_REGISTRY)/$$service:$(VERSION) || true; \
	done
	@echo "$(GREEN)✅ Docker security scan completed$(NC)"

build-all: docker-build docker-push
	@echo "$(GREEN)✅ All services built and pushed$(NC)"

# ☸️  Kubernetes Operations
k8s-validate:
	@echo "$(BLUE)☸️  Validating Kubernetes manifests...$(NC)"
	@for file in k8s/**/*.yaml; do \
		echo "Validating $$file..."; \
		kubectl apply --dry-run=client -f $$file; \
	done
	@echo "$(GREEN)✅ Kubernetes manifests validated$(NC)"

k8s-deploy:
	@echo "$(BLUE)☸️  Deploying to Kubernetes...$(NC)"
	kubectl apply -f k8s/namespace.yaml
	@if [ -d "k8s/configmaps" ]; then kubectl apply -f k8s/configmaps/; fi
	@if [ -d "k8s/secrets" ]; then kubectl apply -f k8s/secrets/; fi
	kubectl apply -f k8s/services/
	kubectl apply -f k8s/deployments/
	@echo "$(GREEN)✅ Deployed to Kubernetes$(NC)"

k8s-status:
	@echo "$(BLUE)☸️  Checking Kubernetes deployment status...$(NC)"
	kubectl get pods -n microservices
	kubectl get services -n microservices
	kubectl get deployments -n microservices

k8s-rollout:
	@echo "$(BLUE)☸️  Rolling out deployments...$(NC)"
	@for deployment in $$(kubectl get deployments -n microservices -o jsonpath='{.items[*].metadata.name}'); do \
		echo "Rolling out $$deployment..."; \
		kubectl rollout restart deployment/$$deployment -n microservices; \
	done
	@echo "$(GREEN)✅ Rollout completed$(NC)"

k8s-rollback:
	@echo "$(YELLOW)⚠️  Rolling back deployments...$(NC)"
	@for deployment in $$(kubectl get deployments -n microservices -o jsonpath='{.items[*].metadata.name}'); do \
		echo "Rolling back $$deployment..."; \
		kubectl rollout undo deployment/$$deployment -n microservices; \
	done
	@echo "$(GREEN)✅ Rollback completed$(NC)"

# 🕸️  Istio Service Mesh
istio-install:
	@echo "$(BLUE)🕸️  Installing Istio...$(NC)"
	istioctl install --set values.defaultRevision=default -y
	kubectl label namespace microservices istio-injection=enabled --overwrite
	@echo "$(GREEN)✅ Istio installed$(NC)"

istio-deploy:
	@echo "$(BLUE)🕸️  Deploying Istio configurations...$(NC)"
	kubectl apply -f istio/
	@echo "$(GREEN)✅ Istio configurations deployed$(NC)"

istio-status:
	@echo "$(BLUE)🕸️  Checking Istio status...$(NC)"
	istioctl proxy-status
	istioctl analyze

# 📊 Monitoring & Observability
monitoring-deploy:
	@echo "$(BLUE)📊 Deploying monitoring stack...$(NC)"
	kubectl apply -f monitoring/monitoring-stack.yaml
	@echo "$(GREEN)✅ Monitoring stack deployed$(NC)"

monitoring-port-forward:
	@echo "$(BLUE)📊 Setting up port forwarding for monitoring...$(NC)"
	kubectl port-forward -n microservices service/prometheus 9090:9090 &
	kubectl port-forward -n microservices service/jaeger 16686:16686 &
	kubectl port-forward -n microservices service/grafana 3000:3000 &
	@echo "$(GREEN)✅ Port forwarding setup completed$(NC)"
	@echo "Prometheus: http://localhost:9090"
	@echo "Jaeger: http://localhost:16686"
	@echo "Grafana: http://localhost:3000"

# 🚀 Development Environment
dev-setup: deps install-tools proto-gen
	@echo "$(GREEN)✅ Development environment setup completed$(NC)"

dev-up:
	@echo "$(BLUE)🚀 Starting development environment...$(NC)"
	docker-compose up -d
	@echo "$(GREEN)✅ Development environment started$(NC)"
	@echo "API Gateway: http://localhost:8080"
	@echo "Prometheus: http://localhost:9090"
	@echo "Jaeger: http://localhost:16686"
	@echo "Grafana: http://localhost:3000"

dev-down:
	@echo "$(BLUE)🛑 Stopping development environment...$(NC)"
	docker-compose down
	@echo "$(GREEN)✅ Development environment stopped$(NC)"

dev-restart: dev-down dev-up
	@echo "$(GREEN)✅ Development environment restarted$(NC)"

dev-logs:
	docker-compose logs -f

dev-logs-%:
	docker-compose logs -f $*

dev-shell-%:
	docker-compose exec $* /bin/sh

# 📈 Performance & Load Testing
load-test:
	@echo "$(BLUE)📈 Running load tests...$(NC)"
	@if command -v k6 >/dev/null 2>&1; then \
		k6 run tests/load/api_gateway_test.js; \
	else \
		echo "$(RED)❌ k6 not installed. Please install k6 for load testing$(NC)"; \
	fi

stress-test:
	@echo "$(BLUE)💪 Running stress tests...$(NC)"
	@if command -v stress >/dev/null 2>&1; then \
		stress --cpu 4 --timeout 60s; \
	else \
		echo "$(RED)❌ stress not installed$(NC)"; \
	fi

# 🔧 Database Operations
db-migrate:
	@echo "$(BLUE)🔧 Running database migrations...$(NC)"
	@for service in user-service order-service product-service payment-service notification-service; do \
		echo "Migrating $$service database..."; \
		kubectl exec -n microservices deploy/$$service -- /app/bin/$$service migrate || true; \
	done
	@echo "$(GREEN)✅ Database migrations completed$(NC)"

db-seed:
	@echo "$(BLUE)🌱 Seeding databases...$(NC)"
	kubectl exec -n microservices deploy/postgres -- psql -U postgres -f /docker-entrypoint-initdb.d/seed.sql || true
	@echo "$(GREEN)✅ Database seeding completed$(NC)"

# 📊 Reporting
report-coverage:
	@echo "$(BLUE)📊 Generating coverage report...$(NC)"
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) ./...
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "$(GREEN)✅ Coverage report: $(COVERAGE_HTML)$(NC)"

report-dependencies:
	@echo "$(BLUE)📊 Generating dependency report...$(NC)"
	$(GOMOD) graph > dependencies.txt
	@echo "$(GREEN)✅ Dependency report: dependencies.txt$(NC)"

# 🧼 Maintenance
tidy:
	@echo "$(BLUE)🧼 Tidying up...$(NC)"
	$(GOMOD) tidy
	$(GOFMT) -s -w .
	@echo "$(GREEN)✅ Tidy completed$(NC)"

update-version:
	@echo "$(BLUE)🔄 Updating version to $(VERSION)...$(NC)"
	@sed -i.bak 's/version: .*/version: $(VERSION)/' k8s/**/*.yaml
	@rm -f k8s/**/*.yaml.bak
	@echo "$(GREEN)✅ Version updated to $(VERSION)$(NC)"

# 🆘 Help
help:
	@echo "$(BLUE)🚀 Distributed Microservices Platform - Available Commands$(NC)"
	@echo ""
	@echo "$(YELLOW)📦 Dependencies & Setup:$(NC)"
	@echo "  deps             - Install Go dependencies"
	@echo "  deps-update      - Update Go dependencies"
	@echo "  install-tools    - Install development tools"
	@echo "  dev-setup        - Complete development setup"
	@echo ""
	@echo "$(YELLOW)🔧 Code Generation:$(NC)"
	@echo "  proto-gen        - Generate protobuf files"
	@echo ""
	@echo "$(YELLOW)🏗️  Building:$(NC)"
	@echo "  build            - Build all services"
	@echo "  build-<service>  - Build specific service"
	@echo "  build-linux      - Build all services for Linux"
	@echo ""
	@echo "$(YELLOW)🧪 Testing:$(NC)"
	@echo "  test             - Run unit tests"
	@echo "  test-coverage    - Run tests with coverage"
	@echo "  test-integration - Run integration tests"
	@echo "  test-e2e         - Run end-to-end tests"
	@echo "  test-all         - Run all tests"
	@echo "  bench            - Run benchmarks"
	@echo ""
	@echo "$(YELLOW)🔍 Code Quality:$(NC)"
	@echo "  fmt              - Format Go code"
	@echo "  lint             - Run linters"
	@echo "  vet              - Run go vet"
	@echo "  security-scan    - Run security scan"
	@echo "  code-quality     - Run all code quality checks"
	@echo ""
	@echo "$(YELLOW)🐳 Docker:$(NC)"
	@echo "  docker-build     - Build all Docker images"
	@echo "  docker-push      - Push all Docker images"
	@echo "  docker-scan      - Scan Docker images for vulnerabilities"
	@echo "  build-all        - Build and push all Docker images"
	@echo ""
	@echo "$(YELLOW)☸️  Kubernetes:$(NC)"
	@echo "  k8s-validate     - Validate Kubernetes manifests"
	@echo "  k8s-deploy       - Deploy to Kubernetes"
	@echo "  k8s-status       - Check deployment status"
	@echo "  k8s-rollout      - Rolling update deployments"
	@echo "  k8s-rollback     - Rollback deployments"
	@echo ""
	@echo "$(YELLOW)🕸️  Istio:$(NC)"
	@echo "  istio-install    - Install Istio service mesh"
	@echo "  istio-deploy     - Deploy Istio configurations"
	@echo "  istio-status     - Check Istio status"
	@echo ""
	@echo "$(YELLOW)📊 Monitoring:$(NC)"
	@echo "  monitoring-deploy      - Deploy monitoring stack"
	@echo "  monitoring-port-forward - Setup port forwarding"
	@echo ""
	@echo "$(YELLOW)🚀 Development:$(NC)"
	@echo "  dev-up           - Start development environment"
	@echo "  dev-down         - Stop development environment"
	@echo "  dev-restart      - Restart development environment"
	@echo "  dev-logs         - View all service logs"
	@echo "  dev-logs-<service> - View specific service logs"
	@echo ""
	@echo "$(YELLOW)🧹 Maintenance:$(NC)"
	@echo "  clean            - Clean build artifacts"
	@echo "  tidy             - Tidy up code and dependencies"
	@echo "  update-version   - Update version in manifests"
	@echo ""
	@echo "$(YELLOW)📈 Performance:$(NC)"
	@echo "  load-test        - Run load tests"
	@echo "  stress-test      - Run stress tests"
	@echo ""
	@echo "$(GREEN)For more information, check the README.md file$(NC)"