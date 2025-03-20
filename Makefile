.PHONY: build test clean all run download-test-videos cross-build release package install help sprint4 sprint5

# Build variables
BINARY_NAME=compressvideo
VERSION=$(shell grep -oP 'Version = "\K[^"]+' pkg/util/version.go)
BUILD_DATE=$(shell date +%Y-%m-%d)
LDFLAGS=-ldflags "-X github.com/cccarv82/compressvideo/pkg/util.BuildDate=$(BUILD_DATE)"
DIST_DIR=dist
BIN_DIR=bin
PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

all: clean test build

# Sprint 4: Correções e Melhorias
sprint4: clean test build
	@echo "Sprint 4 (v0.4.0) concluído com sucesso!"
	@echo "Correções e melhorias aplicadas e testadas."

# Sprint 5: Testes e Finalização
sprint5: clean test
	@echo "Preparando Sprint 5 (Testes e Finalização)..."
	@echo "Tarefas pendentes:"
	@echo " - Testes de integração"
	@echo " - Testes de desempenho"
	@echo " - Testes em diferentes sistemas operacionais"
	@echo " - Empacotamento final para distribuição"
	@echo " - Documentação completa"
	@echo " - Atualização para versão 1.0.0"

# Regular build for current platform
build:
	@echo "Building $(BINARY_NAME) v$(VERSION)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) cmd/compressvideo/main.go

# Build for all platforms 
cross-build: clean
	@echo "Building $(BINARY_NAME) v$(VERSION) for multiple platforms..."
	@mkdir -p $(BIN_DIR)
	@echo "Building for Linux (amd64)..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 cmd/compressvideo/main.go
	@echo "Building for Linux (arm64)..."
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 cmd/compressvideo/main.go
	@echo "Building for macOS (amd64)..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 cmd/compressvideo/main.go
	@echo "Building for macOS (arm64)..."
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 cmd/compressvideo/main.go
	@echo "Building for Windows (amd64)..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe cmd/compressvideo/main.go
	@echo "Cross-platform builds complete."

# Create release packages
package: cross-build
	@echo "Creating distribution packages..."
	@mkdir -p $(DIST_DIR)
	@echo "Packaging for Linux (amd64)..."
	tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(BIN_DIR) $(BINARY_NAME)-linux-amd64
	@echo "Packaging for Linux (arm64)..."
	tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz -C $(BIN_DIR) $(BINARY_NAME)-linux-arm64
	@echo "Packaging for macOS (amd64)..."
	tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz -C $(BIN_DIR) $(BINARY_NAME)-darwin-amd64
	@echo "Packaging for macOS (arm64)..."
	tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz -C $(BIN_DIR) $(BINARY_NAME)-darwin-arm64
	@echo "Packaging for Windows (amd64)..."
	zip -j $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe
	@echo "Packaging complete. Files are in the '$(DIST_DIR)' directory."

# Run the tests
test:
	@echo "Running tests..."
	go test ./...

# Install the application locally
install: build
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) ./cmd/compressvideo

# Clean up build artifacts
clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@rm -rf $(BIN_DIR)
	@rm -rf $(DIST_DIR)

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Create a new release
release: clean test cross-build package
	@echo "Release v$(VERSION) created successfully!"
	@echo "Release files:"
	@ls -la $(DIST_DIR)

# Download test videos
download-test-videos:
	@echo "Downloading test videos to data/ directory..."
	@mkdir -p data
	@python3 scripts/download_test_videos.py

# Help information
help:
	@echo "CompressVideo - Smart Video Compression Tool"
	@echo
	@echo "Available commands:"
	@echo "  make build                - Build the application for current platform"
	@echo "  make cross-build          - Build for multiple platforms"
	@echo "  make package              - Create distribution packages"
	@echo "  make test                 - Run tests"
	@echo "  make install              - Install the application locally"
	@echo "  make clean                - Clean up build artifacts"
	@echo "  make run                  - Build and run the application"
	@echo "  make release              - Create a full release (test, build, package)"
	@echo "  make download-test-videos - Download sample videos for testing"
	@echo "  make all                  - Clean, test, and build"
	@echo "  make help                 - Show this help message"
	@echo
	@echo "Version: $(VERSION)" 