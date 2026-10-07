package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	companyModels "github.com/RodolfoBonis/spooliq/features/company/data/models"
	customerModels "github.com/RodolfoBonis/spooliq/features/customer/data/models"
	presetModels "github.com/RodolfoBonis/spooliq/features/preset/data/models"
	profileModels "github.com/RodolfoBonis/spooliq/features/profile/data/models"
	userModels "github.com/RodolfoBonis/spooliq/features/users/data/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BudgetModel represents the budget data model for GORM
type BudgetModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID string    `gorm:"type:varchar(255);not null;index:idx_budget_org" json:"organization_id"` // FK: references companies(organization_id) ON DELETE RESTRICT
	Name           string    `gorm:"type:varchar(255);not null" json:"name"`
	Description    string    `gorm:"type:text" json:"description"`

	// QuoteNumber is the sequential per-organization quote number. Nullable at the
	// column level: it is backfilled for legacy rows and assigned on create/duplicate.
	// A partial unique index (organization_id, quote_number) is added in RunMigrations.
	QuoteNumber *int `gorm:"type:integer;index:idx_budget_quote_number" json:"quote_number"`

	// ValidUntil is the quote validity instant (timestamptz, end-of-day Sao_Paulo).
	ValidUntil *time.Time `gorm:"type:timestamptz" json:"valid_until"`

	// Public share token. No GORM default tag: the token is written explicitly by the
	// share use case. A partial unique index on public_token is added in RunMigrations.
	PublicToken          *string    `gorm:"type:varchar(43)" json:"public_token"`
	PublicTokenCreatedAt *time.Time `gorm:"type:timestamptz" json:"public_token_created_at"`

	// Customer response (recorded via the public approve/reject endpoints).
	CustomerResponseAt        *time.Time `gorm:"type:timestamptz" json:"customer_response_at"`
	CustomerResponseName      *string    `gorm:"type:varchar(120)" json:"customer_response_name"`
	CustomerResponseIP        *string    `gorm:"type:varchar(45)" json:"customer_response_ip"`
	CustomerResponseUserAgent *string    `gorm:"type:varchar(255)" json:"customer_response_user_agent"`
	RejectionReason           *string    `gorm:"type:text" json:"rejection_reason"`

	// Foreign key to Customer
	CustomerID uuid.UUID `gorm:"type:uuid;not null;index" json:"customer_id"`

	// Status
	Status string `gorm:"type:varchar(20);not null" json:"status"`

	// DEPRECATED: Print time moved to budget_items (per-item, not global)
	// Will be calculated as sum of all items print times
	PrintTimeHours   int `gorm:"type:integer;not null" json:"print_time_hours"`
	PrintTimeMinutes int `gorm:"type:integer;not null" json:"print_time_minutes"`

	// Presets (global - apply to all items unless overridden at item level)
	MachinePresetID *uuid.UUID `gorm:"type:uuid" json:"machine_preset_id"`
	EnergyPresetID  *uuid.UUID `gorm:"type:uuid" json:"energy_preset_id"`
	CostPresetID    *uuid.UUID `gorm:"type:uuid" json:"cost_preset_id"` // For overhead/profit percentages

	// Print profile the presets were resolved from. FK added idempotently in
	// RunMigrations (ON DELETE SET NULL) so deleting a profile never blocks/deletes budgets.
	ProfileID *uuid.UUID `gorm:"type:uuid;index" json:"profile_id"`

	// Configuration flags
	IncludeEnergyCost bool `gorm:"default:false" json:"include_energy_cost"`
	IncludeWasteCost  bool `gorm:"default:false" json:"include_waste_cost"`
	// IncludeMachineCost defaults TRUE for new budgets at the use-case level
	// (resolveIncludeMachineCost). The DB default must stay FALSE: GORM omits zero-valued
	// fields that carry a default tag from INSERT, so default:true would silently turn an
	// explicit false into true. Existing rows keep FALSE so stored budgets are not reinterpreted.
	IncludeMachineCost bool `gorm:"default:false" json:"include_machine_cost"`

	// Discount configuration (nullable).
	DiscountType  *string  `gorm:"type:varchar(10)" json:"discount_type"`
	DiscountValue *float64 `gorm:"type:double precision" json:"discount_value"`

	// Shipping configuration.
	IncludeShipping  bool   `gorm:"default:false" json:"include_shipping"`
	ShippingOverride *int64 `gorm:"type:bigint" json:"shipping_override"` // cents

	// TaxRate is the budget-level "por dentro" tax rate (percent); nil => company default.
	TaxRate *float64 `gorm:"type:double precision" json:"tax_rate"`

	// Calculated costs (in cents)
	FilamentCost       int64 `gorm:"type:bigint;default:0" json:"filament_cost"`
	WasteCost          int64 `gorm:"type:bigint;default:0" json:"waste_cost"`
	EnergyCost         int64 `gorm:"type:bigint;default:0" json:"energy_cost"`
	MachineCost        int64 `gorm:"type:bigint;default:0" json:"machine_cost"`
	SetupCost          int64 `gorm:"type:bigint;default:0" json:"setup_cost"`
	LaborCost          int64 `gorm:"type:bigint;default:0" json:"labor_cost"`
	PostProcessingCost int64 `gorm:"type:bigint;default:0" json:"post_processing_cost"`
	PackagingCost      int64 `gorm:"type:bigint;default:0" json:"packaging_cost"`
	QualityControlCost int64 `gorm:"type:bigint;default:0" json:"quality_control_cost"`
	FailureCost        int64 `gorm:"type:bigint;default:0" json:"failure_cost"`
	OverheadCost       int64 `gorm:"type:bigint;default:0" json:"overhead_cost"`
	ProfitAmount       int64 `gorm:"type:bigint;default:0" json:"profit_amount"`

	// Discount/shipping/tax results.
	DiscountAmount int64   `gorm:"type:bigint;default:0" json:"discount_amount"`
	ShippingCost   int64   `gorm:"type:bigint;default:0" json:"shipping_cost"`
	TaxAmount      int64   `gorm:"type:bigint;default:0" json:"tax_amount"`
	TaxRateApplied float64 `gorm:"type:double precision;default:0" json:"tax_rate_applied"`

	TotalCost int64 `gorm:"type:bigint;default:0" json:"total_cost"` // Final total

	// Additional fields for PDF generation
	DeliveryDays *int    `gorm:"type:integer" json:"delivery_days"`
	PaymentTerms *string `gorm:"type:text" json:"payment_terms"`
	Notes        *string `gorm:"type:text" json:"notes"`
	PDFUrl       *string `gorm:"type:varchar(500)" json:"pdf_url"`

	// Ownership
	OwnerUserID string `gorm:"type:varchar(255);not null;index" json:"owner_user_id"`

	// Timestamps
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	// GORM v2 Relationships
	Organization  *companyModels.CompanyModel      `gorm:"foreignKey:OrganizationID;references:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"organization,omitempty"`
	Customer      *customerModels.CustomerModel    `gorm:"foreignKey:CustomerID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"customer,omitempty"`
	User          *userModels.UserModel            `gorm:"foreignKey:OwnerUserID;references:KeycloakUserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"user,omitempty"`
	MachinePreset *presetModels.MachinePresetModel `gorm:"foreignKey:MachinePresetID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"machine_preset,omitempty"`
	EnergyPreset  *presetModels.EnergyPresetModel  `gorm:"foreignKey:EnergyPresetID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"energy_preset,omitempty"`
	CostPreset    *presetModels.CostPresetModel    `gorm:"foreignKey:CostPresetID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"cost_preset,omitempty"`
	Profile       *profileModels.PrintProfileModel `gorm:"foreignKey:ProfileID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"profile,omitempty"`
	Items         []BudgetItemModel                `gorm:"foreignKey:BudgetID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"items,omitempty"`
	StatusHistory []BudgetStatusHistoryModel       `gorm:"foreignKey:BudgetID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"status_history,omitempty"`
}

// TableName specifies the table name for GORM
func (BudgetModel) TableName() string {
	return "budgets"
}

// BeforeCreate is a GORM hook executed before creating a budget
func (b *BudgetModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	if b.Status == "" {
		b.Status = string(entities.StatusDraft)
	}
	return nil
}

// ToEntity converts the GORM model to domain entity
func (b *BudgetModel) ToEntity() *entities.BudgetEntity {
	return &entities.BudgetEntity{
		ID:                        b.ID,
		OrganizationID:            b.OrganizationID,
		Name:                      b.Name,
		Description:               b.Description,
		CustomerID:                b.CustomerID,
		Status:                    entities.BudgetStatus(b.Status),
		QuoteNumber:               b.QuoteNumber,
		ValidUntil:                b.ValidUntil,
		PublicToken:               b.PublicToken,
		PublicTokenCreatedAt:      b.PublicTokenCreatedAt,
		CustomerResponseAt:        b.CustomerResponseAt,
		CustomerResponseName:      b.CustomerResponseName,
		CustomerResponseIP:        b.CustomerResponseIP,
		CustomerResponseUserAgent: b.CustomerResponseUserAgent,
		RejectionReason:           b.RejectionReason,
		PrintTimeHours:            b.PrintTimeHours,
		PrintTimeMinutes:          b.PrintTimeMinutes,
		ProfileID:                 b.ProfileID,
		MachinePresetID:           b.MachinePresetID,
		EnergyPresetID:            b.EnergyPresetID,
		CostPresetID:              b.CostPresetID,
		IncludeEnergyCost:         b.IncludeEnergyCost,
		IncludeWasteCost:          b.IncludeWasteCost,
		IncludeMachineCost:        b.IncludeMachineCost,
		DiscountType:              b.DiscountType,
		DiscountValue:             b.DiscountValue,
		IncludeShipping:           b.IncludeShipping,
		ShippingOverride:          b.ShippingOverride,
		TaxRate:                   b.TaxRate,
		FilamentCost:              b.FilamentCost,
		WasteCost:                 b.WasteCost,
		EnergyCost:                b.EnergyCost,
		MachineCost:               b.MachineCost,
		SetupCost:                 b.SetupCost,
		LaborCost:                 b.LaborCost,
		PostProcessingCost:        b.PostProcessingCost,
		PackagingCost:             b.PackagingCost,
		QualityControlCost:        b.QualityControlCost,
		FailureCost:               b.FailureCost,
		OverheadCost:              b.OverheadCost,
		ProfitAmount:              b.ProfitAmount,
		DiscountAmount:            b.DiscountAmount,
		ShippingCost:              b.ShippingCost,
		TaxAmount:                 b.TaxAmount,
		TaxRateApplied:            b.TaxRateApplied,
		TotalCost:                 b.TotalCost,
		DeliveryDays:              b.DeliveryDays,
		PaymentTerms:              b.PaymentTerms,
		Notes:                     b.Notes,
		PDFUrl:                    b.PDFUrl,
		OwnerUserID:               b.OwnerUserID,
		CreatedAt:                 b.CreatedAt,
		UpdatedAt:                 b.UpdatedAt,
		DeletedAt:                 getDeletedAt(b.DeletedAt),
	}
}

// getDeletedAt returns nil if deleted_at is not valid, otherwise returns pointer to time
func getDeletedAt(deletedAt gorm.DeletedAt) *time.Time {
	if deletedAt.Valid {
		return &deletedAt.Time
	}
	return nil
}

// FromEntity converts domain entity to GORM model
func (b *BudgetModel) FromEntity(entity *entities.BudgetEntity) {
	b.ID = entity.ID
	b.OrganizationID = entity.OrganizationID
	b.Name = entity.Name
	b.Description = entity.Description
	b.CustomerID = entity.CustomerID
	b.Status = string(entity.Status)
	b.QuoteNumber = entity.QuoteNumber
	b.ValidUntil = entity.ValidUntil
	b.PublicToken = entity.PublicToken
	b.PublicTokenCreatedAt = entity.PublicTokenCreatedAt
	b.CustomerResponseAt = entity.CustomerResponseAt
	b.CustomerResponseName = entity.CustomerResponseName
	b.CustomerResponseIP = entity.CustomerResponseIP
	b.CustomerResponseUserAgent = entity.CustomerResponseUserAgent
	b.RejectionReason = entity.RejectionReason
	b.PrintTimeHours = entity.PrintTimeHours
	b.PrintTimeMinutes = entity.PrintTimeMinutes
	b.ProfileID = entity.ProfileID
	b.MachinePresetID = entity.MachinePresetID
	b.EnergyPresetID = entity.EnergyPresetID
	b.CostPresetID = entity.CostPresetID
	b.IncludeEnergyCost = entity.IncludeEnergyCost
	b.IncludeWasteCost = entity.IncludeWasteCost
	b.IncludeMachineCost = entity.IncludeMachineCost
	b.DiscountType = entity.DiscountType
	b.DiscountValue = entity.DiscountValue
	b.IncludeShipping = entity.IncludeShipping
	b.ShippingOverride = entity.ShippingOverride
	b.TaxRate = entity.TaxRate
	b.FilamentCost = entity.FilamentCost
	b.WasteCost = entity.WasteCost
	b.EnergyCost = entity.EnergyCost
	b.MachineCost = entity.MachineCost
	b.SetupCost = entity.SetupCost
	b.LaborCost = entity.LaborCost
	b.PostProcessingCost = entity.PostProcessingCost
	b.PackagingCost = entity.PackagingCost
	b.QualityControlCost = entity.QualityControlCost
	b.FailureCost = entity.FailureCost
	b.OverheadCost = entity.OverheadCost
	b.ProfitAmount = entity.ProfitAmount
	b.DiscountAmount = entity.DiscountAmount
	b.ShippingCost = entity.ShippingCost
	b.TaxAmount = entity.TaxAmount
	b.TaxRateApplied = entity.TaxRateApplied
	b.TotalCost = entity.TotalCost
	b.DeliveryDays = entity.DeliveryDays
	b.PaymentTerms = entity.PaymentTerms
	b.Notes = entity.Notes
	b.PDFUrl = entity.PDFUrl
	b.OwnerUserID = entity.OwnerUserID
	b.CreatedAt = entity.CreatedAt
	b.UpdatedAt = entity.UpdatedAt
	if entity.DeletedAt != nil {
		b.DeletedAt = gorm.DeletedAt{Time: *entity.DeletedAt, Valid: true}
	}
}
