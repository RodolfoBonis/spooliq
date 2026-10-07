package entities

import (
	"time"
)

// ActivityAction represents the type of action performed.
type ActivityAction string

// Activity actions
const (
	ActionCreated       ActivityAction = "created"
	ActionUpdated       ActivityAction = "updated"
	ActionDeleted       ActivityAction = "deleted"
	ActionStatusChanged ActivityAction = "status_changed"
	ActionApproved      ActivityAction = "approved"
	ActionRejected      ActivityAction = "rejected"
)

// ActivityEntityType represents the type of entity affected.
type ActivityEntityType string

// Entity types
const (
	EntityBrand    ActivityEntityType = "brand"
	EntityCustomer ActivityEntityType = "customer"
	EntityFilament ActivityEntityType = "filament"
	EntityMaterial ActivityEntityType = "material"
	EntityBudget   ActivityEntityType = "budget"
	EntityPreset   ActivityEntityType = "preset"
	EntityModel3D  ActivityEntityType = "model3d"
	// EntityStockMovement is a filament stock ledger entry (Phase 4C).
	EntityStockMovement ActivityEntityType = "stock_movement"
)

// ActivityEntity represents a recorded activity.
type ActivityEntity struct {
	ID             string             `json:"id"`
	OrganizationID string             `json:"organization_id"`
	UserID         string             `json:"user_id"`
	Action         ActivityAction     `json:"action"`
	EntityType     ActivityEntityType `json:"entity_type"`
	EntityID       string             `json:"entity_id"`
	EntityName     string             `json:"entity_name"`
	Description    string             `json:"description,omitempty"`
	Metadata       map[string]any     `json:"metadata,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
}

// ActivityFilter defines filters for listing activities.
type ActivityFilter struct {
	OrganizationID string
	EntityType     *ActivityEntityType
	Action         *ActivityAction
	Page           int
	PageSize       int
}

// PaginatedActivities represents paginated activity results.
type PaginatedActivities struct {
	Activities []ActivityEntity `json:"activities"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}
