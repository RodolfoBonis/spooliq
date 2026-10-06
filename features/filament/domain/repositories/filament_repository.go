package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/filament/domain/entities"
	"github.com/google/uuid"
)

// FilamentRepository defines the interface for filament data operations
type FilamentRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, filament *entities.FilamentEntity) error
	FindByID(ctx context.Context, id uuid.UUID, organizationID string) (*entities.FilamentEntity, error)
	Update(ctx context.Context, filament *entities.FilamentEntity) error
	Delete(ctx context.Context, id uuid.UUID) error

	// List operations. search is an optional case-insensitive term matched
	// against the filament name (and, via join, the brand/material names).
	// order is a safe ORDER BY clause already validated against a whitelist.
	FindAll(ctx context.Context, organizationID, search, order string, limit, offset int) ([]*entities.FilamentEntity, int64, error)

	// Search operations. filters carries the structured filters (brand_id,
	// material_id, color_type, diameter, min/max price); search/order behave as
	// in FindAll.
	SearchFilaments(ctx context.Context, organizationID string, filters map[string]interface{}, search, order string, limit, offset int) ([]*entities.FilamentEntity, int64, error)

	// Validation operations
	ExistsByNameAndBrand(ctx context.Context, name string, brandID uuid.UUID, excludeID *uuid.UUID) (bool, error)

	// Relationship helpers (single lookups)
	GetBrandInfo(ctx context.Context, brandID uuid.UUID) (*entities.BrandInfo, error)
	GetMaterialInfo(ctx context.Context, materialID uuid.UUID) (*entities.MaterialInfo, error)

	// Batch relationship helpers used by list endpoints to avoid N+1 queries:
	// they fetch every requested brand/material in a single query and return a
	// map keyed by ID.
	GetBrandsInfo(ctx context.Context, brandIDs []uuid.UUID) (map[uuid.UUID]*entities.BrandInfo, error)
	GetMaterialsInfo(ctx context.Context, materialIDs []uuid.UUID) (map[uuid.UUID]*entities.MaterialInfo, error)
}
