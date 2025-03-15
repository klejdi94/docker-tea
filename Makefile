# Docker Tea Makefile
BINARY_NAME=docker-tea
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# Default target
.PHONY: all
all: build

# Build the application
.PHONY: build
build:
	@echo "Building ${BINARY_NAME}..."
	@go build ${LDFLAGS} -o ${BINARY_NAME} ./cmd/docker-tea

# Run the application
.PHONY: run
run: build
	@echo "Starting ${BINARY_NAME}..."
	@./${BINARY_NAME}

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -f ${BINARY_NAME}
	@go clean

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod verify

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...

# Build for multiple platforms
.PHONY: build-all
build-all: clean
	@echo "Building for multiple platforms..."
	@GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-amd64 ./cmd/docker-tea
	@GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-amd64 ./cmd/docker-tea
	@GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-windows-amd64.exe ./cmd/docker-tea

# Help command
.PHONY: help
help:
	@echo "Docker Tea Makefile Help"
	@echo "------------------------"
	@echo "make            - Build the application"
	@echo "make build      - Build the application"
	@echo "make run        - Build and run the application"
	@echo "make clean      - Remove build artifacts"
	@echo "make deps       - Install dependencies"
	@echo "make test       - Run tests"
	@echo "make build-all  - Build for multiple platforms"
	@echo "make help       - Show this help message" 