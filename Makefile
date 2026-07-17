.PHONY: build run test clean docker-build docker-run

# Binary name
BINARY_NAME=ticket-system
BUILD_DIR=bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server/main.go

# Run the application
run:
	$(GORUN) cmd/server/main.go

# Run tests
test:
	$(GOTEST) -v ./...

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Build Docker image
docker-build:
	docker build -t $(BINARY_NAME) .

# Run Docker container
docker-run:
	docker run -p 8080:8080 -v ticket_data:/data -e DB_PATH=/data/tickets.db $(BINARY_NAME)

# Format code
fmt:
	gofmt -s -w .

# Lint code
lint:
	golangci-lint run ./...
