package usecases_test

import (
	"testing"

	presetEntities "github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	presetMocks "github.com/RodolfoBonis/spooliq/features/preset/mocks"
	"github.com/RodolfoBonis/spooliq/features/profile/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/profile/domain/usecases"
	profileMocks "github.com/RodolfoBonis/spooliq/features/profile/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	orgA = "org-a"
	orgB = "org-b"
)

// seedPreset inserts a base preset of the given type/name into the preset mock.
func seedPreset(repo *presetMocks.InMemoryPresetRepository, org, name string, t presetEntities.PresetType) uuid.UUID {
	id := uuid.New()
	repo.Seed(&presetEntities.PresetEntity{
		ID:             id,
		Name:           name,
		Type:           t,
		IsActive:       true,
		OrganizationID: org,
	}, nil)
	return id
}

func newUC() (*usecases.ProfileUseCase, *presetMocks.InMemoryPresetRepository, *profileMocks.InMemoryProfileRepository) {
	presetRepo := presetMocks.NewInMemoryPresetRepository()
	profileRepo := profileMocks.NewInMemoryProfileRepository()
	return usecases.NewProfileUseCase(profileRepo, presetRepo), presetRepo, profileRepo
}

func TestCreate_AutoNameFromPresets(t *testing.T) {
	uc, presetRepo, _ := newUC()
	machine := seedPreset(presetRepo, orgA, "Bambu P1S", presetEntities.PresetTypeMachine)
	energy := seedPreset(presetRepo, orgA, "CEMIG/MG", presetEntities.PresetTypeEnergy)

	resp, err := uc.Create(&entities.CreateProfileRequest{
		MachinePresetID: machine,
		EnergyPresetID:  energy,
	}, orgA, "user-1")
	require.NoError(t, err)
	assert.Equal(t, "Bambu P1S · CEMIG/MG", resp.Name)
	assert.Equal(t, "Bambu P1S", resp.MachinePreset.Name)
	assert.Equal(t, "CEMIG/MG", resp.EnergyPreset.Name)
	assert.Nil(t, resp.CostPreset)
	assert.Equal(t, "user-1", resp.CreatedBy)
}

func TestCreate_RejectsWrongTypeMachine(t *testing.T) {
	uc, presetRepo, _ := newUC()
	// An energy preset passed as the machine reference must be rejected.
	notMachine := seedPreset(presetRepo, orgA, "Energy", presetEntities.PresetTypeEnergy)
	energy := seedPreset(presetRepo, orgA, "Energy2", presetEntities.PresetTypeEnergy)

	_, err := uc.Create(&entities.CreateProfileRequest{
		MachinePresetID: notMachine,
		EnergyPresetID:  energy,
	}, orgA, "user-1")
	assert.ErrorIs(t, err, entities.ErrInvalidMachinePreset)
}

func TestCreate_RejectsCrossOrgPreset(t *testing.T) {
	uc, presetRepo, _ := newUC()
	// Presets belong to orgB; orgA must not be able to reference them.
	machine := seedPreset(presetRepo, orgB, "M", presetEntities.PresetTypeMachine)
	energy := seedPreset(presetRepo, orgB, "E", presetEntities.PresetTypeEnergy)

	_, err := uc.Create(&entities.CreateProfileRequest{
		MachinePresetID: machine,
		EnergyPresetID:  energy,
	}, orgA, "user-1")
	assert.ErrorIs(t, err, entities.ErrInvalidMachinePreset)
}

func TestCreate_RejectsWrongTypeCost(t *testing.T) {
	uc, presetRepo, _ := newUC()
	machine := seedPreset(presetRepo, orgA, "M", presetEntities.PresetTypeMachine)
	energy := seedPreset(presetRepo, orgA, "E", presetEntities.PresetTypeEnergy)
	notCost := seedPreset(presetRepo, orgA, "M2", presetEntities.PresetTypeMachine)

	_, err := uc.Create(&entities.CreateProfileRequest{
		MachinePresetID: machine,
		EnergyPresetID:  energy,
		CostPresetID:    &notCost,
	}, orgA, "user-1")
	assert.ErrorIs(t, err, entities.ErrInvalidCostPreset)
}

func TestCreate_SecondDefaultClearsFirst(t *testing.T) {
	uc, presetRepo, profileRepo := newUC()
	machine := seedPreset(presetRepo, orgA, "M", presetEntities.PresetTypeMachine)
	energy := seedPreset(presetRepo, orgA, "E", presetEntities.PresetTypeEnergy)

	first, err := uc.Create(&entities.CreateProfileRequest{MachinePresetID: machine, EnergyPresetID: energy, IsDefault: true}, orgA, "u")
	require.NoError(t, err)
	second, err := uc.Create(&entities.CreateProfileRequest{MachinePresetID: machine, EnergyPresetID: energy, IsDefault: true}, orgA, "u")
	require.NoError(t, err)

	firstID := uuid.MustParse(first.ID)
	secondID := uuid.MustParse(second.ID)
	gotFirst, _ := profileRepo.GetByID(firstID, orgA)
	gotSecond, _ := profileRepo.GetByID(secondID, orgA)
	assert.False(t, gotFirst.IsDefault, "first default should be cleared")
	assert.True(t, gotSecond.IsDefault)
}

func TestDelete_DefaultRejected(t *testing.T) {
	uc, presetRepo, _ := newUC()
	machine := seedPreset(presetRepo, orgA, "M", presetEntities.PresetTypeMachine)
	energy := seedPreset(presetRepo, orgA, "E", presetEntities.PresetTypeEnergy)
	created, err := uc.Create(&entities.CreateProfileRequest{MachinePresetID: machine, EnergyPresetID: energy, IsDefault: true}, orgA, "u")
	require.NoError(t, err)

	err = uc.Delete(uuid.MustParse(created.ID), orgA)
	assert.ErrorIs(t, err, entities.ErrCannotDeleteDefaultProfile)
}

func TestDelete_NonDefaultSucceeds_CrossOrgNotFound(t *testing.T) {
	uc, presetRepo, _ := newUC()
	machine := seedPreset(presetRepo, orgA, "M", presetEntities.PresetTypeMachine)
	energy := seedPreset(presetRepo, orgA, "E", presetEntities.PresetTypeEnergy)
	created, err := uc.Create(&entities.CreateProfileRequest{MachinePresetID: machine, EnergyPresetID: energy}, orgA, "u")
	require.NoError(t, err)

	// Cross-org delete must not find it.
	assert.ErrorIs(t, uc.Delete(uuid.MustParse(created.ID), orgB), gorm.ErrRecordNotFound)
	// Owner delete succeeds.
	require.NoError(t, uc.Delete(uuid.MustParse(created.ID), orgA))
}

func TestSetDefault_ClearsSibling(t *testing.T) {
	uc, presetRepo, _ := newUC()
	machine := seedPreset(presetRepo, orgA, "M", presetEntities.PresetTypeMachine)
	energy := seedPreset(presetRepo, orgA, "E", presetEntities.PresetTypeEnergy)
	a, _ := uc.Create(&entities.CreateProfileRequest{MachinePresetID: machine, EnergyPresetID: energy, IsDefault: true}, orgA, "u")
	b, _ := uc.Create(&entities.CreateProfileRequest{MachinePresetID: machine, EnergyPresetID: energy}, orgA, "u")

	resp, err := uc.SetDefault(uuid.MustParse(b.ID), orgA)
	require.NoError(t, err)
	assert.True(t, resp.IsDefault)

	gotA, _ := uc.Get(uuid.MustParse(a.ID), orgA)
	assert.False(t, gotA.IsDefault)
}

func TestDuplicate_NamesCopyAndNotDefault(t *testing.T) {
	uc, presetRepo, _ := newUC()
	machine := seedPreset(presetRepo, orgA, "M", presetEntities.PresetTypeMachine)
	energy := seedPreset(presetRepo, orgA, "E", presetEntities.PresetTypeEnergy)
	created, err := uc.Create(&entities.CreateProfileRequest{Name: "Meu Perfil", MachinePresetID: machine, EnergyPresetID: energy, IsDefault: true}, orgA, "u")
	require.NoError(t, err)

	dup, err := uc.Duplicate(uuid.MustParse(created.ID), orgA)
	require.NoError(t, err)
	assert.Equal(t, "Meu Perfil (cópia)", dup.Name)
	assert.False(t, dup.IsDefault)
}
