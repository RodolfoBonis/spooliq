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

// CustomerBudgetStats aggregates per-customer budget figures computed by a
// single grouped query. Count is the number of budgets; Total is the summed
// total_cost (in cents) restricted to a given set of statuses.
type CustomerBudgetStats struct {
	Count int64
	Total int64
}
