.PHONY: build run test clean install docker release snapshot help

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS = -s -w \
	-X main.Version=$(VERSION) \
	-X main.BuildTime=$(BUILD_TIME) \
	-X main.GitCommit=$(GIT_COMMIT)

# Build the MCP server
build:
	go build -ldflags="$(LDFLAGS)" -o abaper-mcp .

# Run the MCP server
run: build
	./abaper-mcp

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f abaper-mcp
	rm -rf dist/
	go clean

# Install dependencies
install:
	go mod download
	go mod tidy

# Build for multiple platforms
build-all:
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/abaper-mcp-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/abaper-mcp-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/abaper-mcp-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/abaper-mcp-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/abaper-mcp-windows-amd64.exe .

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Build Docker image
docker:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t bluefunda/abaper-mcp:$(VERSION) \
		-t bluefunda/abaper-mcp:latest \
		.

# Build multi-architecture Docker images
docker-buildx:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t bluefunda/abaper-mcp:$(VERSION) \
		-t bluefunda/abaper-mcp:latest \
		--push \
		.

# Create a release using GoReleaser
release:
	@if [ -z "$(GITHUB_TOKEN)" ]; then \
		echo "Error: GITHUB_TOKEN is not set"; \
		exit 1; \
	fi
	goreleaser release --clean

# Create a snapshot release (no git tags required)
snapshot:
	goreleaser release --snapshot --clean --skip=publish

# Validate GoReleaser configuration
validate-release:
	goreleaser check

# Tag a new version
tag:
	@if [ -z "$(TAG)" ]; then \
		echo "Usage: make tag TAG=v1.0.0"; \
		exit 1; \
	fi
	git tag -a $(TAG) -m "Release $(TAG)"
	git push origin $(TAG)
	@echo "Tagged $(TAG) and pushed to origin"

# Show version information
version:
	@echo "Version:    $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

# Show help
help:
	@echo "ABAPER MCP - Makefile Commands"
	@echo ""
	@echo "Building:"
	@echo "  build         - Build the MCP server binary"
	@echo "  build-all     - Build for multiple platforms"
	@echo "  install       - Install dependencies"
	@echo ""
	@echo "Running:"
	@echo "  run           - Build and run the MCP server"
	@echo "  test          - Run tests"
	@echo ""
	@echo "Docker:"
	@echo "  docker        - Build Docker image"
	@echo "  docker-buildx - Build multi-architecture Docker images"
	@echo ""
	@echo "Releasing:"
	@echo "  release       - Create a release using GoReleaser (requires GITHUB_TOKEN)"
	@echo "  snapshot      - Create a snapshot release (no git tag required)"
	@echo "  validate-release - Validate GoReleaser configuration"
	@echo "  tag TAG=v1.0.0 - Create and push a new git tag"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo ""
	@echo "Utilities:"
	@echo "  clean         - Clean build artifacts"
	@echo "  version       - Show version information"
	@echo "  help          - Show this help message"
