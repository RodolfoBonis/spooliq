package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/filament-metadata/domain/entities"
)

// MaterialRepository defines operations for material management
type MaterialRepository interface {
	Create(ctx context.Context, material *entities.FilamentMaterial) error
	GetByID(ctx context.Context, id uint) (*entities.FilamentMaterial, error)
	GetByName(ctx context.Context, name string) (*entities.FilamentMaterial, error)
	GetAll(ctx context.Context, activeOnly bool) ([]*entities.FilamentMaterial, error)
	Update(ctx context.Context, material *entities.FilamentMaterial) error
	Delete(ctx context.Context, id uint) error
	Exists(ctx context.Context, name string) (bool, error)
}