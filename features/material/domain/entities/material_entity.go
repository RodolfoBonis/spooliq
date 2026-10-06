package entities

import (
	"time"

	"github.com/google/uuid"
)

// MaterialEntity represents a 3D printing material domain entity.
//
// TempTable and TempExtruder serialize as snake_case (temp_table/temp_extruder)
// WITHOUT omitempty so a legitimate 0 °C is preserved in the payload rather than
// silently dropped.
type MaterialEntity struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID string     `json:"organization_id"` // Multi-tenancy
	Name           string     `json:"name"`
	Description    string     `json:"description,omitempty"`
	TempTable      float32    `json:"temp_table"`
	TempExtruder   float32    `json:"temp_extruder"`
	CreatedAt      time.Time  `json:"created_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at,omitempty"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}
