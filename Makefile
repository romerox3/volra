BINARY_NAME := volra
VERSION ?= dev
BUILD_DIR := bin
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test test-integration e2e e2e-deploy lint build-all checksums clean

build:  ## Build binary for current platform
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/volra

test:  ## Run unit tests (no Docker required)
	go test ./... -v -race -count=1

test-integration:  ## Run integration tests (Docker required)
	go test ./... -v -race -count=1 -tags=integration

e2e:  ## Run E2E tests Phase 1+2 (no Docker required)
	go test ./tests/e2e/... -v -run 'TestPhase[12]' -tags=e2e -count=1

e2e-deploy:  ## Run E2E tests Phase 3+4 (Docker required)
	VOLRA_E2E_DEPLOY=1 go test ./tests/e2e/... -v -tags=e2e -count=1 -timeout=30m

lint:  ## Run linters
	golangci-lint run ./...

build-all:  ## Cross-compile for all targets
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/volra
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/volra
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/volra

checksums:  ## Generate SHA256 checksums
	cd $(BUILD_DIR) && \
	  if command -v sha256sum > /dev/null 2>&1; then \
	    sha256sum $(BINARY_NAME)-* > SHA256SUMS; \
	  else \
	    shasum -a 256 $(BINARY_NAME)-* > SHA256SUMS; \
	  fi

clean:  ## Remove build artifacts
	rm -rf $(BUILD_DIR)
