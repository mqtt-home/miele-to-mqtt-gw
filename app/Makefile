GO = go
GOFLAGS = -v

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')
LDFLAGS = -ldflags "-s -w -X 'github.com/mqtt-home/miele-to-mqtt-gw/version.Version=$(VERSION)' -X 'github.com/mqtt-home/miele-to-mqtt-gw/version.GitCommit=$(GIT_COMMIT)' -X 'github.com/mqtt-home/miele-to-mqtt-gw/version.BuildTime=$(BUILD_TIME)'"

BUILD_DIR = build
CONFIG_DIR = $(PWD)/../production/config

BINARY_NAME = miele2mqtt
DOCKER_IMAGE_NAME = pharndt/mielemqtt:latest

.PHONY: build
build:
	@echo "Building the backend..."
	@$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

.PHONY: test
test:
	@echo "Running tests..."
	@$(GO) test $(GOFLAGS) ./...

.PHONY: coverage
coverage:
	@echo "Running tests with coverage..."
	@$(GO) test -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: lint
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

.PHONY: vet
vet:
	@echo "Running go vet..."
	@$(GO) vet ./...

.PHONY: dev
dev: build
	@echo "Starting dev server..."
	@$(BUILD_DIR)/$(BINARY_NAME) $(CONFIG_DIR)/config.json

.PHONY: run
run: build
	@echo "Running the application..."
	@$(BUILD_DIR)/$(BINARY_NAME) $(CONFIG_DIR)/config.json

.PHONY: image
image:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE_NAME) .

.PHONY: docker
docker: image

.PHONY: clean
clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

.PHONY: deps
deps:
	@echo "Downloading Go dependencies..."
	@$(GO) mod download
	@$(GO) mod tidy

.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
