package entities

// SupplyUnits is the set of valid measurement units a supply may use. It is the
// single source of truth behind the `oneof` validation tag below.
var SupplyUnits = []string{"un", "par", "ml", "g", "m", "cm", "kg", "l"}

// UpsertSupplyRequestEntity represents the request payload for creating or
// updating a supply.
//
// UnitCost is in CENTS and must be >= 0. Unit must be one of SupplyUnits.
// IsActive is a pointer so an absent value on update keeps the stored flag,
// while create defaults it to true when omitted (see the use case).
type UpsertSupplyRequestEntity struct {
	Name        string  `json:"name" validate:"required,min=1,max=255"`
	Unit        string  `json:"unit" validate:"required,oneof=un par ml g m cm kg l"`
	UnitCost    int64   `json:"unit_cost" validate:"gte=0"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000"`
	IsActive    *bool   `json:"is_active,omitempty"`
}
