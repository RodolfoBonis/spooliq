// Package mocks provides test doubles for the print profile feature.
package mocks

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/profile/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/profile/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InMemoryProfileRepository is a stateful, in-memory PrintProfileRepository for
// tests. It reproduces the tenant-isolation and single-default contract of the
// real repository.
type InMemoryProfileRepository struct {
	profiles map[uuid.UUID]*entities.ProfileEntity
}

// NewInMemoryProfileRepository creates an empty in-memory repository.
func NewInMemoryProfileRepository() *InMemoryProfileRepository {
	return &InMemoryProfileRepository{profiles: make(map[uuid.UUID]*entities.ProfileEntity)}
}

var _ repositories.PrintProfileRepository = (*InMemoryProfileRepository)(nil)

func (r *InMemoryProfileRepository) liveInOrg(id uuid.UUID, organizationID string) *entities.ProfileEntity {
	p, ok := r.profiles[id]
	if !ok || p.DeletedAt != nil || p.OrganizationID != organizationID {
		return nil
	}
	return p
}

func (r *InMemoryProfileRepository) clearOtherDefaults(organizationID string, exceptID uuid.UUID) {
	for id, p := range r.profiles {
		if id == exceptID || p.DeletedAt != nil {
			continue
		}
		if p.OrganizationID == organizationID {
			p.IsDefault = false
		}
	}
}

// Create stores a new profile, clearing a sibling default when needed.
func (r *InMemoryProfileRepository) Create(profile *entities.ProfileEntity) error {
	if profile.IsDefault {
		r.clearOtherDefaults(profile.OrganizationID, profile.ID)
	}
	clone := *profile
	r.profiles[profile.ID] = &clone
	return nil
}

// GetByID retrieves a profile scoped to the organization.
func (r *InMemoryProfileRepository) GetByID(id uuid.UUID, organizationID string) (*entities.ProfileEntity, error) {
	p := r.liveInOrg(id, organizationID)
	if p == nil {
		return nil, gorm.ErrRecordNotFound
	}
	clone := *p
	return &clone, nil
}

// List returns profiles for an organization.
func (r *InMemoryProfileRepository) List(organizationID string) ([]*entities.ProfileEntity, error) {
	var result []*entities.ProfileEntity
	for _, p := range r.profiles {
		if p.DeletedAt != nil || p.OrganizationID != organizationID {
			continue
		}
		clone := *p
		result = append(result, &clone)
	}
	return result, nil
}

// Update updates a profile scoped to its organization.
func (r *InMemoryProfileRepository) Update(profile *entities.ProfileEntity) error {
	if r.liveInOrg(profile.ID, profile.OrganizationID) == nil {
		return gorm.ErrRecordNotFound
	}
	if profile.IsDefault {
		r.clearOtherDefaults(profile.OrganizationID, profile.ID)
	}
	clone := *profile
	r.profiles[profile.ID] = &clone
	return nil
}

// Delete soft-deletes a profile scoped to its organization.
func (r *InMemoryProfileRepository) Delete(id uuid.UUID, organizationID string) error {
	p := r.liveInOrg(id, organizationID)
	if p == nil {
		return gorm.ErrRecordNotFound
	}
	now := time.Now()
	p.DeletedAt = &now
	return nil
}

// SetDefault marks the profile as the single default for its organization.
func (r *InMemoryProfileRepository) SetDefault(id uuid.UUID, organizationID string) (*entities.ProfileEntity, error) {
	p := r.liveInOrg(id, organizationID)
	if p == nil {
		return nil, gorm.ErrRecordNotFound
	}
	r.clearOtherDefaults(organizationID, id)
	p.IsDefault = true
	p.UpdatedAt = time.Now()
	clone := *p
	return &clone, nil
}

// Duplicate copies a profile within the organization.
func (r *InMemoryProfileRepository) Duplicate(id uuid.UUID, organizationID string, newName string) (*entities.ProfileEntity, error) {
	p := r.liveInOrg(id, organizationID)
	if p == nil {
		return nil, gorm.ErrRecordNotFound
	}
	now := time.Now()
	clone := *p
	clone.ID = uuid.New()
	clone.Name = newName
	clone.IsDefault = false
	clone.CreatedAt = now
	clone.UpdatedAt = now
	clone.DeletedAt = nil
	r.profiles[clone.ID] = &clone
	result := clone
	return &result, nil
}
