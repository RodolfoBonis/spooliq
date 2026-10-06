package di

import (
	"context"
	"errors"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	presetEntities "github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	presetRepos "github.com/RodolfoBonis/spooliq/features/preset/domain/repositories"
	profileEntities "github.com/RodolfoBonis/spooliq/features/profile/domain/entities"
	profileRepos "github.com/RodolfoBonis/spooliq/features/profile/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// profilePresetProvider adapts the profile repository to the budget package's
// narrow usecases.ProfilePresetProvider. Living in the budget feature (not the
// profile feature), it keeps the dependency pointing budget -> profile and avoids
// an import cycle, since the resolver interface is owned by the budget package.
type profilePresetProvider struct {
	repo profileRepos.PrintProfileRepository
}

// NewProfilePresetProvider builds the profile-backed resolver provider.
func NewProfilePresetProvider(repo profileRepos.PrintProfileRepository) usecases.ProfilePresetProvider {
	return &profilePresetProvider{repo: repo}
}

func toProfilePresets(p *profileEntities.ProfileEntity) *usecases.ProfilePresets {
	machine := p.MachinePresetID
	energy := p.EnergyPresetID
	return &usecases.ProfilePresets{
		Name:            p.Name,
		MachinePresetID: &machine,
		EnergyPresetID:  &energy,
		CostPresetID:    p.CostPresetID,
	}
}

// ProfileByID resolves a profile within the organization. A not-found (another
// tenant or soft-deleted) maps to (nil, nil) so the resolver surfaces a 400.
func (a *profilePresetProvider) ProfileByID(_ context.Context, id uuid.UUID, organizationID string) (*usecases.ProfilePresets, error) {
	p, err := a.repo.GetByID(id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toProfilePresets(p), nil
}

// DefaultProfile returns the organization's default profile, or (nil, nil) when no
// default profile exists.
func (a *profilePresetProvider) DefaultProfile(_ context.Context, organizationID string) (*usecases.ProfilePresets, error) {
	profiles, err := a.repo.List(organizationID)
	if err != nil {
		return nil, err
	}
	for _, p := range profiles {
		if p.IsDefault {
			return toProfilePresets(p), nil
		}
	}
	return nil, nil
}

// defaultPresetProvider adapts the preset repository to the budget package's narrow
// usecases.DefaultPresetProvider.
type defaultPresetProvider struct {
	repo presetRepos.PresetRepository
}

// NewDefaultPresetProvider builds the preset-backed resolver provider.
func NewDefaultPresetProvider(repo presetRepos.PresetRepository) usecases.DefaultPresetProvider {
	return &defaultPresetProvider{repo: repo}
}

// DefaultPresetID returns the organization's default preset ID of the given type,
// or nil when the organization has no default preset of that type.
func (a *defaultPresetProvider) DefaultPresetID(_ context.Context, organizationID string, presetType string) (*uuid.UUID, error) {
	t := presetEntities.PresetType(presetType)
	presets, err := a.repo.ListPresets(organizationID, presetEntities.PresetFilters{
		Type:        &t,
		DefaultOnly: true,
		ActiveOnly:  true,
	})
	if err != nil {
		return nil, err
	}
	if len(presets) == 0 {
		return nil, nil
	}
	id := presets[0].ID
	return &id, nil
}
