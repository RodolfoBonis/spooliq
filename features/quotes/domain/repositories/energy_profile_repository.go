package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/quotes/domain/entities"
)

type EnergyProfileRepository interface {
	Create(ctx context.Context, profile *entities.EnergyProfile) error
	GetByID(ctx context.Context, id uint, userID *string) (*entities.EnergyProfile, error)
	GetAll(ctx context.Context, userID *string) ([]*entities.EnergyProfile, error)
	Update(ctx context.Context, profile *entities.EnergyProfile, userID string) error
	Delete(ctx context.Context, id uint, userID string) error
	GetByOwner(ctx context.Context, userID string) ([]*entities.EnergyProfile, error)
	GetGlobal(ctx context.Context) ([]*entities.EnergyProfile, error)
}