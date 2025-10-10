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
	FindByID(ctx context.Context, id uuid.UUID, userID string, isAdmin bool) (*entities.FilamentEntity, error)
	Update(ctx context.Context, filament *entities.FilamentEntity) error
	Delete(ctx context.Context, id uuid.UUID) error

	// List operations
	FindAll(ctx context.Context, userID string, isAdmin bool, limit, offset int) ([]*entities.FilamentEntity, int, error)

	// Search operations
	SearchFilaments(ctx context.Context, userID string, isAdmin bool, filters map[string]interface{}, limit, offset int) ([]*entities.FilamentEntity, int, error)

	// Validation operations
	ExistsByNameAndBrand(ctx context.Context, name string, brandID uuid.UUID, excludeID *uuid.UUID) (bool, error)

	// Relationship helpers
	GetBrandInfo(ctx context.Context, brandID uuid.UUID) (*entities.BrandInfo, error)
	GetMaterialInfo(ctx context.Context, materialID uuid.UUID) (*entities.MaterialInfo, error)
}
