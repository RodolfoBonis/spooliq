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
	FindByID(id uuid.UUID, organizationID string) (*entities.MaterialEntity, error)
	// FindAll returns a page of materials for the organization. search is an
	// optional case-insensitive filter on the material name; order is a safe
	// ORDER BY clause (already validated against a whitelist); limit/offset
	// paginate. It also returns the total row count matching the filter.
	FindAll(organizationID, search, order string, limit, offset int) ([]entities.MaterialEntity, int64, error)
	Exists(name string, organizationID string) (bool, error)
}
