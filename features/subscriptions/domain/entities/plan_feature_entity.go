package entities

import (
	"time"

	"github.com/google/uuid"
)

// PlanFeatureEntity represents a feature of a subscription plan in the domain layer
type PlanFeatureEntity struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}
