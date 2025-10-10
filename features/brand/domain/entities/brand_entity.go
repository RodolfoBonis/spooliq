package entities

import (
	"time"

	"github.com/google/uuid"
)

// BrandEntity represents a filament brand in the domain layer.
type BrandEntity struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID string     `json:"organization_id"` // Multi-tenancy
	Name           string     `json:"name"`
	Description    string     `json:"description,omitempty"`
	CreatedAt      time.Time  `json:"created_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at,omitempty"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}
