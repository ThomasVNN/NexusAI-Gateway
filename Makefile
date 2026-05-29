.PHONY: build run test clean lint bootstrap dev-env-up dev-env-down dev

# Project configuration
BINARY_NAME=nexusai-gateway
CMD_DIR=./cmd/gateway
DOCKER_COMPOSE=docker-compose -f deployments/docker-compose.yml

bootstrap:
	@echo "Checking environment config..."
	@if [ ! -f .env ]; then \
		echo "Copying .env.example to .env..."; \
		cp .env.example .env; \
	else \
		echo ".env already exists."; \
	fi
	@echo "Installing frontend dependencies..."
	@cd web && npm install
	@echo "Building frontend..."
	@cd web && npm run build
	@echo "Bootstrapped successfully!"

dev-env-up:
	@echo "Spinning up local dependencies (Postgres, Redis)..."
	@$(DOCKER_COMPOSE) up -d postgres-nexus redis-nexus
	@echo "Local dependency stack is up and running."

dev-env-down:
	@echo "Stopping local dependency stack..."
	@$(DOCKER_COMPOSE) down -v
	@echo "Local dependency stack stopped."

dev: bootstrap dev-env-up
	@echo "Starting NexusAI-Gateway locally..."
	go run $(CMD_DIR)/main.go

build:
	@echo "Building frontend..."
	@cd web && npm run build || echo "Warning: Frontend build skipped (node/npm not available)"
	@echo "Building Go binary..."
	go build -o bin/$(BINARY_NAME) $(CMD_DIR)/main.go

run:
	go run $(CMD_DIR)/main.go

test:
	go test -v ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/
	rm -rf web/dist/
