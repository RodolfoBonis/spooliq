// Package entities holds the domain entities and DTOs for filament stock control.
package entities

import (
	"time"

	"github.com/google/uuid"
)

// MovementType enumerates the kinds of stock movement.
type MovementType string

// Movement types. purchase/adjustment/waste are user-driven; consumption is written
// only by the system when a budget is completed.
const (
	// MovementPurchase adds stock (grams > 0). May carry a unit price.
	MovementPurchase MovementType = "purchase"
	// MovementAdjustment is a signed correction (grams != 0, positive or negative).
	MovementAdjustment MovementType = "adjustment"
	// MovementWaste removes stock. The request grams are positive; it is stored negative.
	MovementWaste MovementType = "waste"
	// MovementConsumption is system-only, written when a budget is completed. Stored negative.
	MovementConsumption MovementType = "consumption"
)

// IsManual reports whether a movement type may be created through the public
// stock-movement endpoint. consumption is excluded (system only).
// IsKnown reports whether t is any valid movement type (manual or system).
func (t MovementType) IsKnown() bool {
	return t.IsManual() || t == MovementConsumption
}

func (t MovementType) IsManual() bool {
	switch t {
	case MovementPurchase, MovementAdjustment, MovementWaste:
		return true
	default:
		return false
	}
}

// StockMovementEntity is a single row of the filament stock ledger.
type StockMovementEntity struct {
	ID             uuid.UUID    `json:"id"`
	OrganizationID string       `json:"organization_id"`
	FilamentID     uuid.UUID    `json:"filament_id"`
	Type           MovementType `json:"type"`
	// Grams is the signed delta applied to the filament balance (negative for waste
	// and consumption).
	Grams int64 `json:"grams"`
	// UnitPricePerKg is the purchase price in cents per kg (purchase movements only).
	UnitPricePerKg *int64 `json:"unit_price_per_kg"`
	// BudgetID links a consumption movement back to the budget that caused it.
	BudgetID  *uuid.UUID `json:"budget_id"`
	Note      *string    `json:"note"`
	CreatedBy string     `json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
}

// StockMovementResponse is the wire shape of a ledger row. budget_quote_number is
// included only when the movement is linked to a budget that has a quote number.
type StockMovementResponse struct {
	ID                uuid.UUID    `json:"id"`
	FilamentID        uuid.UUID    `json:"filament_id"`
	Type              MovementType `json:"type"`
	Grams             int64        `json:"grams"`
	UnitPricePerKg    *int64       `json:"unit_price_per_kg"`
	BudgetID          *uuid.UUID   `json:"budget_id"`
	BudgetQuoteNumber *int         `json:"budget_quote_number,omitempty"`
	Note              *string      `json:"note"`
	CreatedBy         string       `json:"created_by"`
	CreatedAt         time.Time    `json:"created_at"`
}

// FilamentStockSummary is the compact filament view returned alongside a created
// movement so the client can refresh the balance without a second request.
type FilamentStockSummary struct {
	ID                     uuid.UUID `json:"id"`
	Name                   string    `json:"name"`
	Color                  string    `json:"color"`
	StockGrams             int64     `json:"stock_grams"`
	TrackStock             bool      `json:"track_stock"`
	LowStockThresholdGrams *int      `json:"low_stock_threshold_grams"`
	IsLowStock             bool      `json:"is_low_stock"`
}

// CreateMovementRequest is the POST body for a manual stock movement.
type CreateMovementRequest struct {
	Type MovementType `json:"type" validate:"required"`
	// Grams is the movement size. Sign/positivity rules depend on Type and are
	// validated in the use case (see ValidateCreate).
	Grams int64   `json:"grams"`
	Note  *string `json:"note" validate:"omitempty,max=500"`
	// UnitPricePerKg is accepted only for purchase movements (cents, >= 0).
	UnitPricePerKg *int64 `json:"unit_price_per_kg" validate:"omitempty,gte=0"`
}

// CreateMovementResponse is the 201 body: the new ledger row plus the refreshed
// filament stock summary.
type CreateMovementResponse struct {
	Movement StockMovementResponse `json:"movement"`
	Filament FilamentStockSummary  `json:"filament"`
}

// ListMovementsResponse is the paginated list envelope for stock movements. It
// mirrors helpers.Page[StockMovementResponse] as a concrete type for Swagger.
type ListMovementsResponse struct {
	Data       []StockMovementResponse `json:"data"`
	Total      int64                   `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
}

// LowStockFilament is a tracked filament at or below its low-stock threshold.
type LowStockFilament struct {
	ID         uuid.UUID
	Name       string
	Color      string
	StockGrams int64
}
