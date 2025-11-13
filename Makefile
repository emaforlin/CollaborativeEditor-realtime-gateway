# Makefile for Collaborative Editor WebSocket Gateway

# Variables
APP_NAME := collaborative-editor-gateway
DOCKER_IMAGE := $(APP_NAME):latest
DOCKER_TAG := $(shell git rev-parse --short HEAD 2>/dev/null || echo "latest")
DOCKER_REGISTRY := # Add your registry here, e.g., your-registry.com/

.PHONY: help build run stop clean test docker-build docker-run docker-push dev prod logs

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Local development
build: ## Build the Go application locally
	@echo "Building $(APP_NAME)..."
	go build -o gateway main.go

run: build ## Run the application locally
	@echo "Starting $(APP_NAME)..."
	./gateway

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

clean: ## Clean build artifacts
	@echo "Cleaning up..."
	rm -f gateway
	docker system prune -f

# Docker operations
docker-build: ## Build Docker image
	@echo "Building Docker image $(DOCKER_IMAGE)..."
	docker build -t $(DOCKER_IMAGE) -t $(APP_NAME):$(DOCKER_TAG) .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -p 9001:9001 --name $(APP_NAME) $(DOCKER_IMAGE)

docker-push: docker-build ## Push Docker image to registry
	@if [ -z "$(DOCKER_REGISTRY)" ]; then \
		echo "Error: DOCKER_REGISTRY not set"; \
		exit 1; \
	fi
	docker tag $(DOCKER_IMAGE) $(DOCKER_REGISTRY)$(DOCKER_IMAGE)
	docker tag $(APP_NAME):$(DOCKER_TAG) $(DOCKER_REGISTRY)$(APP_NAME):$(DOCKER_TAG)
	docker push $(DOCKER_REGISTRY)$(DOCKER_IMAGE)
	docker push $(DOCKER_REGISTRY)$(APP_NAME):$(DOCKER_TAG)

# Docker Compose operations
dev: ## Start development environment with docker-compose
	@echo "Starting development environment..."
	docker-compose up --build

prod: ## Start production environment with nginx proxy
	@echo "Starting production environment..."
	docker-compose --profile production up --build -d

stop: ## Stop docker-compose services
	@echo "Stopping services..."
	docker-compose down

logs: ## Show logs from docker-compose services
	docker-compose logs -f

# Utility commands
shell: ## Open shell in running container
	docker exec -it $(APP_NAME) /bin/sh

inspect: ## Inspect the Docker image
	docker inspect $(DOCKER_IMAGE)

size: ## Show Docker image size
	docker images $(APP_NAME) --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"

# Security scanning (requires trivy)
scan: docker-build ## Scan Docker image for vulnerabilities
	@which trivy > /dev/null || (echo "Install trivy first: https://github.com/aquasecurity/trivy"; exit 1)
	trivy image $(DOCKER_IMAGE)

# Health check
health: ## Check application health
	@echo "Checking application health..."
	@curl -f http://localhost:9001/health || echo "Health check failed"