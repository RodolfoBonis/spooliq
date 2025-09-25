package repositories

import (
	"github.com/RodolfoBonis/spooliq/features/brand/domain/entities"
	"github.com/google/uuid"
)

// BrandRepository defines the contract for brand data access operations.
type BrandRepository interface {
	Create(brand *entities.BrandEntity) error
	Update(brand *entities.BrandEntity) error
	Delete(id uuid.UUID) error
	FindByID(id uuid.UUID) (*entities.BrandEntity, error)
	FindAll() ([]entities.BrandEntity, error)
	Exists(name string) (bool, error)
}
