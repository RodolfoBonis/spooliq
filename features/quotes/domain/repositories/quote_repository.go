package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/quotes/domain/entities"
)

type QuoteRepository interface {
	Create(ctx context.Context, quote *entities.Quote) error
	GetByID(ctx context.Context, id uint, userID string) (*entities.Quote, error)
	GetByUser(ctx context.Context, userID string) ([]*entities.Quote, error)
	Update(ctx context.Context, quote *entities.Quote, userID string) error
	Delete(ctx context.Context, id uint, userID string) error
	GetWithFilamentLines(ctx context.Context, id uint, userID string) (*entities.Quote, error)
}
