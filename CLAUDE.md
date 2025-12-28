# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SpoolIQ is a Go-based API for calculating 3D printing costs, supporting multi-color filament calculations, energy costs, wear and overhead, and labor. Built with Gin framework, Uber FX for dependency injection, and multi-tenant architecture with Keycloak authentication.

## Essential Commands

```bash
make run                    # Run application locally (triggers AutoMigration)
make build                  # Build the binary
make test                   # Run all tests
make lint                   # Run linters (gofmt, go vet, golint, staticcheck, goimports)
make infrastructure/raise   # Start PostgreSQL, Redis, RabbitMQ, Keycloak
make infrastructure/down    # Stop infrastructure services
bash scripts/generate-swagger.sh  # Generate Swagger docs
```

Run specific tests:
```bash
go test ./features/auth/...     # Test specific module
go test -v -run TestName ./...  # Run single test by name
```

## Architecture

### Clean Architecture with FX Dependency Injection

Each feature module follows this structure:
```
features/<domain>/
├── di/                    # FX module with dependency wiring
│   └── <domain>_di.go     # fx.Module providing repositories and use cases
├── data/
│   ├── models/            # GORM database models (snake_case table names)
│   └── repositories/      # Repository implementations
├── domain/
│   ├── entities/          # Domain entities and DTOs
│   ├── repositories/      # Repository interfaces
│   └── usecases/          # Business logic (one file per use case)
└── routes.go              # HTTP route definitions with Swagger annotations
```

### Adding a New Feature Module

1. Create the domain structure under `features/<name>/`
2. Define the FX module in `di/<name>_di.go`:
```go
var Module = fx.Module("name", fx.Provide(
    fx.Annotate(func(db *gorm.DB) domainRepos.Repository {
        return repositories.NewRepository(db)
    }),
    fx.Annotate(func(repo domainRepos.Repository, logger logger.Logger) usecases.IUseCase {
        return usecases.NewUseCase(repo, logger)
    }),
))
```
3. Register module in `app/fx.go`
4. Add routes in `routes/router.go`
5. Add model to `core/services/database_service.go` AutoMigration (respect FK order)

### Multi-Tenant Architecture

All tenant-scoped tables use `organization_id` (UUID) as the tenant identifier:
- Extract from JWT context via `helpers.GetOrganizationID(c)`
- Foreign key references `companies(organization_id)` with `ON DELETE RESTRICT`
- Models must include `OrganizationID string` field with appropriate GORM tags

### Role-Based Access Control

Roles defined in `core/roles/roles.go` must match Keycloak realm roles:
- `User` - Basic user access
- `OrgAdmin` - Organization administrator
- `Owner` - Organization owner
- `PlatformAdmin` - System-wide admin

Use `protectFactory(handler, roles.OwnerRole, roles.OrgAdminRole)` in routes.

### Database Migrations

GORM AutoMigration runs on startup. Migration order in `database_service.go:RunMigrations()` is critical:
1. Independent tables (subscription_plans)
2. Companies (root tenant table)
3. Tables with FK to companies only (users, brands, materials)
4. Tables with multiple FKs (filaments, customers, presets)
5. Complex hierarchies (budgets → budget_items → budget_item_filaments)

### Handler Patterns

Two patterns exist:
1. **UseCase-as-Handler**: Use case methods directly as `gin.HandlerFunc` (see `features/brand/routes.go`)
2. **Dedicated Handler**: Separate Handler struct with use case dependencies (see `features/preset/routes.go`)

### Swagger Annotations

Add annotations above handler functions:
```go
// @Summary Create brand
// @Tags Brands
// @Accept json
// @Produce json
// @Param request body entities.UpsertBrandRequest true "Brand data"
// @Success 201 {object} entities.BrandEntity
// @Failure 400 {object} errors.HTTPError
// @Security BearerAuth
// @Router /brands [post]
```

## Key Files

- `app/fx.go` - Central FX app configuration, all modules registered here
- `routes/router.go` - All route registrations
- `core/services/database_service.go` - Database connection and migrations
- `core/middlewares/auth_middleware.go` - JWT validation and context injection
- `core/helpers/context_helpers.go` - Extract user/org info from request context

## Service Dependencies (via Docker Compose)

- PostgreSQL :5432 (config from .env)
- Redis :6379 (password: redis123)
- RabbitMQ :5672/:15672 (admin/admin123)
- Keycloak :8180 (admin/admin123, realm: spooliq-realm)