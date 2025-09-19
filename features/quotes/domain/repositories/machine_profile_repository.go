package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/quotes/domain/entities"
)

type MachineProfileRepository interface {
	Create(ctx context.Context, profile *entities.MachineProfile) error
	GetByID(ctx context.Context, id uint, userID *string) (*entities.MachineProfile, error)
	GetAll(ctx context.Context, userID *string) ([]*entities.MachineProfile, error)
	Update(ctx context.Context, profile *entities.MachineProfile, userID string) error
	Delete(ctx context.Context, id uint, userID string) error
	GetByOwner(ctx context.Context, userID string) ([]*entities.MachineProfile, error)
	GetGlobal(ctx context.Context) ([]*entities.MachineProfile, error)
}
