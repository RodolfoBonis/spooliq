package usecases_test

import (
	"testing"
	"time"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/preset/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	orgA = "org-a"
	orgB = "org-b"
)

// seedMachinePreset inserts a machine preset owned by organizationID and returns its ID.
func seedMachinePreset(repo *mocks.InMemoryPresetRepository, organizationID string, isDefault bool) uuid.UUID {
	id := uuid.New()
	preset := &entities.PresetEntity{
		ID:             id,
		Name:           "Printer",
		Type:           entities.PresetTypeMachine,
		IsActive:       true,
		IsDefault:      isDefault,
		OrganizationID: organizationID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	machine := &entities.MachinePresetEntity{
		ID:               id,
		OrganizationID:   organizationID,
		Brand:            "Prusa",
		BuildVolumeX:     200,
		BuildVolumeY:     200,
		BuildVolumeZ:     200,
		NozzleDiameter:   0.4,
		LayerHeightMin:   0.1,
		LayerHeightMax:   0.3,
		PrintSpeedMax:    120,
		PowerConsumption: 150,
		FilamentDiameter: 1.75,
	}
	repo.Seed(preset, machine)
	return id
}

// Org A must never be able to READ a preset that belongs to Org B.
func TestFindByID_CrossOrgReturnsNotFound(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	findUC := usecases.NewFindPresetUseCase(repo)

	presetID := seedMachinePreset(repo, orgB, false)

	// Org B (the owner) can read it.
	owned, err := findUC.FindByID(presetID, orgB)
	require.NoError(t, err)
	assert.Equal(t, presetID, owned.ID)

	// Org A cannot — and gets "not found", not a 403-style leak.
	got, err := findUC.FindByID(presetID, orgA)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestFindMachinePresetByID_CrossOrgReturnsNotFound(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	findUC := usecases.NewFindPresetUseCase(repo)

	presetID := seedMachinePreset(repo, orgB, false)

	got, err := findUC.FindMachinePresetByID(presetID, orgA)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// Org A must never see Org B's presets when listing.
func TestListPresets_IsScopedByOrganization(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	findUC := usecases.NewFindPresetUseCase(repo)

	seedMachinePreset(repo, orgA, false)
	seedMachinePreset(repo, orgB, false)
	seedMachinePreset(repo, orgB, false)

	aPresets, _, err := findUC.FindPresets(orgA, entities.PresetFilters{}, helpers.ListQuery{})
	require.NoError(t, err)
	assert.Len(t, aPresets, 1)
	for _, p := range aPresets {
		assert.Equal(t, orgA, p.OrganizationID)
	}

	bPresets, _, err := findUC.FindPresets(orgB, entities.PresetFilters{}, helpers.ListQuery{})
	require.NoError(t, err)
	assert.Len(t, bPresets, 2)
}

// Combinable filters must all apply within the same scoped query.
func TestListPresets_CombinableFilters(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	findUC := usecases.NewFindPresetUseCase(repo)

	userID := uuid.New()
	activeDefault := &entities.PresetEntity{
		ID: uuid.New(), Name: "A", Type: entities.PresetTypeMachine,
		IsActive: true, IsDefault: true, OrganizationID: orgA, UserID: &userID,
	}
	activeNonDefault := &entities.PresetEntity{
		ID: uuid.New(), Name: "B", Type: entities.PresetTypeMachine,
		IsActive: true, IsDefault: false, OrganizationID: orgA,
	}
	inactiveDefault := &entities.PresetEntity{
		ID: uuid.New(), Name: "C", Type: entities.PresetTypeEnergy,
		IsActive: false, IsDefault: true, OrganizationID: orgA,
	}
	repo.Seed(activeDefault, nil)
	repo.Seed(activeNonDefault, nil)
	repo.Seed(inactiveDefault, nil)

	machineType := entities.PresetTypeMachine
	// type=machine AND active AND default -> only activeDefault
	result, _, err := findUC.FindPresets(orgA, entities.PresetFilters{
		Type:        &machineType,
		ActiveOnly:  true,
		DefaultOnly: true,
	}, helpers.ListQuery{})
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, activeDefault.ID, result[0].ID)

	// global only -> excludes the user-owned activeDefault
	globalResult, _, err := findUC.FindPresets(orgA, entities.PresetFilters{GlobalOnly: true}, helpers.ListQuery{})
	require.NoError(t, err)
	for _, p := range globalResult {
		assert.Nil(t, p.UserID)
	}
	assert.Len(t, globalResult, 2)
}

// Org A must never be able to UPDATE a preset that belongs to Org B.
func TestUpdateMachinePreset_CrossOrgReturnsNotFound(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	updateUC := usecases.NewUpdatePresetUseCase(repo)

	presetID := seedMachinePreset(repo, orgB, false)
	newName := "Hacked"

	_, err := updateUC.UpdateMachinePreset(&usecases.UpdateMachinePresetRequest{
		ID:   presetID,
		Name: newName,
	}, orgA)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Org B's data must be untouched.
	stored := repo.StoredPreset(presetID)
	require.NotNil(t, stored)
	assert.Equal(t, "Printer", stored.Name)
}

// Org A must never be able to DELETE a preset that belongs to Org B.
func TestDeletePreset_CrossOrgReturnsNotFound(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	deleteUC := usecases.NewDeletePresetUseCase(repo)

	presetID := seedMachinePreset(repo, orgB, false)

	err := deleteUC.Execute(presetID, orgA)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Org B's preset must still be live (not soft-deleted).
	stored := repo.StoredPreset(presetID)
	require.NotNil(t, stored)
	assert.Nil(t, stored.DeletedAt)
}

// Deleting a default preset must be rejected with a dedicated error.
func TestDeletePreset_DefaultCannotBeDeleted(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	deleteUC := usecases.NewDeletePresetUseCase(repo)

	presetID := seedMachinePreset(repo, orgA, true) // default

	err := deleteUC.Execute(presetID, orgA)
	assert.ErrorIs(t, err, entities.ErrCannotDeleteDefaultPreset)

	stored := repo.StoredPreset(presetID)
	require.NotNil(t, stored)
	assert.Nil(t, stored.DeletedAt)
}

// A non-default preset in the caller's org is soft-deleted successfully.
func TestDeletePreset_OwnedSucceeds(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	deleteUC := usecases.NewDeletePresetUseCase(repo)

	presetID := seedMachinePreset(repo, orgA, false)

	require.NoError(t, deleteUC.Execute(presetID, orgA))

	stored := repo.StoredPreset(presetID)
	require.NotNil(t, stored)
	assert.NotNil(t, stored.DeletedAt, "preset should be soft-deleted")
}

// The create use case must stamp the authenticated user, ignoring the body value.
func TestCreateMachinePreset_UsesAuthenticatedUser(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	createUC := usecases.NewCreatePresetUseCase(repo)

	bodyUserID := uuid.New() // attacker-controlled, must be ignored
	authUserID := uuid.New() // from the authenticated context

	preset, err := createUC.CreateMachinePreset(&usecases.CreateMachinePresetRequest{
		Name:             "Printer",
		UserID:           &bodyUserID,
		BuildVolumeX:     200,
		BuildVolumeY:     200,
		BuildVolumeZ:     200,
		NozzleDiameter:   0.4,
		LayerHeightMin:   0.1,
		LayerHeightMax:   0.3,
		PrintSpeedMax:    120,
		PowerConsumption: 150,
		FilamentDiameter: 1.75,
	}, orgA, &authUserID)
	require.NoError(t, err)
	require.NotNil(t, preset.UserID)
	assert.Equal(t, authUserID, *preset.UserID)
	assert.NotEqual(t, bodyUserID, *preset.UserID)
	assert.Equal(t, orgA, preset.OrganizationID)
}

// Org scope must be threaded through to the repository on every read/update/delete.
func TestOrganizationScopeThreadedThrough(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	findUC := usecases.NewFindPresetUseCase(repo)

	presetID := seedMachinePreset(repo, orgA, false)
	_, _ = findUC.FindByID(presetID, orgA)

	require.NotEmpty(t, repo.OrgScopes)
	assert.Contains(t, repo.OrgScopes, orgA)
}
