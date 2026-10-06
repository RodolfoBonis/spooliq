package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/model3d/domain/entities"
	"github.com/google/uuid"
)

// Model3DFilters holds the structured list filters for 3D models. Both fields are
// optional; a nil/empty value disables that filter.
type Model3DFilters struct {
	// Format restricts results to a file format (".stl" or ".3mf").
	Format string
	// CustomerID restricts results to a single customer.
	CustomerID *uuid.UUID
}

// Model3DRepository defines the contract for 3D model data access operations. All
// methods are organization-scoped and context-aware.
type Model3DRepository interface {
	Create(ctx context.Context, model *entities.Model3DEntity) error
	Update(ctx context.Context, model *entities.Model3DEntity) error
	Delete(ctx context.Context, id uuid.UUID, organizationID string) error
	FindByID(ctx context.Context, id uuid.UUID, organizationID string) (*entities.Model3DEntity, error)
	FindAll(ctx context.Context, organizationID string, filters Model3DFilters, search, order string, limit, offset int) ([]*entities.Model3DEntity, int64, error)
	FindByCustomerID(ctx context.Context, customerID uuid.UUID, organizationID string) ([]*entities.Model3DEntity, error)
	// FindByHash returns the live model with the given file hash in the org, or
	// (nil, nil) when none exists.
	FindByHash(ctx context.Context, hash string, organizationID string) (*entities.Model3DEntity, error)
	// CustomerExists reports whether the customer belongs to the organization.
	CustomerExists(ctx context.Context, customerID uuid.UUID, organizationID string) (bool, error)
}
