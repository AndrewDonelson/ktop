# ktop Makefile
# Kubernetes Cluster Monitor TUI

BINARY_NAME=ktop
VERSION=1.0.0
GO=go

# Deploy configuration
DEPLOY_HOST=192.168.1.76
DEPLOY_USER=andrew
DEPLOY_PATH=/usr/local/bin/ktop

# Build flags
LDFLAGS=-ldflags "-s -w -X github.com/nlaak/ktop/internal/config.Version=$(VERSION)"

# Default target
.PHONY: all
all: build

# Build for current platform
.PHONY: build
build:
	$(GO) build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/ktop

# Build for Linux amd64 (most common server target)
.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/ktop

# Build for Linux arm64
.PHONY: build-linux-arm
build-linux-arm:
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 ./cmd/ktop

# macOS builds
.PHONY: build-darwin
build-darwin:
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 ./cmd/ktop
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 ./cmd/ktop

# Windows builds
.PHONY: build-windows
build-windows:
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe ./cmd/ktop

# Build for all platforms
.PHONY: build-all
build-all: build-linux build-linux-arm build-darwin build-windows

# Deploy to cluster master
.PHONY: deploy
deploy: build-linux
	@echo "Deploying ktop to $(DEPLOY_USER)@$(DEPLOY_HOST)..."
	rsync -avz --progress ./$(BINARY_NAME)-linux-amd64 $(DEPLOY_USER)@$(DEPLOY_HOST):~/$(BINARY_NAME)
	ssh -t $(DEPLOY_USER)@$(DEPLOY_HOST) "sudo mv ~/$(BINARY_NAME) $(DEPLOY_PATH) && sudo chmod +x $(DEPLOY_PATH)"
	@echo "Done! Run 'ktop' on $(DEPLOY_HOST)"

# Deploy without sudo (to home bin)
.PHONY: deploy-user
deploy-user: build-linux
	@echo "Deploying ktop to $(DEPLOY_USER)@$(DEPLOY_HOST):~/bin/..."
	ssh $(DEPLOY_USER)@$(DEPLOY_HOST) "mkdir -p ~/bin"
	rsync -avz --progress ./$(BINARY_NAME)-linux-amd64 $(DEPLOY_USER)@$(DEPLOY_HOST):~/bin/$(BINARY_NAME)
	ssh $(DEPLOY_USER)@$(DEPLOY_HOST) "chmod +x ~/bin/$(BINARY_NAME)"
	@echo "Done! Run '~/bin/ktop' on $(DEPLOY_HOST)"

# Build and deploy in one step
.PHONY: ship
ship: deploy

# Run the application
.PHONY: run
run:
	$(GO) run ./cmd/ktop

# Run with flags
.PHONY: run-dev
run-dev:
	$(GO) run ./cmd/ktop -refresh-interval 5s -all-namespaces

# Install to GOPATH/bin
.PHONY: install
install:
	$(GO) install $(LDFLAGS) ./cmd/ktop

# Install to /usr/local/bin (requires sudo)
.PHONY: install-system
install-system: build
	sudo cp $(BINARY_NAME) /usr/local/bin/

# Download dependencies
.PHONY: deps
deps:
	$(GO) mod download
	$(GO) mod tidy

# Verify dependencies
.PHONY: verify
verify:
	$(GO) mod verify

# Run tests
.PHONY: test
test:
	$(GO) test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Lint the code
.PHONY: lint
lint:
	golangci-lint run ./...

# Format the code
.PHONY: fmt
fmt:
	$(GO) fmt ./...

# Vet the code
.PHONY: vet
vet:
	$(GO) vet ./...

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-linux-amd64
	rm -f $(BINARY_NAME)-linux-arm64
	rm -f $(BINARY_NAME)-darwin-amd64
	rm -f $(BINARY_NAME)-darwin-arm64
	rm -f $(BINARY_NAME)-windows-amd64.exe
	rm -f coverage.out coverage.html

# Show help
.PHONY: help
help:
	@echo "ktop - Kubernetes Cluster Monitor"
	@echo ""
	@echo "Usage:"
	@echo "  make              Build for current platform"
	@echo "  make build        Build for current platform"
	@echo "  make build-linux  Build for Linux amd64"
	@echo "  make build-all    Build for all platforms"
	@echo "  make deploy       Build and deploy to cluster master ($(DEPLOY_HOST))"
	@echo "  make deploy-user  Deploy to ~/bin (no sudo required)"
	@echo "  make ship         Alias for deploy"
	@echo "  make run          Run the application"
	@echo "  make install      Install to GOPATH/bin"
	@echo "  make deps         Download dependencies"
	@echo "  make test         Run tests"
	@echo "  make fmt          Format code"
	@echo "  make clean        Clean build artifacts"
	@echo "  make help         Show this help"
	@echo ""
	@echo "Deploy config (edit in Makefile):"
	@echo "  DEPLOY_HOST = $(DEPLOY_HOST)"
	@echo "  DEPLOY_USER = $(DEPLOY_USER)"
	@echo "  DEPLOY_PATH = $(DEPLOY_PATH)"