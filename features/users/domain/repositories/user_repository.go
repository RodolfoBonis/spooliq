package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/users/domain/entities"
	"github.com/google/uuid"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	FindAll(ctx context.Context, organizationID string) ([]*entities.UserEntity, error)
	FindByID(ctx context.Context, id uuid.UUID, organizationID string) (*entities.UserEntity, error)
	FindByEmail(ctx context.Context, email string) (*entities.UserEntity, error)
	FindByKeycloakUserID(ctx context.Context, keycloakUserID string) (*entities.UserEntity, error)
	FindOwner(ctx context.Context, organizationID string) (*entities.UserEntity, error)
	Create(ctx context.Context, user *entities.UserEntity) error
	Update(ctx context.Context, id uuid.UUID, organizationID string, user *entities.UserEntity) error
	Delete(ctx context.Context, id uuid.UUID, organizationID string) error
}

