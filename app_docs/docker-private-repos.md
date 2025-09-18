# Docker Build with Private GitHub Repositories

This project uses private GitHub repositories that require authentication during the Docker build process. This document explains how to configure and build the application with private dependencies.

## Overview

The application depends on private GitHub repositories, specifically:
- `github.com/RodolfoBonis/go_key_guardian` - Private API key management library

To build the Docker image successfully, you need to provide a GitHub Personal Access Token with appropriate permissions.

## Prerequisites

### 1. GitHub Personal Access Token

Create a GitHub Personal Access Token with the following permissions:
- `repo` - Full control of private repositories
- `read:packages` - Read packages (if using GitHub Packages)

**Steps to create:**
1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Select scopes: `repo`, `read:packages`
4. Copy the generated token

### 2. Environment Configuration

Add your GitHub token to the `.env` file:

```bash
# Copy example file
cp .env.example .env

# Edit .env and add your GitHub token
GITHUB_TOKEN=your_github_personal_access_token_here
```

## Docker Build Process

### Multi-Stage Build Architecture

The Dockerfile uses a multi-stage build optimized for private repositories:

1. **Build Environment Stage**: Downloads dependencies and tools
2. **Builder Stage**: Compiles the application
3. **Production Stage**: Creates minimal runtime image

### Key Features

- **Private Repo Access**: Configured Git authentication with OAuth token
- **GOPRIVATE**: Explicitly marks private modules
- **Dependency Caching**: Optimized layer caching for faster builds
- **Swagger Generation**: Automatic API documentation generation
- **Ultra-Minimal Image Size**: Uses scratch base image (~500KB-1MB final image)
- **UPX Compression**: Binary compressed with maximum LZMA compression
- **Optimized Binary**: Stripped symbols, debug info, and compressed

## Building the Application

### 1. Using Docker Compose (Recommended)

```bash
# Set your GitHub token in .env file first
docker-compose build

# Or build and run
docker-compose up --build
```

### 2. Direct Docker Build

```bash
# Build with build arguments
docker build \
  --build-arg GITHUB_TOKEN=your_token_here \
  --build-arg VERSION=v1.0.0 \
  -t spooliq .

# Run the container
docker run -p 8000:8000 spooliq
```

### 3. Build for Different Environments

```bash
# Development build
docker build \
  --build-arg GITHUB_TOKEN=${GITHUB_TOKEN} \
  --build-arg VERSION=dev \
  -t spooliq:dev .

# Production build
docker build \
  --build-arg GITHUB_TOKEN=${GITHUB_TOKEN} \
  --build-arg VERSION=v1.0.0 \
  -t spooliq:v1.0.0 .
```

## Image Size Optimizations

The Dockerfile is optimized for minimal final image size:

### Size Reduction Techniques

1. **Scratch Base Image**: Uses `FROM scratch` (literally 0MB base)
2. **UPX Compression**: Maximum LZMA compression (`upx --best --lzma`)
3. **Optimized Build Flags**: Strips symbols and debug info (`-ldflags="-w -s"`)
4. **Single Binary**: Statically linked Go binary with no external dependencies
5. **Minimal Layer Count**: Only 2 stages with essential operations
6. **Efficient .dockerignore**: Excludes unnecessary files from build context

### Build Flags Explanation

```dockerfile
# Build with maximum optimization
RUN go build \
    -a \                                    # Force rebuild of packages
    -installsuffix cgo \                   # Add suffix to package dir
    -ldflags="-w -s -X main.version=${VERSION}" \  # Strip debug info and set version
    -o spooliq \
    ./main.go

# Compress with UPX for extreme size reduction
RUN upx --best --lzma spooliq
```

**Go Build Flags:**
- `-w`: Omit DWARF symbol table
- `-s`: Omit symbol table and debug info
- `-a`: Force rebuilding of packages that are already up-to-date

**UPX Compression:**
- `--best`: Maximum compression ratio
- `--lzma`: Use LZMA algorithm (best compression)
- **Result**: 60-80% binary size reduction

### Image Size Comparison

| Base Image | Binary | Final Size | Security | Use Case |
|------------|--------|------------|----------|----------|
| `alpine:latest` | Normal | ~15-20MB | Good | General purpose |
| `distroless/static` | Optimized | ~2-5MB | Excellent | Production |
| `scratch` | Optimized | ~1-3MB | Basic | Minimal, no CA certs |
| `scratch` | **UPX Compressed** | **~500KB-1MB** | **Basic** | **Ultra-minimal** |

**Current Implementation**: Scratch + UPX for maximum size reduction

### Ultra-Minimal Considerations

**Advantages:**
- ✅ **Smallest possible size**: ~500KB-1MB final image
- ✅ **Fastest deployment**: Minimal download time
- ✅ **Minimal attack surface**: Only your binary exists
- ✅ **No unnecessary dependencies**: Zero bloat

**Limitations:**
- ❌ **No shell access**: Cannot exec into container for debugging
- ❌ **No CA certificates**: HTTPS requests to external APIs may fail
- ❌ **No utilities**: No `ls`, `ps`, `wget`, etc.
- ❌ **Debugging complexity**: Harder to troubleshoot in production

**When to Use:**
- ✅ Internal microservices (no external HTTPS calls)
- ✅ Production environments with external monitoring
- ✅ Kubernetes with service mesh (handles certificates)
- ✅ Maximum optimization priority

**Alternative Configuration:**
If you need CA certificates later, you can easily switch back to distroless by changing the production stage:

```dockerfile
# Switch to distroless if CA certs needed
FROM gcr.io/distroless/static-debian12:nonroot AS production
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
```

## Dockerfile Structure

### Build Arguments

```dockerfile
ARG GITHUB_TOKEN
ARG VERSION=unknown
```

### Git Configuration

```dockerfile
# Configure git to use GitHub token for private repositories
RUN git config --global url."https://${TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"
```

### Private Module Configuration

```dockerfile
# Configure private module access
RUN go env -w GOPRIVATE=github.com/RodolfoBonis/go_key_guardian

# Ensure private dependency is available
RUN go get github.com/RodolfoBonis/go_key_guardian
```

## CI/CD Configuration

### GitHub Actions Example

```yaml
name: Build and Deploy

on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build Docker image
        run: |
          docker build \
            --build-arg GITHUB_TOKEN=${{ secrets.GITHUB_TOKEN }} \
            --build-arg VERSION=${{ github.sha }} \
            -t spooliq:${{ github.sha }} .
```

### GitLab CI Example

```yaml
build:
  stage: build
  script:
    - docker build 
        --build-arg GITHUB_TOKEN=$CI_JOB_TOKEN 
        --build-arg VERSION=$CI_COMMIT_SHA 
        -t spooliq:$CI_COMMIT_SHA .
  variables:
    DOCKER_DRIVER: overlay2
```

## Security Best Practices

### 1. Token Management

- **Never commit tokens to repository**
- Use environment variables or secret management systems
- Rotate tokens regularly
- Use minimal required permissions

### 2. Multi-Stage Builds

The Dockerfile uses multi-stage builds to ensure:
- Build secrets don't leak to final image
- Minimal production image size
- Clean separation of build and runtime environments

### 3. Build Context

```bash
# Use .dockerignore to exclude sensitive files
echo ".env" >> .dockerignore
echo "*.key" >> .dockerignore
echo ".git" >> .dockerignore
```

## Troubleshooting

### Permission Denied Errors

```bash
# Error: permission denied while trying to connect to the Docker daemon
sudo usermod -aG docker $USER
newgrp docker
```

### Authentication Failures

```bash
# Error: fatal: could not read Username for 'https://github.com'
# Solution: Verify GITHUB_TOKEN is set correctly
echo $GITHUB_TOKEN

# Verify token has correct permissions in GitHub settings
```

### Private Module Access

```bash
# Error: go: github.com/RodolfoBonis/go_key_guardian: reading at revision v0.0.0: unknown revision
# Solution: Check GOPRIVATE configuration and token permissions
```

### Build Cache Issues

```bash
# Clear Docker build cache
docker builder prune

# Build without cache
docker build --no-cache \
  --build-arg GITHUB_TOKEN=${GITHUB_TOKEN} \
  -t spooliq .
```

### Image Size Analysis

```bash
# Check image size
docker images spooliq

# Analyze image layers
docker history spooliq

# Detailed image inspection
docker inspect spooliq

# Compare image sizes
docker images | grep spooliq
```

## Local Development

For local development without Docker:

```bash
# Configure git authentication
git config --global url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"

# Set GOPRIVATE
go env -w GOPRIVATE=github.com/RodolfoBonis/go_key_guardian

# Download dependencies
go mod download

# Run application
go run main.go
```

## Production Deployment

### Container Registry

```bash
# Tag for registry
docker tag spooliq:latest registry.company.com/spooliq:latest

# Push to registry
docker push registry.company.com/spooliq:latest
```

### Environment Variables in Production

```bash
# Use secret management for production
kubectl create secret generic app-secrets \
  --from-literal=github-token=${GITHUB_TOKEN}
```

## Additional Resources

- [GitHub Personal Access Tokens](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
- [Docker Multi-Stage Builds](https://docs.docker.com/develop/dev-best-practices/dockerfile_best-practices/#use-multi-stage-builds)
- [Go Private Modules](https://go.dev/ref/mod#private-modules) 