package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/company/domain/entities"
	"github.com/google/uuid"
)

// CompanyRepository defines the interface for company data operations
type CompanyRepository interface {
	Create(ctx context.Context, company *entities.CompanyEntity) error
	FindByOrganizationID(ctx context.Context, organizationID string) (*entities.CompanyEntity, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entities.CompanyEntity, error)
	Update(ctx context.Context, company *entities.CompanyEntity) error
	Delete(ctx context.Context, id uuid.UUID) error
	ExistsByOrganizationID(ctx context.Context, organizationID string) (bool, error)

	// Admin operations
	FindAllPaginated(ctx context.Context, page, pageSize int, statusFilter string) ([]*entities.CompanyEntity, int64, error)
	FindAllActive(ctx context.Context) ([]*entities.CompanyEntity, error)
}
