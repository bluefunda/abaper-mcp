# SSE/HTTP Integration Guide

This document describes the SSE (Server-Sent Events) HTTP transport implementation for ABAPER MCP server, enabling orchestrator-based deployment for multi-server architectures.

## Overview

ABAPER MCP server now supports SSE/HTTP mode, which allows it to be deployed as an HTTP server that communicates with orchestrators using the Model Context Protocol over HTTP/SSE transport.

### Why SSE Mode?

- **Container Communication**: Docker containers cannot share stdio, but can communicate via HTTP
- **Orchestrator Integration**: Works with orchestrators like cai-mcp-client that manage multiple MCP servers
- **Scalability**: Enables load balancing and horizontal scaling
- **Health Monitoring**: Built-in health check endpoint for monitoring
- **Standard Protocol**: Uses official MCP HTTP/SSE transport from go-sdk

## Architecture

### Communication Flow

```
Browser/UI
    ↓
    ↓ NATS (client-to-orchestrator)
    ↓
cai-mcp-client (Orchestrator)
    ↓
    ↓ HTTP/SSE (orchestrator-to-MCP servers)
    ↓
abaper-mcp (MCP Server on port 8015)
    ↓
    ↓ ADT REST API
    ↓
SAP ABAP System
```

### Transport Protocol

The server uses the **Streamable HTTP Transport** from the official Go MCP SDK:

- **GET /**: Opens SSE stream for server→client messages
- **POST /**: Receives client→server messages
- **GET /health**: Health check endpoint

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ABAPER_MODE` | Operational mode | `stdio` | Yes (set to `sse`) |
| `ABAPER_HTTP_PORT` | HTTP server port | `8015` | No |
| `ABAPER_HTTP_HOST` | Host to bind to | `0.0.0.0` | No |
| `SAP_HOST` | SAP ADT host URL | - | Yes |
| `SAP_CLIENT` | SAP client number | `100` | No |
| `SAP_USERNAME` | SAP username | - | Yes |
| `SAP_PASSWORD` | SAP password | - | Yes |

## Usage

### Local Development

```bash
# Set environment variables
export ABAPER_MODE=sse
export ABAPER_HTTP_PORT=8015
export SAP_HOST=https://sapdev.company.com:8000
export SAP_CLIENT=100
export SAP_USERNAME=DEVELOPER01
export SAP_PASSWORD=SecurePassword123

# Run the server
./abaper-mcp
```

**Expected output:**
```
Starting ABAPER MCP server in sse mode
Running in SSE/HTTP mode (orchestrator compatible via HTTP)
SSE/HTTP MCP server listening on http://0.0.0.0:8015
Health check available at http://0.0.0.0:8015/health
Press Ctrl+C to exit.
```

### Docker Deployment

#### Standalone Container

```bash
docker run -d \
  -e ABAPER_MODE=sse \
  -e ABAPER_HTTP_PORT=8015 \
  -e SAP_HOST=https://saphost:8000 \
  -e SAP_CLIENT=100 \
  -e SAP_USERNAME=user \
  -e SAP_PASSWORD=pass \
  -p 8015:8015 \
  --name abaper-mcp \
  --network trm-network \
  bluefunda/abaper-mcp:latest
```

#### Production Deployment (GitHub Actions)

The production deployment workflow automatically deploys in SSE mode:

```yaml
docker run -d \
  -e SAP_HOST=${{ secrets.SAP_HOST }} \
  -e SAP_CLIENT=${{ secrets.SAP_CLIENT }} \
  -e SAP_USERNAME=${{ secrets.SAP_USERNAME }} \
  -e SAP_PASSWORD=${{ secrets.SAP_PASSWORD }} \
  -e ABAPER_MODE=sse \
  -e ABAPER_HTTP_PORT=8015 \
  -e ABAPER_HTTP_HOST=0.0.0.0 \
  -p 8015:8015 \
  --name abaper-mcp \
  --network trm-network \
  $DOCKER_IMAGE:$VERSION_TAG
```

## Testing

### Health Check

```bash
# Test health endpoint
curl http://localhost:8015/health

# Expected response:
# {"status":"healthy","version":"v0.1.16","mode":"sse"}
```

### Docker Health Check

The Dockerfile includes automatic health checks:

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD if [ "$ABAPER_MODE" = "sse" ]; then \
            curl -f http://localhost:8015/health || exit 1; \
        else \
            /app/abaper-mcp --version || exit 1; \
        fi
```

Check container health:
```bash
docker ps
# Look for "healthy" in STATUS column

docker inspect abaper-mcp | grep -A 5 Health
```

### MCP Protocol Testing

Using the MCP Inspector or similar tools:

```bash
# Connect to SSE endpoint
curl -N http://localhost:8015/

# Should receive SSE stream with session endpoint
```

## Orchestrator Integration

### cai-mcp-client Configuration

The orchestrator expects MCP servers to be available via HTTP/SSE. Configure it to connect to abaper-mcp:

**Python configuration example:**
```python
mcp_servers = {
    "abaper": {
        "mcp_server_url": "http://abaper-mcp:8015/"
    }
}
```

**Note**: Use the Docker container name (`abaper-mcp`) as the hostname when both containers are on the same Docker network (`trm-network`).

### Request Flow

1. **Client Request**: User interacts with web app
2. **Orchestrator Routing**: cai-mcp-client receives request, identifies "abaper" tools
3. **HTTP Connection**: Orchestrator connects to `http://abaper-mcp:8015/`
4. **SSE Stream**: Server opens SSE stream for responses
5. **Tool Execution**: Server executes ABAP operation via ADT
6. **Response**: Results streamed back via SSE to orchestrator
7. **Client Response**: Orchestrator returns results to web app

## Networking

### Docker Network Setup

All containers must be on the same Docker network:

```bash
# Create network (if not exists)
docker network create trm-network

# Verify network
docker network inspect trm-network

# Both containers should show:
# - cai-mcp-client
# - abaper-mcp
```

### Port Mapping

- **Host**: Port 8015 is exposed to the host for external access
- **Container**: Port 8015 inside the container
- **Network**: Containers communicate via `http://abaper-mcp:8015/` (no port mapping needed)

### Firewall Configuration

For production deployments, ensure port 8015 is accessible:

```bash
# Example: Allow port 8015 (firewall-specific)
sudo ufw allow 8015/tcp
```

## Monitoring

### Logs

```bash
# View container logs
docker logs -f abaper-mcp

# Expected output:
# Starting ABAPER MCP server in sse mode
# Running in SSE/HTTP mode (orchestrator compatible via HTTP)
# SSE/HTTP MCP server listening on http://0.0.0.0:8015
# Health check available at http://0.0.0.0:8015/health
```

### Health Monitoring

**Manual check:**
```bash
curl http://localhost:8015/health
```

**Automated monitoring** (Prometheus example):
```yaml
- job_name: 'abaper-mcp'
  metrics_path: '/health'
  static_configs:
    - targets: ['abaper-mcp:8015']
```

### Request Logging

The server logs all incoming requests. Monitor logs for:
- Tool execution requests
- Resource access
- Prompt invocations
- Error messages

## Troubleshooting

### Server Won't Start

**Problem**: Server exits immediately or fails to start

**Check**:
```bash
# View full logs
docker logs abaper-mcp

# Common issues:
# - Port already in use
# - Missing environment variables
# - Invalid SAP credentials
```

**Solutions**:
```bash
# Check port availability
netstat -tuln | grep 8015

# Verify environment variables
docker inspect abaper-mcp | grep -A 20 Env

# Test SAP connection separately
curl -u username:password https://saphost:8000/sap/bc/adt/discovery
```

### Health Check Failing

**Problem**: Docker shows container as "unhealthy"

**Check**:
```bash
# Manual health check
docker exec abaper-mcp curl -f http://localhost:8015/health

# View health check logs
docker inspect abaper-mcp | grep -A 10 Health
```

**Solutions**:
- Ensure server started successfully (check logs)
- Verify port 8015 is listening inside container
- Check `ABAPER_MODE` is set to `sse`

### Orchestrator Cannot Connect

**Problem**: cai-mcp-client cannot reach abaper-mcp

**Check**:
```bash
# Verify both containers on same network
docker network inspect trm-network

# Test connectivity from orchestrator container
docker exec cai-mcp-client curl http://abaper-mcp:8015/health
```

**Solutions**:
```bash
# Ensure both containers on trm-network
docker network connect trm-network abaper-mcp
docker network connect trm-network cai-mcp-client

# Verify container name matches configuration
docker ps | grep abaper-mcp
```

### SSE Stream Issues

**Problem**: SSE connection drops or times out

**Check**:
- Session timeout configuration (default: 30 minutes)
- Network proxy/load balancer timeout settings
- Container resource limits

**Solutions**:
```go
// Adjust session timeout in main.go:
&mcp.StreamableHTTPOptions{
    SessionTimeout: 60 * time.Minute,  // Increase if needed
}
```

## Performance Considerations

### Connection Management

- **Stateless Mode**: Disabled (uses sessions for connection management)
- **Session Timeout**: 30 minutes (configurable)
- **Concurrent Connections**: Limited by Go HTTP server defaults

### Resource Usage

Typical resource usage in SSE mode:
- **Memory**: ~50-100 MB (base) + per-connection overhead
- **CPU**: Minimal when idle, spikes during ABAP operations
- **Network**: Depends on ABAP object size and request frequency

### Scaling

For high-traffic scenarios:
1. **Horizontal Scaling**: Deploy multiple abaper-mcp containers
2. **Load Balancing**: Use nginx/traefik to distribute requests
3. **Connection Pooling**: Orchestrator should reuse connections

## Security Considerations

### Network Security

1. **Internal Network**: Keep abaper-mcp on internal Docker network
2. **TLS/HTTPS**: Use reverse proxy (nginx) for TLS termination
3. **Firewall**: Restrict port 8015 to orchestrator only

### Credential Security

1. **Environment Variables**: Use Docker secrets or vault for passwords
2. **SAP Password**: Never commit to version control
3. **Transport Security**: Always use HTTPS for SAP_HOST

### Access Control

1. **No Authentication**: SSE mode has no built-in auth (relies on network isolation)
2. **Orchestrator Auth**: Ensure cai-mcp-client handles authentication
3. **Container Isolation**: Use Docker user namespacing

## Comparison with Other Modes

| Feature | stdio | sse | nats | dual |
|---------|-------|-----|------|------|
| Use Case | Claude Desktop | Orchestrator | Legacy | Legacy |
| Transport | stdin/stdout | HTTP/SSE | NATS pub/sub | stdio + NATS |
| Port | None | 8015 | None | None |
| Container-to-Container | ❌ No | ✅ Yes | ✅ Yes | ❌ No |
| Health Check | Version cmd | HTTP endpoint | Version cmd | Version cmd |
| Recommended | Desktop only | Production | No | No |

## Migration Guide

### From NATS Mode to SSE Mode

1. **Update environment variables:**
   ```bash
   # Before (NATS mode)
   ABAPER_MODE=nats
   NATS_URL=tls://connect.ngs.global:4222
   NATS_CREDS=/etc/ngs/user.creds

   # After (SSE mode)
   ABAPER_MODE=sse
   ABAPER_HTTP_PORT=8015
   # Remove NATS variables
   ```

2. **Update deployment:**
   ```bash
   # Add port mapping
   -p 8015:8015

   # Remove NATS volume mount
   # -v /home/admin/etc/ngs:/etc/ngs:ro  # REMOVE
   ```

3. **Update orchestrator configuration:**
   ```python
   # Before (NATS)
   # Orchestrator connected via NATS subjects

   # After (SSE)
   mcp_servers = {
       "abaper": {
           "mcp_server_url": "http://abaper-mcp:8015/"
       }
   }
   ```

4. **Verify health:**
   ```bash
   curl http://localhost:8015/health
   ```

## References

- [Model Context Protocol Specification](https://modelcontextprotocol.io)
- [Go MCP SDK Documentation](https://github.com/modelcontextprotocol/go-sdk)
- [Go MCP SDK Streamable Transport](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp#StreamableHTTPHandler)
- [cai-mcp-client Orchestrator](https://github.com/bluefunda/cai-mcp-client)
- [ABAPER Library](https://github.com/bluefunda/abaper)

## Support

For SSE integration issues:
- GitHub Issues: https://github.com/bluefunda/abaper-mcp/issues
- Check logs: `docker logs abaper-mcp`
- Test health: `curl http://localhost:8015/health`
- Verify network: `docker network inspect trm-network`
