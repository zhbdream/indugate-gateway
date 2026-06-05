.PHONY: help build run test lint fmt clean docker-build docker-up docker-down deps tidy

APP_NAME    := gateway
MAIN_PATH   := ./cmd/gateway
BUILD_DIR   := ./bin
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME  := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS     := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -w -s"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

deps: ## Download Go module dependencies
	go mod download

tidy: ## Tidy Go module dependencies
	go mod tidy

build: ## Build the gateway binary
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Built $(BUILD_DIR)/$(APP_NAME)"

run: ## Run the gateway locally
	go run $(MAIN_PATH)

test: ## Run unit tests
	go test -v -race -cover ./...

lint: ## Run golangci-lint (requires golangci-lint installed)
	golangci-lint run ./...

fmt: ## Format Go source code
	go fmt ./...
	gofmt -s -w .

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR) data/*.db

docker-build: ## Build Docker image
	docker build -f deployments/docker/Dockerfile -t indugate/gateway:$(VERSION) .

docker-up: ## Start all-in-one via Docker Compose (recommended)
	docker compose up -d --build

docker-up-full: ## Start full stack (Gateway + PostgreSQL + InfluxDB + Mosquitto)
	docker compose -f deployments/docker/docker-compose.yml up -d --build

docker-down: ## Stop Docker Compose services
	docker compose down

docker-down-full: ## Stop full stack
	docker compose -f deployments/docker/docker-compose.yml down

docker-logs: ## Tail container logs
	docker compose logs -f

web-dev: ## Start frontend dev server
	cd web && npm run dev

web-build: ## Build frontend for production
	cd web && npm run build

dev: deps run ## Install deps and run locally

.DEFAULT_GOAL := help
