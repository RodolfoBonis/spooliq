# Makefile for the Go API project

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOTOOLS=$(GOCMD) tool

# Docker parameters
DOCKER_COMPOSE=docker-compose

# Binary name - automatically derived from directory name
BINARY_NAME=$(shell basename $(CURDIR))
PROJECT_NAME=$(shell basename $(CURDIR))

.PHONY: all build run test clean lint help infrastructure/raise infrastructure/down infrastructure/logs infrastructure/restart app/run app/down app/logs cache/test cache/status cache/clear db/migrate db/rollback db/status db/dry-run db/seed db/reset

all: help

# Build the application
build:
	@echo "Building the application..."
	@$(GOBUILD) -o $(BINARY_NAME) main.go

# Run the application locally
run:
	@echo "Running the application locally..."
	@$(GORUN) main.go

# Run the tests
test:
	@echo "Running tests..."
	@$(GOTEST) ./...

# Clean the binary
clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)

# Run the linter
lint:
	@echo "Running linter..."
	@sh .config/scripts/lint.sh

# Raise the infrastructure (PostgreSQL + RabbitMQ + Keycloak + Redis)
infrastructure/raise:
	@echo "🚀 Starting infrastructure services..."
	@$(DOCKER_COMPOSE) up -d postgres rabbitmq keycloak redis
	@echo "✅ Infrastructure services started!"
	@echo "📊 PostgreSQL running on localhost:5432"
	@echo "🐰 RabbitMQ Management UI: http://localhost:15672 (admin/admin123)"
	@echo "🔐 Keycloak Admin Console: http://localhost:8180 (admin/admin123)"
	@echo "🌐 Keycloak Realm: spooliq-realm"
	@echo "🗄️  Redis Cache: localhost:6379 (password: redis123)"
	@echo "👤 Test Users:"
	@echo "   - admin/admin123 (admin role)"
	@echo "   - testuser/test123 (user role)"

# Stop the infrastructure
infrastructure/down:
	@echo "🛑 Stopping infrastructure services..."
	@$(DOCKER_COMPOSE) down
	@echo "✅ Infrastructure services stopped!"

# Show infrastructure logs
infrastructure/logs:
	@echo "📋 Showing infrastructure logs..."
	@$(DOCKER_COMPOSE) logs -f postgres rabbitmq keycloak redis

# Restart the infrastructure
infrastructure/restart:
	@echo "🔄 Restarting infrastructure services..."
	@$(DOCKER_COMPOSE) restart postgres rabbitmq keycloak redis
	@echo "✅ Infrastructure services restarted!"

# Run the full application with infrastructure
app/run:
	@echo "🚀 Starting full application with infrastructure..."
	@$(DOCKER_COMPOSE) up -d
	@echo "✅ Application and infrastructure running!"

# Stop the full application
app/down:
	@echo "🛑 Stopping full application..."
	@$(DOCKER_COMPOSE) down
	@echo "✅ Application stopped!"

# Show application logs
app/logs:
	@echo "📋 Showing application logs..."
	@$(DOCKER_COMPOSE) logs -f

# Test cache endpoints
cache/test:
	@echo "🧪 Testing cache endpoints..."
	@echo ""
	@echo "1. Testing normal endpoint (no cache):"
	@curl -s http://localhost:8080/v1/system | jq '.server.timestamp // .timestamp // "No timestamp found"'
	@echo ""
	@echo "2. Testing cached endpoint (first call - MISS):"
	@curl -s -H "X-Test-User: user123" http://localhost:8080/v1/system/cached | head -1
	@echo ""
	@echo "3. Testing cached endpoint (second call - should be HIT):"
	@curl -s -H "X-Test-User: user123" http://localhost:8080/v1/system/cached | head -1
	@echo ""
	@echo "4. Testing user-specific cache:"
	@curl -s http://localhost:8080/v1/system/user-specific | jq '.user_id'

# Check Redis connection and keys
cache/status:
	@echo "🗄️  Redis Cache Status:"
	@echo ""
	@echo "📊 Redis Info:"
	@docker exec spooliq_redis redis-cli --no-auth-warning -a redis123 info memory | grep used_memory_human || echo "Redis not running"
	@echo ""
	@echo "🔑 Cache Keys:"
	@docker exec spooliq_redis redis-cli --no-auth-warning -a redis123 keys "cache:*" | head -10 || echo "No cache keys found"

# Clear cache
cache/clear:
	@echo "🧹 Clearing cache..."
	@docker exec spooliq_redis redis-cli --no-auth-warning -a redis123 flushdb
	@echo "✅ Cache cleared!"

# Database operations
db/migrate:
	@echo "📊 Running database migrations..."
	@$(GORUN) cmd/migrate.go up
	@echo "✅ Migrations completed!"

# Rollback last migration
db/rollback:
	@echo "⏪ Rolling back last migration..."
	@$(GORUN) cmd/migrate.go down
	@echo "✅ Rollback completed!"

# Show migration status
db/status:
	@echo "📋 Checking migration status..."
	@$(GORUN) cmd/migrate.go status

# Show pending migrations (dry run)
db/dry-run:
	@echo "👁️  Showing pending migrations (dry run)..."
	@$(GORUN) cmd/migrate.go dry-run

# Seed database with initial data
db/seed:
	@echo "🌱 Seeding database with initial data..."
	@echo "Note: Seeds run automatically on app start"
	@$(GORUN) main.go --seed-only
	@echo "✅ Database seeded!"

# Reset database (drop and recreate)
db/reset:
	@echo "⚠️  WARNING: This will delete all data!"
	@echo "🗑️  Resetting database..."
	@rm -f ./spooliq.db
	@echo "🔄 Database reset completed. Run 'make run' to recreate with fresh data."

# Docker image analysis
docker/size:
	@echo "🐳 Docker Image Size Analysis:"
	@echo ""
	@echo "📊 Image sizes:"
	@docker images | grep spooliq || echo "No spooliq images found"
	@echo ""
	@echo "📋 Image layers (latest):"
	@docker history spooliq:latest --format "table {{.CreatedBy}}\t{{.Size}}" | head -10 || echo "No spooliq:latest image found"

# Build optimized image
docker/build:
	@echo "🔨 Building ultra-optimized Docker image (scratch + UPX)..."
	@$(DOCKER_COMPOSE) build --no-cache
	@echo ""
	@echo "✅ Build complete! Ultra-minimal image size:"
	@docker images spooliq --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"
	@echo ""
	@echo "🎯 Expected size: ~500KB-1MB (with UPX compression)"

# Build alternative image with CA certificates
docker/build-ca:
	@echo "🔨 Building optimized image with CA certificates..."
	@docker build \
		--build-arg GITHUB_TOKEN=${GITHUB_TOKEN} \
		--build-arg VERSION=${VERSION:-latest} \
		-f dockerfile.with-ca \
		-t spooliq:with-ca .
	@echo ""
	@echo "✅ Build complete! Image with CA certs:"
	@docker images spooliq:with-ca --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"
	@echo ""
	@echo "🎯 Expected size: ~1-3MB (distroless + UPX)"

# Analyze Docker image
docker/analyze:
	@echo "🔍 Analyzing Docker image..."
	@echo ""
	@echo "📊 Image information:"
	@docker inspect spooliq:latest | jq '.[0] | {Size: .Size, Architecture: .Architecture, Os: .Os}' || echo "No spooliq:latest image found"
	@echo ""
	@echo "📋 Layer breakdown:"
	@docker history spooliq:latest --format "table {{.CreatedBy}}\t{{.Size}}\t{{.CreatedSince}}" || echo "No image found"

# Clean Docker resources
docker/clean:
	@echo "🧹 Cleaning Docker resources..."
	@docker system prune -f
	@docker builder prune -f
	@echo "✅ Docker cleanup complete!"

# Display help
help:
	@echo "Available commands:"
	@echo ""
	@echo "🔨 Build & Development:"
	@echo "  make build                - Build the application"
	@echo "  make run                  - Run the application locally"
	@echo "  make test                 - Run the tests"
	@echo "  make clean                - Clean the binary"
	@echo "  make lint                 - Run the linter"
	@echo ""
	@echo "🏗️  Infrastructure:"
	@echo "  make infrastructure/raise - Start PostgreSQL + RabbitMQ + Keycloak + Redis"
	@echo "  make infrastructure/down  - Stop infrastructure services"
	@echo "  make infrastructure/logs  - Show infrastructure logs"
	@echo "  make infrastructure/restart - Restart infrastructure services"
	@echo ""
	@echo "🚀 Full Application:"
	@echo "  make app/run              - Start full application + infrastructure"
	@echo "  make app/down             - Stop full application"
	@echo "  make app/logs             - Show application logs"
	@echo ""
	@echo "🗄️  Cache Commands:"
	@echo "  make cache/test           - Test cache endpoints"
	@echo "  make cache/status         - Show Redis status and cache keys"
	@echo "  make cache/clear          - Clear all cache"
	@echo ""
	@echo "🗄️  Database Commands:"
	@echo "  make db/migrate           - Run database migrations"
	@echo "  make db/rollback          - Rollback last migration"
	@echo "  make db/status            - Show migration status"
	@echo "  make db/dry-run           - Show pending migrations (dry run)"
	@echo "  make db/seed              - Seed database with initial data"
	@echo "  make db/reset             - Reset database (SQLite only)"
	@echo ""
	@echo "🐳 Docker Commands:"
	@echo "  make docker/build         - Build ultra-optimized image (scratch + UPX)"
	@echo "  make docker/build-ca      - Build optimized image with CA certificates"
	@echo "  make docker/size          - Show Docker image size analysis"
	@echo "  make docker/analyze       - Detailed Docker image analysis"
	@echo "  make docker/clean         - Clean Docker build cache and unused resources"
	@echo ""
	@echo "  make help                 - Display this help message"

