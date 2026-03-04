# CZManager Agent Makefile
# Cross-compilation for Windows and Linux

BINARY_NAME=czmanager-agent
VERSION=1.0.0
BUILD_DIR=build

# Go build flags for smaller binary
LDFLAGS=-s -w -X main.Version=$(VERSION)

.PHONY: all clean windows linux windows-amd64 linux-amd64 linux-arm64

all: windows linux

clean:
	rm -rf $(BUILD_DIR)

# Windows builds
windows: windows-amd64

windows-amd64:
	@echo "Building for Windows AMD64..."
	@mkdir -p $(BUILD_DIR)/windows-amd64
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/windows-amd64/$(BINARY_NAME).exe .
	@echo "Output: $(BUILD_DIR)/windows-amd64/$(BINARY_NAME).exe"

# Linux builds
linux: linux-amd64 linux-arm64

linux-amd64:
	@echo "Building for Linux AMD64..."
	@mkdir -p $(BUILD_DIR)/linux-amd64
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-amd64/$(BINARY_NAME) .
	@echo "Output: $(BUILD_DIR)/linux-amd64/$(BINARY_NAME)"

linux-arm64:
	@echo "Building for Linux ARM64 (Steam Deck)..."
	@mkdir -p $(BUILD_DIR)/linux-arm64
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/linux-arm64/$(BINARY_NAME) .
	@echo "Output: $(BUILD_DIR)/linux-arm64/$(BINARY_NAME)"

# Build for current platform (development)
dev:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

# Run for development
run:
	go run .

# Download dependencies
deps:
	go mod download
	go mod tidy

# Check code
check:
	go vet ./...
	go fmt ./...
