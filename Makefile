.PHONY: build test clean all run download-test-videos

# Build variables
BINARY_NAME=compressvideo

all: clean test build

build:
	@echo "Building ${BINARY_NAME}..."
	go build -o ${BINARY_NAME} cmd/compressvideo/main.go

build-all: clean
	@echo "Building ${BINARY_NAME} for multiple platforms..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/${BINARY_NAME}-linux-amd64 cmd/compressvideo/main.go
	GOOS=windows GOARCH=amd64 go build -o bin/${BINARY_NAME}-windows-amd64.exe cmd/compressvideo/main.go
	GOOS=darwin GOARCH=amd64 go build -o bin/${BINARY_NAME}-darwin-amd64 cmd/compressvideo/main.go

test:
	@echo "Running tests..."
	go test ./...

clean:
	@echo "Cleaning up..."
	@rm -f ${BINARY_NAME}
	@rm -rf bin/

run: build
	@echo "Running ${BINARY_NAME}..."
	./${BINARY_NAME}

download-test-videos:
	@echo "Downloading test videos to data/ directory..."
	@mkdir -p data
	@python3 scripts/download_test_videos.py

help:
	@echo "Available commands:"
	@echo "  make build                - Build the application"
	@echo "  make build-all            - Build for multiple platforms"
	@echo "  make test                 - Run tests"
	@echo "  make clean                - Clean up build artifacts"
	@echo "  make run                  - Build and run the application"
	@echo "  make download-test-videos - Download sample videos for testing"
	@echo "  make all                  - Clean, test, and build"
	@echo "  make help                 - Show this help message" 