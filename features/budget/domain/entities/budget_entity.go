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
	StatusExpired   BudgetStatus = "expired"   // StatusExpired represents a sent budget whose validity has passed
	StatusCancelled BudgetStatus = "cancelled" // StatusCancelled represents a cancelled budget
)

// Discount type constants for BudgetEntity.DiscountType.
const (
	DiscountTypePercent = "percent" // DiscountTypePercent: DiscountValue is a 0-100 percentage of base_price
	DiscountTypeFixed   = "fixed"   // DiscountTypeFixed: DiscountValue is a fixed amount in reais
)

// KnownStatuses returns every valid budget status, in lifecycle order. It is the
// single source of truth for validating a status filter on the list endpoint.
func KnownStatuses() []BudgetStatus {
	return []BudgetStatus{
		StatusDraft,
		StatusSent,
		StatusApproved,
		StatusRejected,
		StatusPrinting,
		StatusCompleted,
		StatusExpired,
		StatusCancelled,
	}
}

// IsKnownStatus reports whether s is one of the known budget statuses.
func IsKnownStatus(s string) bool {
	for _, known := range KnownStatuses() {
		if string(known) == s {
			return true
		}
	}
	return false
}

// BudgetEntity represents a budget/quote in the domain layer
type BudgetEntity struct {
	ID             uuid.UUID    `json:"id"`
	OrganizationID string       `json:"organization_id"` // Multi-tenancy
	Name           string       `json:"name"`
	Description    string       `json:"description,omitempty"`
	CustomerID     uuid.UUID    `json:"customer_id"`
	Status         BudgetStatus `json:"status"`

	// QuoteNumber is the sequential, per-organization quote number (starts at 1),
	// assigned on create and duplicate. Nil only for rows not yet backfilled.
	QuoteNumber *int `json:"quote_number,omitempty"`

	// ValidUntil is the date (stored as timestamptz, end-of-day in America/Sao_Paulo)
	// after which a sent budget is considered expired. Nil when no validity is set.
	ValidUntil *time.Time `json:"valid_until,omitempty"`

	// Public share token fields. PublicToken is a base64url (unpadded) random token
	// used by the customer-facing public link; nil when the budget was never shared
	// or the token was revoked.
	PublicToken          *string    `json:"public_token,omitempty"`
	PublicTokenCreatedAt *time.Time `json:"public_token_created_at,omitempty"`

	// Customer response (recorded via the public approve/reject endpoints).
	// ApprovedAt / CompletedAt: last approval and completion instants (dashboard
	// profit is counted by approval date; realized profit by completion date).
	ApprovedAt           *time.Time `json:"approved_at,omitempty"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
	CustomerResponseAt   *time.Time `json:"customer_response_at,omitempty"`
	CustomerResponseName *string    `json:"customer_response_name,omitempty"`
	// CustomerResponseIP / CustomerResponseUserAgent are stored for audit only and
	// are intentionally NOT exposed in API responses.
	CustomerResponseIP        *string `json:"-"`
	CustomerResponseUserAgent *string `json:"-"`
	RejectionReason           *string `json:"rejection_reason,omitempty"`

	// Print time (manual input for now)
	PrintTimeHours   int `json:"print_time_hours"`
	PrintTimeMinutes int `json:"print_time_minutes"`

	// Print profile the presets were resolved from (nil when none was used).
	ProfileID *uuid.UUID `json:"profile_id,omitempty"`

	// Presets used for calculations
	MachinePresetID *uuid.UUID `json:"machine_preset_id,omitempty"`
	EnergyPresetID  *uuid.UUID `json:"energy_preset_id,omitempty"`
	CostPresetID    *uuid.UUID `json:"cost_preset_id,omitempty"` // For overhead/profit percentages

	// Configuration flags
	IncludeEnergyCost  bool `json:"include_energy_cost"`
	IncludeWasteCost   bool `json:"include_waste_cost"`
	IncludeMachineCost bool `json:"include_machine_cost"` // charge per-item machine time (cost_per_hour)

	// Discount configuration (nullable). DiscountType is "percent" or "fixed";
	// DiscountValue is a 0-100 percent or a reais amount accordingly.
	DiscountType  *string  `json:"discount_type,omitempty"`
	DiscountValue *float64 `json:"discount_value,omitempty"`

	// Shipping configuration. When IncludeShipping is set the shipping cost is
	// ShippingOverride (cents) when provided, else computed from the budget cost preset.
	IncludeShipping  bool   `json:"include_shipping"`
	ShippingOverride *int64 `json:"shipping_override,omitempty"` // cents

	// TaxRate is the budget-level "por dentro" tax rate (percent). When nil the
	// company default_tax_rate is used. TaxRateApplied records the rate actually used.
	TaxRate *float64 `json:"tax_rate,omitempty"`

	// Calculated costs (in cents for precision)
	FilamentCost       int64 `json:"filament_cost"`        // cents - Sum of all items filament costs
	WasteCost          int64 `json:"waste_cost"`           // cents - Sum of all items waste costs
	EnergyCost         int64 `json:"energy_cost"`          // cents - Sum of all items energy costs
	MachineCost        int64 `json:"machine_cost"`         // cents - Sum of all items machine-time costs
	SetupCost          int64 `json:"setup_cost"`           // cents - Sum of all items setup costs
	LaborCost          int64 `json:"labor_cost"`           // cents - Sum of all items manual labor costs
	PostProcessingCost int64 `json:"post_processing_cost"` // cents - Sum of all items post-processing costs
	PackagingCost      int64 `json:"packaging_cost"`       // cents - Sum of all items packaging costs
	QualityControlCost int64 `json:"quality_control_cost"` // cents - Sum of all items quality-control costs
	FailureCost        int64 `json:"failure_cost"`         // cents - Sum of all items failure-rate costs
	OverheadCost       int64 `json:"overhead_cost"`        // cents - Overhead calculated on subtotal
	ProfitAmount       int64 `json:"profit_amount"`        // cents - Profit margin calculated

	// Discount/shipping/tax results (cents, except TaxRateApplied which is a percent).
	DiscountAmount int64   `json:"discount_amount"`  // cents
	ShippingCost   int64   `json:"shipping_cost"`    // cents
	TaxAmount      int64   `json:"tax_amount"`       // cents
	TaxRateApplied float64 `json:"tax_rate_applied"` // percent actually applied

	TotalCost int64 `json:"total_cost"` // cents - Final total (base - discount + shipping + tax)

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

// BasePrice returns the sale price before discount/shipping/tax adjustments,
// derived from the stored total: base = total + discount - shipping - tax. It is
// the figure the PDF/response show as the pre-adjustment subtotal.
func (b *BudgetEntity) BasePrice() int64 {
	return b.TotalCost + b.DiscountAmount - b.ShippingCost - b.TaxAmount
}

// IsExpired reports whether the budget's validity has passed as of now. It is a
// pure check on valid_until and does not consider the stored status, so public
// endpoints can report expiry without depending on the background expiry job.
func (b *BudgetEntity) IsExpired(now time.Time) bool {
	return b.ValidUntil != nil && now.After(*b.ValidUntil)
}

// EffectiveStatus returns the status as the customer should see it: a sent budget
// whose valid_until has passed reports as expired even if the background job has
// not run yet.
func (b *BudgetEntity) EffectiveStatus(now time.Time) BudgetStatus {
	if b.Status == StatusSent && b.IsExpired(now) {
		return StatusExpired
	}
	return b.Status
}

// CanRespond reports whether the customer may still approve/reject the budget:
// only a sent budget that has not expired is answerable.
func (b *BudgetEntity) CanRespond(now time.Time) bool {
	return b.Status == StatusSent && !b.IsExpired(now)
}

// validTransitions is the single source of truth for allowed status transitions.
var validTransitions = map[BudgetStatus][]BudgetStatus{
	StatusDraft:     {StatusSent, StatusCancelled},
	StatusSent:      {StatusApproved, StatusRejected, StatusExpired, StatusCancelled},
	StatusApproved:  {StatusPrinting, StatusCancelled},
	StatusRejected:  {StatusDraft},
	StatusExpired:   {StatusDraft},
	StatusCancelled: {StatusDraft},
	StatusPrinting:  {StatusCompleted},
	StatusCompleted: {},
}

// IsValidTransition checks if a status transition is valid
func (b *BudgetEntity) IsValidTransition(newStatus BudgetStatus) bool {
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

// ExpiredBudget identifies a budget moved to expired by the expiry job.
type ExpiredBudget struct {
	ID             uuid.UUID
	OrganizationID string
	Name           string
	QuoteNumber    *int
}
