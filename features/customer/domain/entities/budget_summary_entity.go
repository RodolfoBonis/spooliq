package entities

import (
	"time"

	"github.com/google/uuid"
)

// BudgetSummary represents a simplified budget information for customer details
type BudgetSummary struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	TotalCost int64     `json:"total_cost"`
	CreatedAt time.Time `json:"created_at"`
}
