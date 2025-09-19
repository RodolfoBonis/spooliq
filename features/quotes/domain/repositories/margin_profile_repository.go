package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/quotes/domain/entities"
)

type MarginProfileRepository interface {
	Create(ctx context.Context, profile *entities.MarginProfile) error
	GetByID(ctx context.Context, id uint, userID *string) (*entities.MarginProfile, error)
	GetAll(ctx context.Context, userID *string) ([]*entities.MarginProfile, error)
	Update(ctx context.Context, profile *entities.MarginProfile, userID string) error
	Delete(ctx context.Context, id uint, userID string) error
	GetByOwner(ctx context.Context, userID string) ([]*entities.MarginProfile, error)
	GetGlobal(ctx context.Context) ([]*entities.MarginProfile, error)
}
