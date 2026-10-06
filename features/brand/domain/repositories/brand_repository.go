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
	FindByID(id uuid.UUID, organizationID string) (*entities.BrandEntity, error)
	// FindAll returns a page of brands for the organization. search is an
	// optional case-insensitive filter on the brand name; order is a safe
	// ORDER BY clause (already validated against a whitelist); limit/offset
	// paginate. It also returns the total row count matching the filter.
	FindAll(organizationID, search, order string, limit, offset int) ([]entities.BrandEntity, int64, error)
	Exists(name string, organizationID string) (bool, error)
}
