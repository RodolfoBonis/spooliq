package entities

import (
	"time"

	"github.com/google/uuid"
)

// Model3DEntity represents a 3D model in the domain layer.
type Model3DEntity struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID string     `json:"organization_id"`
	CustomerID     *uuid.UUID `json:"customer_id,omitempty"`
	Name           string     `json:"name"`
	Description    string     `json:"description,omitempty"`
	FileName       string     `json:"file_name"`
	FileURL        string     `json:"file_url"`
	FileFormat     string     `json:"file_format"`
	FileSizeBytes  int64      `json:"file_size_bytes"`
	FileHash       string     `json:"file_hash"`
	ThumbnailURL   *string    `json:"thumbnail_url,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	Tags           *string    `json:"tags,omitempty"`
	OwnerUserID    string     `json:"owner_user_id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}
