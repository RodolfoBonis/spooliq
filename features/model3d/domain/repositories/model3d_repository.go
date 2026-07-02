package repositories

import (
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/entities"
	"github.com/google/uuid"
)

// Model3DFilters represents the query filters for listing 3D models.
type Model3DFilters struct {
	Search     string
	CustomerID *uuid.UUID
	Format     string
	Page       int
	PageSize   int
}

// Model3DRepository defines the contract for 3D model data access operations.
type Model3DRepository interface {
	Create(model *entities.Model3DEntity) error
	Update(model *entities.Model3DEntity) error
	Delete(id uuid.UUID) error
	FindByID(id uuid.UUID, organizationID string) (*entities.Model3DEntity, error)
	FindAll(organizationID string, filters Model3DFilters) (*entities.FindAllModel3DResponse, error)
	FindByCustomerID(customerID uuid.UUID, organizationID string) ([]*entities.Model3DEntity, error)
	FindByHash(hash string, organizationID string) (*entities.Model3DEntity, error)
}
