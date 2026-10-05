// Package repositories defines the data-access contracts for the print profile feature.
package repositories

import (
	"github.com/RodolfoBonis/spooliq/features/profile/domain/entities"
	"github.com/google/uuid"
)

// PrintProfileRepository defines the contract for print profile data operations.
//
// Every read, update and delete is scoped by organizationID to guarantee tenant
// isolation; cross-organization access surfaces as gorm.ErrRecordNotFound so
// callers map it to HTTP 404 (never 403), avoiding leaking other tenants' data.
type PrintProfileRepository interface {
	Create(profile *entities.ProfileEntity) error
	GetByID(id uuid.UUID, organizationID string) (*entities.ProfileEntity, error)
	List(organizationID string) ([]*entities.ProfileEntity, error)
	Update(profile *entities.ProfileEntity) error
	Delete(id uuid.UUID, organizationID string) error

	// SetDefault marks the profile as the single default for its organization,
	// clearing any other default in the same transaction.
	SetDefault(id uuid.UUID, organizationID string) (*entities.ProfileEntity, error)
	// Duplicate copies a profile within the same organization in one transaction.
	// The copy is never a default and takes the given name.
	Duplicate(id uuid.UUID, organizationID string, newName string) (*entities.ProfileEntity, error)
}
