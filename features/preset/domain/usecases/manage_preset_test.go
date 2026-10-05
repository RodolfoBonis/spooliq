package usecases_test

import (
	"testing"

	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/preset/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSuggestName_PerType(t *testing.T) {
	uc := usecases.NewManagePresetUseCase(mocks.NewInMemoryPresetRepository())

	machine, err := uc.SuggestName(&usecases.SuggestNameRequest{Type: entities.PresetTypeMachine, Brand: "Bambu Lab", Model: "P1S", NozzleDiameter: 0.4})
	require.NoError(t, err)
	assert.Equal(t, "Bambu Lab P1S · bico 0.4mm", machine)

	energy, err := uc.SuggestName(&usecases.SuggestNameRequest{Type: entities.PresetTypeEnergy, Provider: "CEMIG", State: "MG", EnergyCostPerKwh: 0.95})
	require.NoError(t, err)
	assert.Equal(t, "CEMIG/MG · R$ 0,95/kWh", energy)

	cost, err := uc.SuggestName(&usecases.SuggestNameRequest{Type: entities.PresetTypeCost, LaborCostPerHour: 50, ProfitMarginPercentage: 20})
	require.NoError(t, err)
	assert.Equal(t, "Mão de obra R$ 50/h · margem 20%", cost)

	_, err = uc.SuggestName(&usecases.SuggestNameRequest{Type: "bogus"})
	assert.ErrorIs(t, err, entities.ErrInvalidPresetType)
}

func TestCreateFromTemplate_UnknownKey(t *testing.T) {
	createUC := usecases.NewCreatePresetUseCase(mocks.NewInMemoryPresetRepository())
	_, err := createUC.CreateFromTemplate("nope", usecases.FromTemplateOverrides{}, "org-a", nil)
	assert.ErrorIs(t, err, entities.ErrTemplateNotFound)
}

func TestCreateFromTemplate_MachineDefaults(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	createUC := usecases.NewCreatePresetUseCase(repo)

	preset, err := createUC.CreateFromTemplate("bambu-p1s", usecases.FromTemplateOverrides{IsDefault: true}, "org-a", nil)
	require.NoError(t, err)
	assert.Equal(t, entities.PresetTypeMachine, preset.Type)
	assert.Equal(t, "Bambu Lab P1S", preset.Name)
	assert.True(t, preset.IsDefault)
}

func TestManageSetDefaultAndDuplicate(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	createUC := usecases.NewCreatePresetUseCase(repo)
	manageUC := usecases.NewManagePresetUseCase(repo)

	a, err := createUC.CreateFromTemplate("custo-hobby", usecases.FromTemplateOverrides{IsDefault: true}, "org-a", nil)
	require.NoError(t, err)
	b, err := createUC.CreateFromTemplate("custo-profissional", usecases.FromTemplateOverrides{}, "org-a", nil)
	require.NoError(t, err)

	// SetDefault on b clears a (both are cost presets).
	updated, err := manageUC.SetDefault(b.ID, "org-a")
	require.NoError(t, err)
	assert.True(t, updated.IsDefault)
	gotA := repo.StoredPreset(a.ID)
	assert.False(t, gotA.IsDefault)

	// Duplicate names the copy and clears default.
	dup, err := manageUC.Duplicate(b.ID, "org-a")
	require.NoError(t, err)
	assert.Equal(t, b.Name+" (cópia)", dup.Name)
	assert.False(t, dup.IsDefault)

	// Cross-org duplicate is not found.
	_, err = manageUC.Duplicate(b.ID, "org-b")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Duplicate of a non-existent preset returns not found.
	_, err = manageUC.Duplicate(uuid.New(), "org-a")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestDelete_BlockedWhenReferencedByProfile(t *testing.T) {
	repo := mocks.NewInMemoryPresetRepository()
	repo.ProfileReferenced = map[uuid.UUID]bool{}
	createUC := usecases.NewCreatePresetUseCase(repo)
	deleteUC := usecases.NewDeletePresetUseCase(repo)

	preset, err := createUC.CreateFromTemplate("bambu-p1s", usecases.FromTemplateOverrides{}, "org-a", nil)
	require.NoError(t, err)

	// Not referenced yet: delete succeeds.
	other, err := createUC.CreateFromTemplate("bambu-a1", usecases.FromTemplateOverrides{}, "org-a", nil)
	require.NoError(t, err)
	require.NoError(t, deleteUC.Execute(other.ID, "org-a"))

	// Referenced by a live profile: delete is blocked with the domain error.
	repo.ProfileReferenced[preset.ID] = true
	err = deleteUC.Execute(preset.ID, "org-a")
	assert.ErrorIs(t, err, entities.ErrPresetInUseByProfile)
}
