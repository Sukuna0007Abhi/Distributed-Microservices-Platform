# Build variables
DOCKER_REGISTRY ?= localhost:5000
VERSION ?= latest
SERVICES = user-service order-service product-service payment-service notification-service api-gateway

# Go variables
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod

# Proto variables
PROTO_DIR = proto
PROTO_OUT_DIR = pkg/proto

.PHONY: all build clean test deps proto-gen docker-build docker-push k8s-deploy

# Default target
all: deps proto-gen build

# Install dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Generate protobuf files
proto-gen:
	@echo "Generating protobuf files..."
	@mkdir -p $(PROTO_OUT_DIR)
	protoc --go_out=$(PROTO_OUT_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT_DIR) --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=$(PROTO_OUT_DIR) --grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=$(PROTO_OUT_DIR) \
		$(PROTO_DIR)/*.proto

# Build all services
build:
	@echo "Building all services..."
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		cd services/$$service && $(GOBUILD) -o bin/$$service ./cmd/main.go && cd ../..; \
	done

# Build specific service
build-%:
	@echo "Building $*..."
	cd services/$* && $(GOBUILD) -o bin/$* ./cmd/main.go

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@for service in $(SERVICES); do \
		rm -rf services/$$service/bin; \
	done
	$(GOCLEAN)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -tags=integration ./tests/integration/...

# Run end-to-end tests
test-e2e:
	@echo "Running e2e tests..."
	$(GOTEST) -v -tags=e2e ./tests/e2e/...

# Docker build all services
docker-build:
	@echo "Building Docker images..."
	@for service in $(SERVICES); do \
		echo "Building Docker image for $$service..."; \
		docker build -t $(DOCKER_REGISTRY)/$$service:$(VERSION) -f services/$$service/Dockerfile .; \
	done

# Docker build specific service
docker-build-%:
	@echo "Building Docker image for $*..."
	docker build -t $(DOCKER_REGISTRY)/$*:$(VERSION) -f services/$*/Dockerfile .

# Push Docker images
docker-push:
	@echo "Pushing Docker images..."
	@for service in $(SERVICES); do \
		echo "Pushing $$service..."; \
		docker push $(DOCKER_REGISTRY)/$$service:$(VERSION); \
	done

# Push specific service
docker-push-%:
	@echo "Pushing $*..."
	docker push $(DOCKER_REGISTRY)/$*:$(VERSION)

# Build and push all
build-all: docker-build docker-push

# Deploy to Kubernetes
k8s-deploy:
	@echo "Deploying to Kubernetes..."
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/configmaps/
	kubectl apply -f k8s/secrets/
	kubectl apply -f k8s/services/
	kubectl apply -f k8s/deployments/

# Deploy Istio configurations
istio-deploy:
	@echo "Deploying Istio configurations..."
	kubectl apply -f istio/

# Deploy monitoring stack
monitoring-deploy:
	@echo "Deploying monitoring stack..."
	kubectl apply -f monitoring/

# Start local development environment
dev-up:
	docker-compose up -d

# Stop local development environment
dev-down:
	docker-compose down

# View logs for all services
logs:
	docker-compose logs -f

# View logs for specific service
logs-%:
	docker-compose logs -f $*

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Install tools
install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

# Help
help:
	@echo "Available targets:"
	@echo "  all              - Build everything"
	@echo "  deps             - Install dependencies"
	@echo "  proto-gen        - Generate protobuf files"
	@echo "  build            - Build all services"
	@echo "  build-<service>  - Build specific service"
	@echo "  clean            - Clean build artifacts"
	@echo "  test             - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-e2e         - Run end-to-end tests"
	@echo "  docker-build     - Build all Docker images"
	@echo "  docker-push      - Push all Docker images"
	@echo "  k8s-deploy       - Deploy to Kubernetes"
	@echo "  istio-deploy     - Deploy Istio configurations"
	@echo "  monitoring-deploy- Deploy monitoring stack"
	@echo "  dev-up           - Start local development"
	@echo "  dev-down         - Stop local development"
	@echo "  logs             - View all service logs"
	@echo "  logs-<service>   - View specific service logs"
	@echo "  fmt              - Format Go code"
	@echo "  lint             - Lint Go code"
	@echo "  install-tools    - Install required tools"