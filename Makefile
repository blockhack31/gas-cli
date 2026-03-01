.PHONY: build install clean test lint help

BINARY_NAME=gascli
VERSION?=2.0.0
BUILD_DIR=dist
GO=go
GOFLAGS=-ldflags="-s -w -X github.com/calghar/gas-cli/cmd.version=$(VERSION)"

## help: Show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: Build the binary to ./bin (project directory)
build:
	@echo "Building $(BINARY_NAME) v$(VERSION)..."
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -o bin/$(BINARY_NAME) .
	@echo "Build complete: ./bin/$(BINARY_NAME)"

## build-all: Build binaries for all platforms
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "All builds complete in $(BUILD_DIR)/"

## install: Build and install to Go bin (in PATH)
install: build
	@BIN_DIR=$$($(GO) env GOBIN); \
	[ -n "$$BIN_DIR" ] || BIN_DIR=$$($(GO) env GOPATH)/bin; \
	mkdir -p "$$BIN_DIR"; \
	cp bin/$(BINARY_NAME) "$$BIN_DIR/$(BINARY_NAME)"; \
	echo "Installed to $$BIN_DIR/$(BINARY_NAME)"

## test: Run tests
test:
	$(GO) test -v -race -coverprofile=coverage.out ./...

## lint: Run golangci-lint
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, install from https://golangci-lint.run/" && exit 1)
	golangci-lint run ./...

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out
	@echo "Clean complete"

## deps: Download dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

## fmt: Format code
fmt:
	$(GO) fmt ./...
	gofmt -s -w .

.DEFAULT_GOAL := help
