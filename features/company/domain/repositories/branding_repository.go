package repositories

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/company/domain/entities"
)

// BrandingRepository defines the interface for branding data operations
type BrandingRepository interface {
	// FindByOrganizationID retrieves branding configuration by organization ID
	FindByOrganizationID(ctx context.Context, orgID string) (*entities.CompanyBrandingEntity, error)

	// Create creates a new branding configuration
	Create(ctx context.Context, branding *entities.CompanyBrandingEntity) error

	// Update updates an existing branding configuration
	Update(ctx context.Context, branding *entities.CompanyBrandingEntity) error

	// GetTemplates returns all available branding templates
	GetTemplates() []entities.BrandingTemplate
}
