package entities

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionPlanEntity represents a subscription plan in the domain layer
type SubscriptionPlanEntity struct {
	ID          uuid.UUID           `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Price       float64             `json:"price"`
	Cycle       string              `json:"cycle"` // MONTHLY, YEARLY
	Features    []PlanFeatureEntity `json:"features"`
	IsActive    bool                `json:"is_active"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	DeletedAt   *time.Time          `json:"deleted_at,omitempty"`
}

// Subscription plan cycles
const (
	CycleMonthly = "MONTHLY"
	CycleYearly  = "YEARLY"
)

// SubscriptionPlanCreateRequest represents the request to create a plan
type SubscriptionPlanCreateRequest struct {
	Name        string                     `json:"name" binding:"required"`
	Description string                     `json:"description"`
	Price       float64                    `json:"price" binding:"required,gt=0"`
	Cycle       string                     `json:"cycle" binding:"required,oneof=MONTHLY YEARLY"`
	Features    []PlanFeatureCreateRequest `json:"features"`
}

// PlanFeatureCreateRequest represents a feature to be created with a plan
type PlanFeatureCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

// SubscriptionPlanUpdateRequest represents the request to update a plan
type SubscriptionPlanUpdateRequest struct {
	Name        *string                    `json:"name"`
	Description *string                    `json:"description"`
	Price       *float64                   `json:"price" binding:"omitempty,gt=0"`
	Cycle       *string                    `json:"cycle" binding:"omitempty,oneof=MONTHLY YEARLY"`
	Features    []PlanFeatureCreateRequest `json:"features"` // Replace all features
	IsActive    *bool                      `json:"is_active"`
}

// SubscriptionPlanResponse represents the response when fetching plans
type SubscriptionPlanResponse struct {
	ID          uuid.UUID           `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Price       float64             `json:"price"`
	Cycle       string              `json:"cycle"`
	Features    []PlanFeatureEntity `json:"features"`
	IsActive    bool                `json:"is_active"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}
