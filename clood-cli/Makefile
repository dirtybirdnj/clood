# clood Makefile
# Build, install, and manage clood CLI

BINARY_NAME=clood
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

.PHONY: all build install clean test lint help

all: build

## build: Build the clood binary
build:
	@echo "Building clood $(VERSION)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/clood
	@echo "Built: ./$(BINARY_NAME)"

## install: Install clood to /usr/local/bin
install: build
	@echo "Installing to /usr/local/bin/$(BINARY_NAME)..."
	sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "Installed. Run 'clood --help' to get started."

## install-user: Install to ~/bin (no sudo required)
install-user: build
	@mkdir -p ~/bin
	cp $(BINARY_NAME) ~/bin/
	@echo "Installed to ~/bin/$(BINARY_NAME)"
	@echo "Make sure ~/bin is in your PATH"

## clean: Remove build artifacts
clean:
	rm -f $(BINARY_NAME)
	go clean

## test: Run tests
test:
	go test -v ./...

## lint: Run linter
lint:
	@which golangci-lint > /dev/null || (echo "Install golangci-lint first" && exit 1)
	golangci-lint run

## deps: Download dependencies
deps:
	go mod download
	go mod tidy

## cross: Build for multiple platforms
cross:
	@echo "Building for multiple platforms..."
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 ./cmd/clood
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 ./cmd/clood
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/clood
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 ./cmd/clood
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe ./cmd/clood
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-arm64.exe ./cmd/clood
	@echo "Built binaries for darwin/linux/windows on arm64/amd64"

## windows: Build Windows binaries only
windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe ./cmd/clood
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-arm64.exe ./cmd/clood
	@echo "Built: $(BINARY_NAME)-windows-amd64.exe, $(BINARY_NAME)-windows-arm64.exe"

## quick-test: Quick functionality test
quick-test: build
	@echo "=== Quick Test ==="
	./$(BINARY_NAME) --version
	./$(BINARY_NAME) system
	./$(BINARY_NAME) hosts
	./$(BINARY_NAME) grep "func.*Cmd" internal/commands/ --files-only
	@echo "=== All tests passed ==="

## help: Show this help
help:
	@echo "clood Makefile targets:"
	@echo ""
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/  /'
	@echo ""
	@echo "Quick start:"
	@echo "  make build      # Build the binary"
	@echo "  make install    # Install to /usr/local/bin"
	@echo "  clood --help    # Get started"
