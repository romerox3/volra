BINARY_NAME := volra
VERSION ?= dev
BUILD_DIR := bin
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test test-integration lint build-all checksums clean

build:  ## Build binary for current platform
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/volra

test:  ## Run unit tests (no Docker required)
	go test ./... -v -race -count=1

test-integration:  ## Run integration tests (Docker required)
	go test ./... -v -race -count=1 -tags=integration

lint:  ## Run linters
	golangci-lint run ./...

build-all:  ## Cross-compile for all targets
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/volra
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/volra

checksums:  ## Generate SHA256 checksums
	cd $(BUILD_DIR) && shasum -a 256 $(BINARY_NAME)-* > SHA256SUMS

clean:  ## Remove build artifacts
	rm -rf $(BUILD_DIR)
