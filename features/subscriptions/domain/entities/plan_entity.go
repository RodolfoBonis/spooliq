package entities

import (
	"time"

	"github.com/google/uuid"
)

// PlanEntity represents a subscription plan in the domain layer
type PlanEntity struct {
	ID          uuid.UUID      `json:"id"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"` // starter, professional, enterprise
	Description string         `json:"description"`
	Price       float64        `json:"price"`
	Currency    string         `json:"currency"` // BRL, USD, etc
	Interval    string         `json:"interval"` // MONTHLY, YEARLY
	Active      bool           `json:"active"`
	Popular     bool           `json:"popular"`
	Recommended bool           `json:"recommended"`
	SortOrder   int            `json:"sort_order"`
	Features    []PlanFeature  `json:"features,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// PlanFeature represents a feature included in a plan
type PlanFeature struct {
	ID          uuid.UUID `json:"id"`
	PlanID      uuid.UUID `json:"plan_id"`
	Name        string    `json:"name"`
	Key         string    `json:"key"` // Unique identifier like "max_users", "storage_gb"
	Description string    `json:"description"`
	Value       string    `json:"value"` // Stored as string, can be "5", "true", "unlimited", etc
	ValueType   string    `json:"value_type"` // number, boolean, text
	Available   bool      `json:"available"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
