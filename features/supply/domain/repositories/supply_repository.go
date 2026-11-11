package repositories

import (
	"github.com/RodolfoBonis/spooliq/features/supply/domain/entities"
	"github.com/google/uuid"
)

// SupplyRepository defines the contract for supply data access operations.
type SupplyRepository interface {
	Create(supply *entities.SupplyEntity) error
	Update(supply *entities.SupplyEntity) error
	Delete(id uuid.UUID) error
	FindByID(id uuid.UUID, organizationID string) (*entities.SupplyEntity, error)
	// FindAll returns a page of supplies for the organization. search is an
	// optional case-insensitive filter on the supply name; isActive, when non-nil,
	// filters by the active flag; order is a safe ORDER BY clause (already
	// validated against a whitelist); limit/offset paginate. It also returns the
	// total row count matching the filter.
	FindAll(organizationID, search string, isActive *bool, order string, limit, offset int) ([]entities.SupplyEntity, int64, error)
	Exists(name string, organizationID string) (bool, error)
}
