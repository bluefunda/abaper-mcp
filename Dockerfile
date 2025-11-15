# Multi-stage Dockerfile for ABAPER MCP Server
# Supports multi-architecture builds (amd64, arm64)

# Build arguments
ARG GO_VERSION=1.23
ARG ALPINE_VERSION=3.20

# Stage 1: Build the Go binary
FROM golang:${GO_VERSION}-alpine AS builder

# Build arguments for version information
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT
ARG TARGETARCH

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with version information
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w \
    -X main.Version=${VERSION} \
    -X main.BuildTime=${BUILD_TIME} \
    -X main.GitCommit=${GIT_COMMIT}" \
    -o abaper-mcp .

# Stage 2: Create minimal runtime image
FROM alpine:${ALPINE_VERSION}

# Build arguments for metadata
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT
ARG TARGETARCH

# Labels following OCI image spec
LABEL org.opencontainers.image.title="ABAPER MCP Server"
LABEL org.opencontainers.image.description="Model Context Protocol server for SAP ABAP development"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.created="${BUILD_TIME}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.opencontainers.image.source="https://github.com/bluefunda/abaper-mcp"
LABEL org.opencontainers.image.documentation="https://github.com/bluefunda/abaper-mcp/blob/main/README.md"
LABEL org.opencontainers.image.vendor="BlueFunda, Inc."
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.authors="BlueFunda, Inc. <info@bluefunda.com>"
LABEL com.bluefunda.architecture="${TARGETARCH}"

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    dumb-init \
    su-exec

# Create non-root user
RUN addgroup -g 1000 abaper && \
    adduser -D -u 1000 -G abaper abaper

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/abaper-mcp /app/abaper-mcp

# Create directories for logs, config, and NATS credentials
RUN mkdir -p /var/log/abaper /etc/abaper /var/nats/creds && \
    chown -R abaper:abaper /app /var/log/abaper /etc/abaper /var/nats/creds

# Create startup script
RUN echo '#!/bin/sh' > /app/entrypoint.sh && \
    echo 'echo "========================================="' >> /app/entrypoint.sh && \
    echo 'echo "ABAPER MCP Server"' >> /app/entrypoint.sh && \
    echo 'echo "Version: '"${VERSION}"'"' >> /app/entrypoint.sh && \
    echo 'echo "Build Time: '"${BUILD_TIME}"'"' >> /app/entrypoint.sh && \
    echo 'echo "Git Commit: '"${GIT_COMMIT}"'"' >> /app/entrypoint.sh && \
    echo 'echo "Architecture: '"${TARGETARCH}"'"' >> /app/entrypoint.sh && \
    echo 'echo "========================================="' >> /app/entrypoint.sh && \
    echo 'echo ""' >> /app/entrypoint.sh && \
    echo 'exec su-exec abaper "$@"' >> /app/entrypoint.sh && \
    chmod +x /app/entrypoint.sh

# Volumes for persistent data
VOLUME ["/var/log/abaper", "/etc/abaper", "/var/nats/creds"]

# Expose stdio for MCP communication
# MCP servers communicate via stdin/stdout, no ports needed
# But we can add a health check port if needed in the future

# Health check - MCP servers don't typically have HTTP endpoints
# This is placeholder for future monitoring capabilities
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD /app/abaper-mcp --version || exit 1

# Use dumb-init to handle signals properly
ENTRYPOINT ["dumb-init", "--", "/app/entrypoint.sh"]

# Default command - run the MCP server
CMD ["/app/abaper-mcp"]
