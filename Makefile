.PHONY: build run test clean lint

# Project configuration
BINARY_NAME=nexusai-gateway
CMD_DIR=./cmd/gateway

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
