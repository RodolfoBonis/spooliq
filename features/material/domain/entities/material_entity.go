package entities

import (
	"time"

	"github.com/google/uuid"
)

// MaterialEntity represents a 3D printing material domain entity
type MaterialEntity struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID string     `json:"organization_id"` // Multi-tenancy
	Name           string     `json:"name"`
	Description    string     `json:"description,omitempty"`
	TempTable      float32    `json:"tempTable,omitempty"`
	TempExtruder   float32    `json:"tempExtruder,omitempty"`
	CreatedAt      time.Time  `json:"created_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at,omitempty"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}
