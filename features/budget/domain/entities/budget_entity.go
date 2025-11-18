package entities

import (
	"time"

	"github.com/google/uuid"
)

// BudgetStatus represents the status of a budget
type BudgetStatus string

// Budget status constants
const (
	StatusDraft     BudgetStatus = "draft"     // StatusDraft represents a budget in draft state
	StatusSent      BudgetStatus = "sent"      // StatusSent represents a budget sent to customer
	StatusApproved  BudgetStatus = "approved"  // StatusApproved represents an approved budget
	StatusRejected  BudgetStatus = "rejected"  // StatusRejected represents a rejected budget
	StatusPrinting  BudgetStatus = "printing"  // StatusPrinting represents a budget currently being printed
	StatusCompleted BudgetStatus = "completed" // StatusCompleted represents a completed budget
)

// BudgetEntity represents a budget/quote in the domain layer
type BudgetEntity struct {
	ID             uuid.UUID    `json:"id"`
	OrganizationID string       `json:"organization_id"` // Multi-tenancy
	Name           string       `json:"name"`
	Description    string       `json:"description,omitempty"`
	CustomerID     uuid.UUID    `json:"customer_id"`
	Status         BudgetStatus `json:"status"`

	// Print time (manual input for now)
	PrintTimeHours   int `json:"print_time_hours"`
	PrintTimeMinutes int `json:"print_time_minutes"`

	// Presets used for calculations
	MachinePresetID *uuid.UUID `json:"machine_preset_id,omitempty"`
	EnergyPresetID  *uuid.UUID `json:"energy_preset_id,omitempty"`
	CostPresetID    *uuid.UUID `json:"cost_preset_id,omitempty"` // For overhead/profit percentages

	// Configuration flags
	IncludeEnergyCost bool `json:"include_energy_cost"`
	IncludeWasteCost  bool `json:"include_waste_cost"`

	// Calculated costs (in cents for precision)
	FilamentCost int64 `json:"filament_cost"` // cents - Sum of all items filament costs
	WasteCost    int64 `json:"waste_cost"`    // cents - Sum of all items waste costs
	EnergyCost   int64 `json:"energy_cost"`   // cents - Sum of all items energy costs
	SetupCost    int64 `json:"setup_cost"`    // cents - Sum of all items setup costs
	LaborCost    int64 `json:"labor_cost"`    // cents - Sum of all items manual labor costs
	OverheadCost int64 `json:"overhead_cost"` // cents - Overhead calculated on subtotal (from CostPreset.OverheadPercentage)
	ProfitAmount int64 `json:"profit_amount"` // cents - Profit margin calculated (from CostPreset.ProfitMarginPercentage)
	TotalCost    int64 `json:"total_cost"`    // cents - Final total: Filament + Waste + Energy + Setup + Labor + Overhead + Profit

	// Additional fields for PDF generation
	DeliveryDays *int    `json:"delivery_days,omitempty"` // prazo de entrega em dias
	PaymentTerms *string `json:"payment_terms,omitempty"` // condições de pagamento
	Notes        *string `json:"notes,omitempty"`         // observações adicionais
	PDFUrl       *string `json:"pdf_url,omitempty"`       // URL do PDF gerado

	// Ownership
	OwnerUserID string `json:"owner_user_id"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// IsValidTransition checks if a status transition is valid
func (b *BudgetEntity) IsValidTransition(newStatus BudgetStatus) bool {
	validTransitions := map[BudgetStatus][]BudgetStatus{
		StatusDraft:     {StatusSent},
		StatusSent:      {StatusApproved, StatusRejected},
		StatusApproved:  {StatusPrinting},
		StatusRejected:  {StatusDraft}, // Allow reopening
		StatusPrinting:  {StatusCompleted},
		StatusCompleted: {}, // No transitions from completed
	}

	allowedTransitions, exists := validTransitions[b.Status]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == newStatus {
			return true
		}
	}

	return false
}

// CanBeEdited checks if the budget can be fully edited (only draft budgets)
func (b *BudgetEntity) CanBeEdited() bool {
	return b.Status == StatusDraft
}

// CanBeDeleted checks if the budget can be deleted
func (b *BudgetEntity) CanBeDeleted() bool {
	return b.Status != StatusPrinting && b.Status != StatusCompleted
}
