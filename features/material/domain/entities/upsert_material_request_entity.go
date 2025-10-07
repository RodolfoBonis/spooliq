package entities

// UpsertMaterialRequestEntity represents the request payload for creating or updating a material.
type UpsertMaterialRequestEntity struct {
	Name         string  `json:"name" validate:"required,min=1,max=255"`
	Description  string  `json:"description,omitempty"`
	TempTable    float32 `json:"tempTable,omitempty" validate:"min=0,max=300"`
	TempExtruder float32 `json:"tempExtruder,omitempty" validate:"min=0,max=500"`
}
