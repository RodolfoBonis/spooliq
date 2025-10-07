package entities

// UpsertBrandRequestEntity represents the request payload for creating or updating a brand.
type UpsertBrandRequestEntity struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description,omitempty"`
}
