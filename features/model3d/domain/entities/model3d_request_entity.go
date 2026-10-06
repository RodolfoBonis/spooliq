package entities

import "encoding/json"

// CreateModel3DRequest represents the request payload for creating a 3D model via
// a multipart upload. The scalar metadata is bound from the multipart form (see
// the `form` tags) and validated with the shared validator; the file itself is
// read separately from the multipart file part.
type CreateModel3DRequest struct {
	Name        string  `json:"name" form:"name" validate:"required,min=1,max=255"`
	Description string  `json:"description,omitempty" form:"description" validate:"omitempty,max=2000"`
	CustomerID  *string `json:"customer_id,omitempty" form:"customer_id" validate:"omitempty"`
	Tags        *string `json:"tags,omitempty" form:"tags" validate:"omitempty,max=1000"`
	Notes       *string `json:"notes,omitempty" form:"notes" validate:"omitempty,max=2000"`
}

// NullableString distinguishes three states in a JSON PATCH-style body:
//   - absent  (Set == false): the client did not mention the field; keep current.
//   - null    (Set == true, Value == nil): the client explicitly cleared it.
//   - value   (Set == true, Value != nil): the client set a new value.
//
// It is used so an Update can tell "leave customer_id unchanged" apart from
// "detach the customer" (explicit JSON null), which a plain *string cannot.
type NullableString struct {
	Value *string
	Set   bool
}

// UnmarshalJSON records that the key was present and decodes null vs a string.
func (n *NullableString) UnmarshalJSON(data []byte) error {
	n.Set = true
	if string(data) == "null" {
		n.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	n.Value = &s
	return nil
}

// UpdateModel3DRequest represents the request payload for updating a 3D model's
// metadata. CustomerID uses NullableString so an explicit JSON null detaches the
// customer, while omitting the field leaves it unchanged.
type UpdateModel3DRequest struct {
	Name        *string        `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string        `json:"description,omitempty" validate:"omitempty,max=2000"`
	CustomerID  NullableString `json:"customer_id"`
	Tags        *string        `json:"tags,omitempty" validate:"omitempty,max=1000"`
	Notes       *string        `json:"notes,omitempty" validate:"omitempty,max=2000"`
}
