package entities

import "encoding/json"

// UpsertMaterialRequestEntity represents the request payload for creating or
// updating a material.
//
// The temperature fields are accepted under BOTH the canonical snake_case keys
// (temp_table/temp_extruder) and the legacy camelCase keys (tempTable/
// tempExtruder) via a custom UnmarshalJSON. When a field is present under both
// spellings, the snake_case value wins. Validation ranges are preserved.
type UpsertMaterialRequestEntity struct {
	Name         string  `json:"name" validate:"required,min=1,max=255"`
	Description  string  `json:"description,omitempty"`
	TempTable    float32 `json:"temp_table" validate:"min=0,max=300"`
	TempExtruder float32 `json:"temp_extruder" validate:"min=0,max=500"`
}

// UnmarshalJSON decodes the request accepting both snake_case and legacy
// camelCase temperature keys. snake_case takes precedence when both appear.
func (r *UpsertMaterialRequestEntity) UnmarshalJSON(data []byte) error {
	var raw struct {
		Name               string   `json:"name"`
		Description        string   `json:"description"`
		TempTable          *float32 `json:"temp_table"`
		TempExtruder       *float32 `json:"temp_extruder"`
		TempTableLegacy    *float32 `json:"tempTable"`
		TempExtruderLegacy *float32 `json:"tempExtruder"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.Name = raw.Name
	r.Description = raw.Description

	switch {
	case raw.TempTable != nil:
		r.TempTable = *raw.TempTable
	case raw.TempTableLegacy != nil:
		r.TempTable = *raw.TempTableLegacy
	}

	switch {
	case raw.TempExtruder != nil:
		r.TempExtruder = *raw.TempExtruder
	case raw.TempExtruderLegacy != nil:
		r.TempExtruder = *raw.TempExtruderLegacy
	}

	return nil
}
