package repositories

import (
	"github.com/RodolfoBonis/spooliq/features/material/domain/entities"
	"github.com/google/uuid"
)

// MaterialRepository defines the contract for material data access operations.
type MaterialRepository interface {
	Create(material *entities.MaterialEntity) error
	Update(material *entities.MaterialEntity) error
	Delete(id uuid.UUID) error
	FindByID(id uuid.UUID) (*entities.MaterialEntity, error)
	FindAll() ([]entities.MaterialEntity, error)
	Exists(name string) (bool, error)
}
