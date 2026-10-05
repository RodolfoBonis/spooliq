package usecases

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/profile/domain/entities"
	"github.com/google/uuid"
)

// duplicateNameSuffix is appended to a profile's name when it is duplicated.
const duplicateNameSuffix = " (cópia)"

// Create validates the referenced presets, auto-generates the name when empty
// and persists a new profile. createdBy is the authenticated user's id.
func (uc *ProfileUseCase) Create(req *entities.CreateProfileRequest, organizationID, createdBy string) (*entities.ProfileResponse, error) {
	resolved, err := uc.validateRefs(req.MachinePresetID, req.EnergyPresetID, req.CostPresetID, organizationID)
	if err != nil {
		return nil, err
	}

	name := req.Name
	if name == "" {
		name = entities.GenerateProfileName(resolved.machine.Name, resolved.energy.Name)
	}

	now := time.Now()
	profile := &entities.ProfileEntity{
		ID:              uuid.New(),
		OrganizationID:  organizationID,
		Name:            name,
		Description:     req.Description,
		MachinePresetID: req.MachinePresetID,
		EnergyPresetID:  req.EnergyPresetID,
		CostPresetID:    req.CostPresetID,
		IsDefault:       req.IsDefault,
		CreatedBy:       createdBy,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := uc.profileRepo.Create(profile); err != nil {
		return nil, err
	}

	resp := uc.buildResponse(profile)
	return &resp, nil
}

// List returns all profiles for an organization as API responses.
func (uc *ProfileUseCase) List(organizationID string) ([]entities.ProfileResponse, error) {
	profiles, err := uc.profileRepo.List(organizationID)
	if err != nil {
		return nil, err
	}
	responses := make([]entities.ProfileResponse, 0, len(profiles))
	for _, p := range profiles {
		responses = append(responses, uc.buildResponse(p))
	}
	return responses, nil
}

// Get returns a single profile by id, scoped to the organization.
func (uc *ProfileUseCase) Get(id uuid.UUID, organizationID string) (*entities.ProfileResponse, error) {
	profile, err := uc.profileRepo.GetByID(id, organizationID)
	if err != nil {
		return nil, err
	}
	resp := uc.buildResponse(profile)
	return &resp, nil
}

// Update applies the provided fields to an existing profile, re-validating any
// changed preset references and re-generating the name only when the caller
// clears it while changing a referenced preset.
func (uc *ProfileUseCase) Update(req *entities.UpdateProfileRequest, organizationID string) (*entities.ProfileResponse, error) {
	profile, err := uc.profileRepo.GetByID(req.ID, organizationID)
	if err != nil {
		return nil, err
	}

	if req.MachinePresetID != nil {
		profile.MachinePresetID = *req.MachinePresetID
	}
	if req.EnergyPresetID != nil {
		profile.EnergyPresetID = *req.EnergyPresetID
	}
	if req.CostPresetID != nil {
		profile.CostPresetID = req.CostPresetID
	}

	// Validate the (possibly updated) references and load names for naming.
	resolved, err := uc.validateRefs(profile.MachinePresetID, profile.EnergyPresetID, profile.CostPresetID, organizationID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		profile.Name = req.Name
	} else if profile.Name == "" {
		profile.Name = entities.GenerateProfileName(resolved.machine.Name, resolved.energy.Name)
	}
	if req.Description != "" {
		profile.Description = req.Description
	}
	if req.IsDefault != nil {
		profile.IsDefault = *req.IsDefault
	}
	profile.UpdatedAt = time.Now()

	if err := uc.profileRepo.Update(profile); err != nil {
		return nil, err
	}

	resp := uc.buildResponse(profile)
	return &resp, nil
}

// Delete soft deletes a profile. The default profile cannot be deleted
// (ErrCannotDeleteDefaultProfile), mapped to HTTP 409 by the handler.
func (uc *ProfileUseCase) Delete(id uuid.UUID, organizationID string) error {
	profile, err := uc.profileRepo.GetByID(id, organizationID)
	if err != nil {
		return err
	}
	if profile.IsDefault {
		return entities.ErrCannotDeleteDefaultProfile
	}
	return uc.profileRepo.Delete(id, organizationID)
}

// SetDefault marks a profile as the single default for its organization.
func (uc *ProfileUseCase) SetDefault(id uuid.UUID, organizationID string) (*entities.ProfileResponse, error) {
	profile, err := uc.profileRepo.SetDefault(id, organizationID)
	if err != nil {
		return nil, err
	}
	resp := uc.buildResponse(profile)
	return &resp, nil
}

// Duplicate copies a profile within the organization, naming it "<name> (cópia)".
func (uc *ProfileUseCase) Duplicate(id uuid.UUID, organizationID string) (*entities.ProfileResponse, error) {
	profile, err := uc.profileRepo.GetByID(id, organizationID)
	if err != nil {
		return nil, err
	}
	duplicated, err := uc.profileRepo.Duplicate(id, organizationID, profile.Name+duplicateNameSuffix)
	if err != nil {
		return nil, err
	}
	resp := uc.buildResponse(duplicated)
	return &resp, nil
}
