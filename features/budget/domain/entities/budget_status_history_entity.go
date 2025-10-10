package entities

import (
	"time"

	"github.com/google/uuid"
)

// BudgetStatusHistoryEntity represents a status change in a budget's history
type BudgetStatusHistoryEntity struct {
	ID             uuid.UUID    `json:"id"`
	BudgetID       uuid.UUID    `json:"budget_id"`
	PreviousStatus BudgetStatus `json:"previous_status"`
	NewStatus      BudgetStatus `json:"new_status"`
	ChangedBy      string       `json:"changed_by"` // user_id
	Notes          string       `json:"notes,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
}
