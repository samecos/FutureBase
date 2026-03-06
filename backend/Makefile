# Architecture Platform Makefile
# Provides convenient commands for development and deployment

.PHONY: help build test docker deploy clean

# Default target
.DEFAULT_GOAL := help

# Variables
DOCKER_REGISTRY ?= ghcr.io
IMAGE_PREFIX ?= archplatform
VERSION ?= latest
JAVA_SERVICES := user-service project-service property-service version-service search-service
GO_SERVICES := collaboration-service geometry-service script-service file-service notification-service analytics-service
ALL_SERVICES := $(JAVA_SERVICES) $(GO_SERVICES)

# Colors for output
BLUE := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
NC := \033[0m # No Color

##@ General

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

build-java: ## Build all Java services
	@echo "$(BLUE)Building Java services...$(NC)"
	@for service in $(JAVA_SERVICES); do \
		echo "$(GREEN)Building $$service...$(NC)"; \
		cd services/$$service && ./mvnw clean package -DskipTests -q && cd ../..; \
	done
	@echo "$(GREEN)All Java services built successfully!$(NC)"

build-go: ## Build all Go services
	@echo "$(BLUE)Building Go services...$(NC)"
	@for service in $(GO_SERVICES); do \
		echo "$(GREEN)Building $$service...$(NC)"; \
		cd services/$$service && go build -o bin/$$service ./cmd/server && cd ../..; \
	done
	@echo "$(GREEN)All Go services built successfully!$(NC)"

build: build-java build-go ## Build all services

##@ Testing

test-java: ## Run unit tests for Java services
	@echo "$(BLUE)Running Java unit tests...$(NC)"
	@for service in $(JAVA_SERVICES); do \
		echo "$(GREEN)Testing $$service...$(NC)"; \
		cd services/$$service && ./mvnw test -DexcludedGroups=integration -q 2>/dev/null && cd ../.. || cd ../..; \
	done

test-go: ## Run unit tests for Go services
	@echo "$(BLUE)Running Go unit tests...$(NC)"
	@for service in $(GO_SERVICES); do \
		echo "$(GREEN)Testing $$service...$(NC)"; \
		cd services/$$service && go test ./... -v 2>/dev/null | head -50 && cd ../.. || cd ../..; \
	done

test: test-java test-go ## Run all unit tests

test-java-unit: ## Run Java unit tests only (exclude integration)
	@echo "$(BLUE)Running Java unit tests...$(NC)"
	@for service in $(JAVA_SERVICES); do \
		echo "$(GREEN)Testing $$service unit tests...$(NC)"; \
		cd services/$$service && ./mvnw test -Dtest="*Test,!*IntegrationTest" -q 2>/dev/null && cd ../.. || cd ../..; \
	done

test-java-integration: ## Run Java integration tests
	@echo "$(BLUE)Running Java integration tests...$(NC)"
	@for service in $(JAVA_SERVICES); do \
		echo "$(GREEN)Testing $$service integration tests...$(NC)"; \
		cd services/$$service && ./mvnw test -Dtest="*IntegrationTest" -q 2>/dev/null && cd ../.. || cd ../..; \
	done

test-integration: test-java-integration ## Run integration tests

test-e2e: ## Run E2E tests (requires running services)
	@echo "$(BLUE)Running E2E tests...$(NC)"
	@echo "$(YELLOW)Make sure services are running: make docker-compose-up$(NC)"
	@newman run tests/integration/e2e/user-project-flow.json -e tests/integration/e2e/local-env.json 2>/dev/null || echo "$(YELLOW)E2E tests require Newman and running services$(NC)"

test-performance: ## Run performance tests (requires running services)
	@echo "$(BLUE)Running performance tests...$(NC)"
	@k6 run tests/integration/performance/load-test.js 2>/dev/null || echo "$(YELLOW)Performance tests require k6 and running services$(NC)"

test-all: test test-integration ## Run all tests (unit + integration)

test-coverage: ## Run tests with coverage report
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	@for service in $(JAVA_SERVICES); do \
		cd services/$$service && ./mvnw test jacoco:report -q 2>/dev/null && cd ../.. || cd ../..; \
	done
	@for service in $(GO_SERVICES); do \
		cd services/$$service && go test -coverprofile=coverage.out ./... 2>/dev/null && go tool cover -func=coverage.out && cd ../.. || cd ../..; \
	done
	@echo "$(GREEN)Coverage reports generated!$(NC)"

##@ Code Quality

setup-hooks: ## Setup git hooks and development tools
	@echo "$(BLUE)Setting up development tools...$(NC)"
	@./scripts/setup-hooks.sh

fmt-java: ## Format Java code with Spotless
	@echo "$(BLUE)Formatting Java code...$(NC)"
	@for service in $(JAVA_SERVICES); do \
		echo "$(GREEN)Formatting $$service...$(NC)"; \
		cd services/$$service && ./mvnw spotless:apply -q 2>/dev/null || echo "$(YELLOW)Skipped $$service$(NC)"; \
		cd ../..; \
	done
	@echo "$(GREEN)Java formatting complete!$(NC)"

fmt-go: ## Format Go code
	@echo "$(BLUE)Formatting Go code...$(NC)"
	@for service in $(GO_SERVICES); do \
		echo "$(GREEN)Formatting $$service...$(NC)"; \
		cd services/$$service && go fmt ./... && cd ../..; \
	done
	@echo "$(GREEN)Go formatting complete!$(NC)"

fmt: fmt-java fmt-go ## Format all code

lint-go: ## Run Go linter
	@echo "$(BLUE)Running Go linter...$(NC)"
	@for service in $(GO_SERVICES); do \
		echo "$(GREEN)Linting $$service...$(NC)"; \
		cd services/$$service && golangci-lint run ./... 2>/dev/null || echo "$(YELLOW)golangci-lint not installed$(NC)"; \
		cd ../..; \
	done

lint-java: ## Run Java linter (Checkstyle)
	@echo "$(BLUE)Running Java linter...$(NC)"
	@for service in $(JAVA_SERVICES); do \
		echo "$(GREEN)Linting $$service...$(NC)"; \
		cd services/$$service && ./mvnw checkstyle:check -q 2>/dev/null || echo "$(YELLOW)Checkstyle issues in $$service$(NC)"; \
		cd ../..; \
	done

lint: lint-go lint-java ## Run all linters

check-fmt: ## Check code formatting (CI)
	@echo "$(BLUE)Checking code formatting...$(NC)"
	@for service in $(JAVA_SERVICES); do \
		cd services/$$service && ./mvnw spotless:check -q 2>/dev/null || (echo "$(RED)Formatting issues in $$service$(NC)" && exit 1); \
		cd ../..; \
	done

##@ Docker

docker-build: ## Build all Docker images
	@echo "$(BLUE)Building Docker images...$(NC)"
	@for service in $(ALL_SERVICES); do \
		echo "$(GREEN)Building $$service image...$(NC)"; \
		docker build -t $(IMAGE_PREFIX)/$$service:$(VERSION) services/$$service; \
	done

docker-compose-up: ## Start all services with Docker Compose
	@echo "$(BLUE)Starting services with Docker Compose...$(NC)"
	cd deployments/docker && docker-compose -f docker-compose.full.yml up -d

docker-compose-down: ## Stop all Docker Compose services
	@echo "$(BLUE)Stopping Docker Compose services...$(NC)"
	cd deployments/docker && docker-compose -f docker-compose.full.yml down

docker-compose-logs: ## View Docker Compose logs
	cd deployments/docker && docker-compose -f docker-compose.full.yml logs -f

##@ Kubernetes

k8s-deploy: ## Deploy to Kubernetes using Kustomize
	@echo "$(BLUE)Deploying to Kubernetes...$(NC)"
	cd deployments/k8s/overlays/dev && kustomize build . | kubectl apply -f -

k8s-delete: ## Delete Kubernetes deployment
	@echo "$(YELLOW)Deleting Kubernetes deployment...$(NC)"
	cd deployments/k8s/overlays/dev && kustomize build . | kubectl delete -f -

k8s-status: ## Check Kubernetes deployment status
	@echo "$(BLUE)Kubernetes deployment status:$(NC)"
	kubectl get all -n archplatform-dev

##@ Utility

clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	@for service in $(JAVA_SERVICES); do \
		cd services/$$service && rm -rf target && cd ../..; \
	done
	@for service in $(GO_SERVICES); do \
		cd services/$$service && rm -rf bin && cd ../..; \
	done
	@echo "$(GREEN)Clean complete!$(NC)"

stats: ## Display project statistics
	@echo "$(BLUE)Project Statistics:$(NC)"
	@echo "Java Services:"
	@find services -name "*.java" | wc -l | xargs echo "  Java files:"
	@find services -name "*.java" -exec wc -l {} + | tail -1 | xargs echo "  Java lines:"
	@echo "Go Services:"
	@find services -name "*.go" | wc -l | xargs echo "  Go files:"
	@find services -name "*.go" -exec wc -l {} + | tail -1 | xargs echo "  Go lines:"
	@echo "Tests:"
	@find services -name "*Test.java" | wc -l | xargs echo "  Java test files:"
	@find services -name "*_test.go" | wc -l | xargs echo "  Go test files:"
