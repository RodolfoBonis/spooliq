package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/filament-metadata/domain/entities"
)

// BrandRepository defines operations for brand management
type BrandRepository interface {
	Create(ctx context.Context, brand *entities.FilamentBrand) error
	GetByID(ctx context.Context, id uint) (*entities.FilamentBrand, error)
	GetByName(ctx context.Context, name string) (*entities.FilamentBrand, error)
	GetAll(ctx context.Context, activeOnly bool) ([]*entities.FilamentBrand, error)
	Update(ctx context.Context, brand *entities.FilamentBrand) error
	Delete(ctx context.Context, id uint) error
	Exists(ctx context.Context, name string) (bool, error)
}
