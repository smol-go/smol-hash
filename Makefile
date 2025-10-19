.PHONY: all build test bench clean run examples help

# Default target
all: test build

# Build the main demo
build:
	@echo "Building smol-hash..."
	@go build -o bin/smol-hash cmd/smol-hash/main.go
	@echo "Built: bin/smol-hash"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@go test -cover ./...
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Run the main demo
run: build
	@echo "Running smol-hash demo..."
	@./bin/smol-hash

# Build and run examples
example-basic:
	@echo "Running basic example..."
	@go run examples/basic/main.go

example-loadbalancer:
	@echo "Running load balancer example..."
	@go run examples/loadbalancer/main.go

example-visualize:
	@echo "Running visualizer..."
	@go run examples/visualize/main.go

examples: example-basic example-loadbalancer example-visualize

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@go vet ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "Cleaned"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Run everything
demo: clean test build run examples

# Help
help:
	@echo "smol-hash - Consistent Hashing with Bounded Loads"
	@echo ""
	@echo "Available targets:"
	@echo "  make build              - Build the main demo"
	@echo "  make test               - Run unit tests"
	@echo "  make coverage           - Run tests with coverage report"
	@echo "  make bench              - Run benchmarks"
	@echo "  make run                - Build and run main demo"
	@echo "  make example-basic      - Run basic usage example"
	@echo "  make example-loadbalancer - Run load balancer example"
	@echo "  make example-visualize  - Run ring visualizer"
	@echo "  make examples           - Run all examples"
	@echo "  make fmt                - Format code"
	@echo "  make lint               - Lint code"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make deps               - Install dependencies"
	@echo "  make demo               - Run everything (clean, test, build, run, examples)"
	@echo "  make help               - Show this help"