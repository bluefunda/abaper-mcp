# Deployment Guide

This guide covers deploying the ABAPER MCP server in various environments.

## Table of Contents

- [GitHub Actions CI/CD](#github-actions-cicd)
- [Docker Deployment](#docker-deployment)
- [GoReleaser](#goreleaser)
- [Manual Deployment](#manual-deployment)
- [Environment Variables](#environment-variables)

## GitHub Actions CI/CD

The project includes three GitHub Actions workflows for automated deployment:

### 1. Production Deployment (`prd.yml`)

**Trigger**: Pushes to `main` branch

**Features**:
- Automatic semantic versioning
- Docker image builds and push to Docker Hub
- Deployment to production server via SSH
- Automatic cleanup of old images

**Semantic Versioning**:
- `[major]` in commit message → Major version bump (1.0.0 → 2.0.0)
- `[minor]` in commit message → Minor version bump (1.0.0 → 1.1.0)
- Default → Patch version bump (1.0.0 → 1.0.1)

**Example Commit Messages**:
```bash
git commit -m "feat: add new MCP tool [minor]"
git commit -m "fix: correct resource handler bug"
git commit -m "feat: major API overhaul [major]"
```

**Required Secrets**:
- `DOCKER_USERNAME` - Docker Hub username
- `DOCKER_PASSWORD` - Docker Hub password/token
- `SERVER_USER` - SSH username for production server
- `SERVER_IP` - Production server IP address
- `SAP_HOST` - SAP system host
- `SAP_CLIENT` - SAP client number
- `SAP_USERNAME` - SAP username
- `SAP_PASSWORD` - SAP password

### 2. Staging Deployment (`staging.yml`)

**Trigger**: Pushes to `staging` branch

**Features**:
- Builds Docker image with `alpha` tag
- Deploys to staging server
- Uses separate staging credentials

**Required Secrets**:
- `DOCKER_USERNAME`
- `DOCKER_PASSWORD`
- `STAGING_SERVER_USER`
- `STAGING_SERVER_IP`
- `STAGING_SAP_HOST`
- `STAGING_SAP_CLIENT`
- `STAGING_SAP_USERNAME`
- `STAGING_SAP_PASSWORD`

### 3. Release Workflow (`release.yml`)

**Trigger**: Git tags matching `v*` (e.g., `v1.0.0`)

**Features**:
- Creates GitHub releases
- Builds binaries for multiple platforms
- Publishes Docker images (multi-architecture)
- Creates Linux packages (deb, rpm, apk)
- Generates checksums

**Platforms Supported**:
- Linux: amd64, arm64, armv6, armv7
- macOS: amd64 (Intel), arm64 (Apple Silicon)
- Windows: amd64
- FreeBSD: amd64

**Creating a Release**:
```bash
# Create and push a tag
make tag TAG=v1.0.0

# Or manually
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## Docker Deployment

### Building Images

#### Local Build (Single Architecture)
```bash
# Build for current architecture
make docker

# Or with custom version
VERSION=1.0.0 make docker
```

#### Multi-Architecture Build
```bash
# Build and push for amd64 and arm64
make docker-buildx

# Requires Docker Buildx and authentication
docker buildx create --use
docker login
```

### Running the Container

#### Using Docker Compose
Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  abaper-mcp:
    image: bluefunda/abaper-mcp:latest
    container_name: abaper-mcp
    restart: unless-stopped
    environment:
      - SAP_HOST=https://your-sap-host:8000
      - SAP_CLIENT=100
      - SAP_USERNAME=your-username
      - SAP_PASSWORD=your-password
    volumes:
      - ./logs:/var/log/abaper
      - ./config:/etc/abaper
    networks:
      - mcp-network

networks:
  mcp-network:
    driver: bridge
```

Run with:
```bash
docker-compose up -d
```

#### Using Docker Run
```bash
docker run -d \
  --name abaper-mcp \
  --restart unless-stopped \
  -e SAP_HOST="https://your-sap-host:8000" \
  -e SAP_CLIENT="100" \
  -e SAP_USERNAME="your-username" \
  -e SAP_PASSWORD="your-password" \
  -v $(pwd)/logs:/var/log/abaper \
  -v $(pwd)/config:/etc/abaper \
  bluefunda/abaper-mcp:latest
```

### Health Checks
```bash
# Check container status
docker ps --filter name=abaper-mcp

# View logs
docker logs abaper-mcp

# Check health
docker inspect --format='{{.State.Health.Status}}' abaper-mcp
```

## GoReleaser

GoReleaser automates the release process including building, packaging, and publishing.

### Installation
```bash
# macOS
brew install goreleaser

# Linux
curl -sfL https://goreleaser.com/static/run | sh

# Or download from https://github.com/goreleaser/goreleaser/releases
```

### Creating a Release

#### Full Release (requires tag)
```bash
# Set GitHub token
export GITHUB_TOKEN="your-github-token"

# Create release
make release

# Or directly
goreleaser release --clean
```

#### Snapshot Release (no tag required)
```bash
# Create local build without publishing
make snapshot

# Artifacts will be in dist/
ls dist/
```

#### Validate Configuration
```bash
# Check .goreleaser.yml syntax
make validate-release
```

### Release Artifacts

GoReleaser creates:

1. **Binary Archives**:
   - `abaper-mcp_<version>_Linux_x86_64.tar.gz`
   - `abaper-mcp_<version>_Darwin_arm64.tar.gz`
   - etc.

2. **Linux Packages**:
   - `abaper-mcp_<version>_amd64.deb` (Debian/Ubuntu)
   - `abaper-mcp_<version>_amd64.rpm` (RHEL/CentOS)
   - `abaper-mcp_<version>_x86_64.apk` (Alpine)

3. **Docker Images**:
   - `bluefunda/abaper-mcp:latest`
   - `bluefunda/abaper-mcp:<version>`
   - `bluefunda/abaper-mcp:<version>-amd64`
   - `bluefunda/abaper-mcp:<version>-arm64`

4. **Checksums**:
   - `checksums.txt` with SHA256 hashes

## Manual Deployment

### Building from Source

```bash
# Clone repository
git clone https://github.com/bluefunda/abaper-mcp.git
cd abaper-mcp

# Install dependencies
make install

# Build
make build

# Or build for all platforms
make build-all
```

### Installing Binary

#### Linux/macOS
```bash
# Copy to system path
sudo cp abaper-mcp /usr/local/bin/

# Make executable
sudo chmod +x /usr/local/bin/abaper-mcp

# Verify installation
abaper-mcp --version
```

#### Windows
```powershell
# Copy to a directory in PATH
Copy-Item abaper-mcp.exe C:\Windows\System32\

# Or add to PATH
$env:Path += ";C:\path\to\abaper-mcp"
```

### Installing Linux Packages

#### Debian/Ubuntu
```bash
# Download .deb file
wget https://github.com/bluefunda/abaper-mcp/releases/download/v1.0.0/abaper-mcp_1.0.0_amd64.deb

# Install
sudo dpkg -i abaper-mcp_1.0.0_amd64.deb

# Or with apt
sudo apt install ./abaper-mcp_1.0.0_amd64.deb
```

#### RHEL/CentOS/Fedora
```bash
# Download .rpm file
wget https://github.com/bluefunda/abaper-mcp/releases/download/v1.0.0/abaper-mcp_1.0.0_amd64.rpm

# Install
sudo rpm -i abaper-mcp_1.0.0_amd64.rpm

# Or with dnf
sudo dnf install abaper-mcp_1.0.0_amd64.rpm
```

#### Alpine
```bash
# Download .apk file
wget https://github.com/bluefunda/abaper-mcp/releases/download/v1.0.0/abaper-mcp_1.0.0_x86_64.apk

# Install
sudo apk add --allow-untrusted abaper-mcp_1.0.0_x86_64.apk
```

## Environment Variables

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `SAP_HOST` | SAP system host with protocol and port | `https://sapdev.company.com:8000` |
| `SAP_CLIENT` | SAP client number | `100` |
| `SAP_USERNAME` | SAP username | `DEVELOPER01` |
| `SAP_PASSWORD` | SAP password | `SecurePassword123` |

### Optional Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |
| `LOG_FILE` | Path to log file | stdout |

### Setting Environment Variables

#### Linux/macOS
```bash
# In shell
export SAP_HOST="https://sapdev.company.com:8000"
export SAP_CLIENT="100"
export SAP_USERNAME="DEVELOPER01"
export SAP_PASSWORD="SecurePassword123"

# In .env file
cat > .env << EOF
SAP_HOST=https://sapdev.company.com:8000
SAP_CLIENT=100
SAP_USERNAME=DEVELOPER01
SAP_PASSWORD=SecurePassword123
EOF

# Load .env file
source .env
```

#### Windows PowerShell
```powershell
$env:SAP_HOST = "https://sapdev.company.com:8000"
$env:SAP_CLIENT = "100"
$env:SAP_USERNAME = "DEVELOPER01"
$env:SAP_PASSWORD = "SecurePassword123"
```

## Deployment Checklist

- [ ] Set up GitHub repository secrets
- [ ] Configure `.env` file with SAP credentials
- [ ] Test build locally: `make build`
- [ ] Test Docker build: `make docker`
- [ ] Validate GoReleaser config: `make validate-release`
- [ ] Create staging branch for testing
- [ ] Test staging deployment
- [ ] Create release tag for production
- [ ] Verify GitHub release artifacts
- [ ] Test Docker image pull and run
- [ ] Configure Claude Desktop with server
- [ ] Test MCP tools, resources, and prompts
- [ ] Monitor logs for errors
- [ ] Set up monitoring/alerting (optional)

## Troubleshooting

### Build Issues

**Problem**: Build fails with module errors
```bash
# Solution: Clean and reinstall dependencies
go clean -modcache
make install
make build
```

**Problem**: GoReleaser fails
```bash
# Solution: Validate configuration
make validate-release

# Check for uncommitted changes
git status
```

### Docker Issues

**Problem**: Image won't start
```bash
# Solution: Check logs
docker logs abaper-mcp

# Check environment variables
docker inspect abaper-mcp
```

**Problem**: Connection to SAP fails
```bash
# Solution: Verify network connectivity
docker exec abaper-mcp ping your-sap-host

# Check environment variables are set
docker exec abaper-mcp env | grep SAP
```

### GitHub Actions Issues

**Problem**: Workflow fails on secrets
```bash
# Solution: Verify secrets are set in GitHub
# Settings → Secrets and variables → Actions
```

**Problem**: Docker push fails
```bash
# Solution: Verify Docker Hub credentials
# Regenerate Docker Hub token if needed
```

## Support

For deployment issues:
- GitHub Issues: https://github.com/bluefunda/abaper-mcp/issues
- Documentation: https://github.com/bluefunda/abaper-mcp
- ABAPER Library: https://github.com/bluefunda/abaper
