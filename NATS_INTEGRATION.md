# NATS Integration Guide

This document describes the NATS integration for ABAPER MCP server, enabling orchestrator-based deployment and multi-server architectures.

## Overview

ABAPER MCP server supports three operational modes:

1. **stdio** (default) - For Claude Desktop integration via standard input/output
2. **nats** - For orchestrator integration via NATS messaging
3. **dual** - Both stdio and NATS simultaneously

## Architecture

### NATS Components

The integration uses three NATS features:

1. **Core NATS** - Pub/sub messaging for tool requests/responses
2. **JetStream** - Persistence and streaming capabilities
3. **Key-Value Store** - Configuration storage

### Subject Hierarchy

```
mcp.abaper.tools.request        # Direct requests
mcp.abaper.tools.response       # Direct responses
mcp.*.abaper.tools.request      # Realm-based requests (e.g., mcp.production.abaper.tools.request)
mcp.*.abaper.tools.response     # Realm-based responses
```

### Configuration Sources

Configuration is loaded in priority order:

1. NATS KV Store (if enabled) - Key: `SAPConfig`
2. Environment variables (fallback)

## Operational Modes

### stdio Mode (Default)

Classic MCP server mode for Claude Desktop.

```bash
# No ABAPER_MODE variable needed (default)
export SAP_HOST="https://saphost:8000"
export SAP_CLIENT="100"
export SAP_USERNAME="user"
export SAP_PASSWORD="pass"

./abaper-mcp
```

**Output:**
```
Starting ABAPER MCP server in stdio mode
Running in stdio mode (Claude Desktop compatible)
```

### nats Mode

NATS-only mode for orchestrator deployment.

```bash
export ABAPER_MODE="nats"
export NATS_URL="tls://connect.ngs.global:4222"
export NATS_CREDS="/path/to/user.creds"
export NATS_ENABLE_KV="true"
export NATS_ENABLE_MESSAGING="true"

# SAP config from environment (if not in KV)
export SAP_HOST="https://saphost:8000"
export SAP_CLIENT="100"
export SAP_USERNAME="user"
export SAP_PASSWORD="pass"

./abaper-mcp
```

**Output:**
```
Starting ABAPER MCP server in nats mode
Running in NATS mode (orchestrator compatible)
Successfully connected to NATS at tls://connect.ngs.global:4222
JetStream context created
KV bucket 'AbaperMCPConfigBucket' ready
NATS MCP server started and listening for tool requests
NATS MCP server is running. Press Ctrl+C to exit.
```

### dual Mode

Both stdio and NATS simultaneously.

```bash
export ABAPER_MODE="dual"
export NATS_URL="tls://connect.ngs.global:4222"
export NATS_CREDS="/path/to/user.creds"
export NATS_ENABLE_MESSAGING="true"
# ... other environment variables

./abaper-mcp
```

**Output:**
```
Starting ABAPER MCP server in dual mode
Running in dual mode (stdio + NATS)
Successfully connected to NATS at tls://connect.ngs.global:4222
NATS listener started successfully
Running in stdio mode (Claude Desktop compatible)
```

## Environment Variables

### Required for NATS Mode

| Variable | Description | Example |
|----------|-------------|---------|
| `ABAPER_MODE` | Operational mode | `stdio`, `nats`, `dual` |
| `NATS_URL` | NATS server URL | `tls://connect.ngs.global:4222` |
| `NATS_CREDS` | Path to NATS credentials file | `/var/nats/creds/user.creds` |

### Optional NATS Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `NATS_ENABLE_KV` | Enable Key-Value store | `false` |
| `NATS_ENABLE_MESSAGING` | Enable messaging | `false` |
| `NATS_KV_BUCKET` | KV bucket name | `AbaperMCPConfigBucket` |

### SAP Configuration

These can be set via environment OR stored in NATS KV:

| Variable | Description | Example |
|----------|-------------|---------|
| `SAP_HOST` | SAP ADT host URL | `https://sapdev.company.com:8000` |
| `SAP_CLIENT` | SAP client number | `100` |
| `SAP_USERNAME` | SAP username | `DEVELOPER01` |
| `SAP_PASSWORD` | SAP password | `SecurePassword123` |

## NATS KV Configuration Storage

### Storing SAP Configuration

Instead of environment variables, store SAP config in NATS KV:

```go
// Example: Using NATS CLI or API to store config
{
  "host": "https://sapdev.company.com:8000",
  "client": "100",
  "username": "DEVELOPER01",
  "password": "SecurePassword123"
}
```

Store with key: `SAPConfig`

### Loading from KV Store

When `NATS_ENABLE_KV=true`, the server will:

1. Connect to NATS
2. Access the KV bucket
3. Retrieve `SAPConfig` key
4. Use KV config (overriding environment variables)
5. Fall back to environment variables if KV fails

**Example output:**
```
Loaded SAP configuration from NATS KV
```

## Message Protocol

### Tool Request Format

```json
{
  "tool": "get-object",
  "arguments": {
    "object_type": "class",
    "object_name": "ZCL_EXAMPLE"
  },
  "request_id": "req-12345"
}
```

**Subject:** `mcp.abaper.tools.request` or `mcp.production.abaper.tools.request`

### Tool Response Format

```json
{
  "request_id": "req-12345",
  "success": true,
  "result": {
    "object_type": "class",
    "object_name": "ZCL_EXAMPLE",
    "source_code": "CLASS zcl_example DEFINITION..."
  },
  "timestamp": "2025-11-15T20:30:00Z"
}
```

**Subject:** `mcp.abaper.tools.response` or `mcp.production.abaper.tools.response`

### Error Response Format

```json
{
  "request_id": "req-12345",
  "success": false,
  "error": "object_type is required",
  "timestamp": "2025-11-15T20:30:00Z"
}
```

## Supported Tools via NATS

All MCP tools are available via NATS:

1. **get-object** - Retrieve ABAP object source
2. **search-objects** - Search for ABAP objects
3. **list-packages** - List ABAP packages
4. **test-connection** - Test SAP connectivity
5. **create-program** - Create new ABAP program
6. **create-class** - Create new ABAP class

## Docker Deployment with NATS

### Production Deployment

```bash
docker run -d \
  --name abaper-mcp \
  --network mcp-network \
  --restart unless-stopped \
  -e ABAPER_MODE="nats" \
  -e NATS_URL="tls://connect.ngs.global:4222" \
  -e NATS_CREDS="/var/nats/creds/user.creds" \
  -e NATS_ENABLE_KV="true" \
  -e NATS_ENABLE_MESSAGING="true" \
  -e SAP_HOST="https://saphost:8000" \
  -e SAP_CLIENT="100" \
  -e SAP_USERNAME="user" \
  -e SAP_PASSWORD="pass" \
  -v /var/log/abaper-mcp:/var/log/abaper \
  -v /etc/abaper-mcp:/etc/abaper \
  -v /var/nats/creds:/var/nats/creds:ro \
  bluefunda/abaper-mcp:latest
```

### Dual Mode (Claude Desktop + Orchestrator)

```bash
docker run -d \
  --name abaper-mcp \
  --network mcp-network \
  --restart unless-stopped \
  -e ABAPER_MODE="dual" \
  -e NATS_URL="tls://connect.ngs.global:4222" \
  -e NATS_CREDS="/var/nats/creds/user.creds" \
  -e NATS_ENABLE_MESSAGING="true" \
  -e SAP_HOST="https://saphost:8000" \
  -e SAP_CLIENT="100" \
  -e SAP_USERNAME="user" \
  -e SAP_PASSWORD="pass" \
  -v /var/nats/creds:/var/nats/creds:ro \
  bluefunda/abaper-mcp:latest
```

## NATS Credentials Setup

### 1. Obtain NATS Credentials

From NATS Cloud (NGS) or your NATS server:

```bash
# Download your credentials file (user.creds)
# Contains both NKey seed and JWT token
```

### 2. Store Credentials Securely

```bash
# On host machine
mkdir -p /var/nats/creds
chmod 700 /var/nats/creds
cp user.creds /var/nats/creds/
chmod 400 /var/nats/creds/user.creds
```

### 3. Mount as Read-Only Volume

The Docker deployment mounts credentials as read-only:

```bash
-v /var/nats/creds:/var/nats/creds:ro
```

## GitHub Actions Secrets

Configure these secrets for automated deployment:

### Production

- `NATS_URL` - Production NATS URL
- `NATS_CREDS` - Production NATS credentials (base64 encoded)
- `ABAPER_MODE` - Set to `nats` or `dual`
- `SAP_HOST`, `SAP_CLIENT`, `SAP_USERNAME`, `SAP_PASSWORD`

### Staging

- `STAGING_NATS_URL`
- `STAGING_NATS_CREDS`
- `STAGING_ABAPER_MODE`
- `STAGING_SAP_*` variables

## Orchestrator Integration

### Compatible with cai-mcp-client

This implementation is compatible with the cai-mcp-client orchestrator:

1. **Request Routing** - Orchestrator publishes to `mcp.abaper.tools.request`
2. **Response Handling** - ABAPER MCP publishes to `mcp.abaper.tools.response`
3. **Realm Support** - Supports multi-tenant deployments via realm-based subjects

### Multi-Server Architecture

```
[Claude/AI Client]
        |
        v
[cai-mcp-client Orchestrator]
        |
        +---> [NATS Server]
                   |
                   +---> mcp.abaper.tools.request ---> [ABAPER MCP Server]
                   +---> mcp.xmlodata.tools.request -> [XML/OData MCP Server]
                   +---> mcp.*.tools.request --------> [Other MCP Servers]
```

## Testing NATS Integration

### 1. Test NATS Connection

```bash
# Set environment
export ABAPER_MODE="nats"
export NATS_URL="tls://connect.ngs.global:4222"
export NATS_CREDS="/path/to/user.creds"
export NATS_ENABLE_MESSAGING="true"
export SAP_HOST="https://saphost:8000"
export SAP_CLIENT="100"
export SAP_USERNAME="user"
export SAP_PASSWORD="pass"

# Run server
./abaper-mcp

# Expected output:
# Successfully connected to NATS at tls://connect.ngs.global:4222
# NATS MCP server started and listening for tool requests
```

### 2. Test Tool Request (using NATS CLI)

```bash
# Install NATS CLI
# brew install nats-io/nats-tools/nats

# Send test request
nats req mcp.abaper.tools.request \
  '{"tool":"test-connection","arguments":{},"request_id":"test-1"}' \
  --creds /path/to/user.creds
```

**Expected response:**
```json
{
  "request_id": "test-1",
  "success": true,
  "result": {
    "connected": true,
    "message": "Successfully connected to SAP system"
  },
  "timestamp": "2025-11-15T20:45:00Z"
}
```

### 3. Test KV Configuration

```bash
# Store config in KV
nats kv put AbaperMCPConfigBucket SAPConfig \
  '{"host":"https://saphost:8000","client":"100","username":"user","password":"pass"}' \
  --creds /path/to/user.creds

# Run server with KV enabled
export NATS_ENABLE_KV="true"
./abaper-mcp

# Expected output:
# Loaded SAP configuration from NATS KV
```

## Monitoring and Debugging

### Connection Status

```bash
# Check NATS connection
docker logs abaper-mcp | grep "NATS"

# Should show:
# Successfully connected to NATS at...
# NATS MCP server started and listening...
```

### Request Logging

```bash
# Monitor incoming requests
docker logs -f abaper-mcp

# Output shows:
# Received NATS request on subject: mcp.abaper.tools.request
```

### NATS Server Monitoring

```bash
# Using NATS CLI
nats server list --creds /path/to/user.creds
nats server info --creds /path/to/user.creds
```

## Troubleshooting

### Connection Issues

**Problem:** Cannot connect to NATS

```bash
# Check URL and credentials
echo $NATS_URL
echo $NATS_CREDS

# Verify credentials file exists
cat $NATS_CREDS

# Test connection with NATS CLI
nats server ping --creds $NATS_CREDS
```

### KV Store Issues

**Problem:** Failed to load config from NATS KV

```bash
# Check KV bucket exists
nats kv ls --creds $NATS_CREDS

# Check SAPConfig key
nats kv get AbaperMCPConfigBucket SAPConfig --creds $NATS_CREDS

# Create bucket if missing
nats kv add AbaperMCPConfigBucket --creds $NATS_CREDS
```

### Message Delivery Issues

**Problem:** Requests not reaching server

```bash
# Check subscriptions
nats sub "mcp.>" --creds $NATS_CREDS

# Monitor all MCP traffic
nats sub "mcp.abaper.>" --creds $NATS_CREDS
```

### Permission Issues

**Problem:** NATS permission denied

```bash
# Verify user permissions
nats account info --creds $NATS_CREDS

# User needs permissions for:
# - Publish: mcp.abaper.tools.response
# - Subscribe: mcp.abaper.tools.request, mcp.*.abaper.tools.request
# - KV: AbaperMCPConfigBucket (if using KV)
```

## Security Considerations

### Credential Security

1. **Never commit credentials** to version control
2. **Use read-only mounts** for credential files in Docker
3. **Rotate credentials** regularly
4. **Use NATS account isolation** for multi-tenant deployments

### Network Security

1. **Use TLS** for all NATS connections
2. **Enable JetStream encryption** for sensitive data
3. **Use NATS account limits** to prevent abuse
4. **Monitor connection metrics** for anomalies

### SAP Password Security

1. **Prefer NATS KV** over environment variables for passwords
2. **Use KV encryption** if available
3. **Implement credential rotation** policies
4. **Audit access** to KV configuration

## Migration Guide

### From stdio-only to NATS Mode

1. **Set up NATS credentials**
   ```bash
   # Obtain credentials from NATS provider
   cp user.creds /var/nats/creds/
   ```

2. **Store SAP config in NATS KV** (optional)
   ```bash
   nats kv put AbaperMCPConfigBucket SAPConfig \
     '{"host":"...","client":"...","username":"...","password":"..."}' \
     --creds /var/nats/creds/user.creds
   ```

3. **Update environment variables**
   ```bash
   export ABAPER_MODE="dual"  # Use dual for gradual migration
   export NATS_URL="tls://connect.ngs.global:4222"
   export NATS_CREDS="/var/nats/creds/user.creds"
   export NATS_ENABLE_MESSAGING="true"
   export NATS_ENABLE_KV="true"
   ```

4. **Restart server**
   ```bash
   docker restart abaper-mcp
   ```

5. **Verify both modes work**
   - Test Claude Desktop (stdio)
   - Test orchestrator requests (NATS)

6. **Switch to NATS-only** when ready
   ```bash
   export ABAPER_MODE="nats"
   docker restart abaper-mcp
   ```

## Performance Considerations

### Connection Pooling

- NATS connection is singleton per server instance
- Thread-safe for concurrent requests
- Automatic reconnection on connection loss

### Message Size Limits

- Default NATS max payload: 1MB
- Large ABAP sources may need chunking
- Consider JetStream for large objects

### Concurrent Requests

- Each tool request handled in separate goroutine
- Concurrent limit based on SAP connection pool
- NATS subscription queue for load distribution

## References

- [NATS Documentation](https://docs.nats.io)
- [NATS JetStream](https://docs.nats.io/nats-concepts/jetstream)
- [NATS Key-Value Store](https://docs.nats.io/nats-concepts/jetstream/key-value-store)
- [cai-mcp-client Orchestrator](https://github.com/bluefunda/cai-mcp-client)
- [cai-xml-odata MCP Server](https://github.com/bluefunda/cai-xml-odata)
- [Model Context Protocol](https://modelcontextprotocol.io)

## Support

For NATS integration issues:
- GitHub Issues: https://github.com/bluefunda/abaper-mcp/issues
- Check logs: `docker logs abaper-mcp`
- Monitor NATS: `nats sub "mcp.abaper.>" --creds $NATS_CREDS`
