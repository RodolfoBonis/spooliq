# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SpoolIq is a Go-based API for calculating 3D printing costs, supporting multi-color filament calculations, energy costs, wear and overhead, and labor. Built with Gin framework, Uber FX for dependency injection, and infrastructure services including PostgreSQL, Redis, RabbitMQ, and Keycloak.

## Essential Commands

### Development
```bash
make run                    # Run application locally
make build                  # Build the binary
make test                   # Run tests
make lint                   # Run linters (gofmt, go vet, golint, staticcheck, goimports)
```

### Infrastructure Management
```bash
make infrastructure/raise   # Start PostgreSQL, Redis, RabbitMQ, Keycloak
make infrastructure/down    # Stop all infrastructure services
make app/run               # Start full application with Docker Compose
make app/down              # Stop full application
```

### Database Operations
```bash
make db/migrate            # Run pending migrations
make db/rollback           # Rollback last migration
make db/create NAME="name" # Create new migration
make db/status             # Check migration status
```

### Cache Management
```bash
make cache/test            # Test cache endpoints
make cache/status          # Show Redis status and cache keys
make cache/clear           # Clear all cache
```

### Docker
```bash
make docker/build          # Build ultra-optimized image (scratch + UPX)
make docker/build-ca       # Build image with CA certificates
```

### Swagger Documentation
```bash
bash scripts/generate-swagger.sh  # Generate Swagger docs
# Auto-generated API docs available at http://localhost:8000/docs/index.html
```

## Architecture

### Directory Structure
- **app/** - Application initialization, FX modules, lifecycle hooks
- **core/** - Central components: config, logger, middlewares, services, errors
  - **config/** - App configuration and environment management
  - **middlewares/** - Auth, CORS, cache, monitoring middlewares
  - **services/** - Database, Redis, AMQP, Auth services
  - **migrations/** - SQL migration system
- **features/** - Business domain modules (currently auth module)
- **routes/** - API route definitions and router setup
- **docs/** - Swagger/OpenAPI documentation

### Key Technologies
- **Framework**: Gin for HTTP routing
- **DI**: Uber FX for dependency injection
- **Database**: GORM with PostgreSQL
- **Cache**: Redis with decorator-style caching
- **Auth**: Keycloak integration with JWT
- **Monitoring**: Prometheus metrics, structured logging with Zap
- **Message Queue**: RabbitMQ via AMQP

### Service Dependencies
The application expects these services (provided via Docker Compose):
- PostgreSQL on :5432 (user/password from .env)
- Redis on :6379 (password: redis123)
- RabbitMQ on :5672/:15672 (admin/admin123)
- Keycloak on :8180 (admin/admin123, realm: spooliq-realm)

### Authentication
- JWT-based authentication with Keycloak
- Role-based access control (admin, user roles)
- API key authentication support
- Middleware in `core/middlewares/auth_middleware.go`

### Caching System
- Redis-based caching with TypeScript-like decorators
- Cache strategies: time-based, user-specific, query-aware
- Cache middleware for automatic HTTP response caching
- Detailed documentation in `app_docs/cache.md`

### Git Workflow
Conventional commits are enforced (see `.cursor/rules/commit-flow.mdc`):
- feat: New feature
- fix: Bug fix  
- refactor: Code refactoring
- test: Test changes
- docs: Documentation
- chore: Maintenance

### Testing Approach
Tests use Go's standard testing package. Run specific tests with:
```bash
go test ./features/auth/...  # Test specific module
go test -v ./...             # Verbose test output
```

## Important Configuration Files
- **.env** - Environment variables (copy from .env.example)
- **docker-compose.yaml** - Full stack orchestration
- **Makefile** - All development commands
- **go.mod** - Go dependencies

## Docusaurus Documentation
A separate Docusaurus site exists in `spooliq_docs/`:
```bash
cd spooliq_docs
npm install
npm start  # Starts on localhost:3000
```