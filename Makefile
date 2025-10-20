.PHONY: run build test demo clean docker-up docker-down deps

run:
	@echo "Starting Aegis Gateway..."
	@go run cmd/aegis/main.go

build:
	@echo "Building Aegis Gateway..."
	@go build -o bin/aegis-gateway cmd/aegis/main.go

deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

test:
	@echo "Running tests..."
	@go test -v ./...

demo:
	@echo "Running demo scripts..."
	@chmod +x scripts/demo.sh
	@./scripts/demo.sh

hot-reload-test:
	@echo "Testing hot-reload..."
	@chmod +x scripts/test-hot-reload.sh
	@./scripts/test-hot-reload.sh

docker-up:
	@echo "Starting Docker containers..."
	@cd deploy && docker-compose up --build -d

docker-down:
	@echo "Stopping Docker containers..."
	@cd deploy && docker-compose down

docker-logs:
	@cd deploy && docker-compose logs -f aegis-gateway

clean:
	@echo "Cleaning up..."
	@rm -rf bin/
	@rm -rf logs/*.log
	@go clean

help:
	@echo "Aegis Gateway - Available commands:"
	@echo "  make run              - Run the gateway locally"
	@echo "  make build            - Build the binary"
	@echo "  make deps             - Install/update dependencies"
	@echo "  make demo             - Run the demo script"
	@echo "  make hot-reload-test  - Test policy hot-reload"
	@echo "  make docker-up        - Start with Docker Compose"
	@echo "  make docker-down      - Stop Docker containers"
	@echo "  make docker-logs      - View Docker logs"
	@echo "  make clean            - Clean build artifacts"
