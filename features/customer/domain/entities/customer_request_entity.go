package entities

import "github.com/google/uuid"

// CreateCustomerRequest represents the request to create a new customer
type CreateCustomerRequest struct {
	Name     string  `json:"name" validate:"required,min=1,max=255"`
	Email    *string `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Phone    *string `json:"phone,omitempty" validate:"omitempty,max=50"`
	Document *string `json:"document,omitempty" validate:"omitempty,max=50"`
	Address  *string `json:"address,omitempty" validate:"omitempty,max=500"`
	City     *string `json:"city,omitempty" validate:"omitempty,max=255"`
	State    *string `json:"state,omitempty" validate:"omitempty,max=100"`
	ZipCode  *string `json:"zip_code,omitempty" validate:"omitempty,max=20"`
	Notes    *string `json:"notes,omitempty" validate:"omitempty,max=1000"`
}

// UpdateCustomerRequest represents the request to update an existing customer
type UpdateCustomerRequest struct {
	Name     *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Email    *string `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Phone    *string `json:"phone,omitempty" validate:"omitempty,max=50"`
	Document *string `json:"document,omitempty" validate:"omitempty,max=50"`
	Address  *string `json:"address,omitempty" validate:"omitempty,max=500"`
	City     *string `json:"city,omitempty" validate:"omitempty,max=255"`
	State    *string `json:"state,omitempty" validate:"omitempty,max=100"`
	ZipCode  *string `json:"zip_code,omitempty" validate:"omitempty,max=20"`
	Notes    *string `json:"notes,omitempty" validate:"omitempty,max=1000"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// SearchCustomerRequest represents the request to search customers
type SearchCustomerRequest struct {
	Name     string     `form:"name"`
	Email    string     `form:"email"`
	Phone    string     `form:"phone"`
	Document string     `form:"document"`
	City     string     `form:"city"`
	State    string     `form:"state"`
	IsActive *bool      `form:"is_active"`
	Page     int        `form:"page" validate:"omitempty,min=1"`
	PageSize int        `form:"page_size" validate:"omitempty,min=1,max=100"`
	SortBy   string     `form:"sort_by" validate:"omitempty,oneof=name email created_at"`
	SortDir  string     `form:"sort_dir" validate:"omitempty,oneof=asc desc"`
	IDFilter *uuid.UUID `form:"id"`
}
