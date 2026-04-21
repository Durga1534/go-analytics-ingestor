.PHONY: help build run dev test clean lint fmt

help:
	@echo "Available commands:"
	@echo "  make build     - Build the application"
	@echo "  make run       - Build and run the application"
	@echo "  make dev       - Run in development mode (with local reloading via air)"
	@echo "  make test      - Run all tests"
	@echo "  make clean     - Clean build artifacts"
	@echo "  make lint      - Run linter (golangci-lint)"
	@echo "  make fmt       - Format code (gofmt)"
	@echo "  make tidy      - Tidy Go module dependencies"

build:
	@echo "Building go-analytics-ingestor..."
	go build -o ./bin/server ./cmd/server

run: build
	@echo "Running application..."
	./bin/server

dev:
	@echo "Running in development mode (requires 'air')..."
	@command -v air >/dev/null 2>&1 || { echo "Installing air..."; go install github.com/cosmtrek/air@latest; }
	air

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	@echo "Cleaning build artifacts..."
	rm -rf ./bin
	rm -f coverage.out coverage.html

lint:
	@echo "Running linter..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Installing golangci-lint..."; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; }
	golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

tidy:
	@echo "Tidying Go module dependencies..."
	go mod tidy

docker-build:
	@echo "Building Docker image..."
	docker build -t go-analytics-ingestor:latest .

docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file .env go-analytics-ingestor:latest
