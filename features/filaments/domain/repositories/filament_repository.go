package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/filaments/domain/entities"
)

// FilamentRepository defines the interface for filament data access operations.
type FilamentRepository interface {
	Create(ctx context.Context, filament *entities.Filament) error
	GetByID(ctx context.Context, id uint, userID *string) (*entities.Filament, error)
	GetByIDWithUserCheck(ctx context.Context, id uint, userID *string, username string) (*entities.Filament, error)
	GetAll(ctx context.Context, userID *string) ([]*entities.Filament, error)
	Update(ctx context.Context, filament *entities.Filament, userID *string) error
	Delete(ctx context.Context, id uint, userID *string) error
	GetByOwner(ctx context.Context, userID string) ([]*entities.Filament, error)
	GetGlobal(ctx context.Context) ([]*entities.Filament, error)
}
