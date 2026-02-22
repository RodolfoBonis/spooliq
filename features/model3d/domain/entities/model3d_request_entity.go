package entities

// CreateModel3DRequest represents the request payload for creating a 3D model via upload.
type CreateModel3DRequest struct {
	Name        string  `json:"name" form:"name" validate:"required,min=1,max=255"`
	Description string  `json:"description,omitempty" form:"description" validate:"omitempty,max=2000"`
	CustomerID  *string `json:"customer_id,omitempty" form:"customer_id" validate:"omitempty,uuid"`
	Tags        *string `json:"tags,omitempty" form:"tags" validate:"omitempty,max=1000"`
	Notes       *string `json:"notes,omitempty" form:"notes" validate:"omitempty,max=2000"`
}

// UpdateModel3DRequest represents the request payload for updating a 3D model's metadata.
type UpdateModel3DRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=2000"`
	CustomerID  *string `json:"customer_id,omitempty" validate:"omitempty,uuid"`
	Tags        *string `json:"tags,omitempty" validate:"omitempty,max=1000"`
	Notes       *string `json:"notes,omitempty" validate:"omitempty,max=2000"`
}
