package repositories_test

import (
	"testing"
	"time"

	"github.com/RodolfoBonis/spooliq/features/preset/data/models"
	repoimpl "github.com/RodolfoBonis/spooliq/features/preset/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/usecases"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedEnergy inserts an energy preset (base + child) for org via the repository.
func seedEnergy(t *testing.T, repo repositories.PresetRepository, org string, isDefault bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	preset := &entities.PresetEntity{
		ID:             id,
		Name:           "Tarifa",
		Type:           entities.PresetTypeEnergy,
		IsActive:       true,
		IsDefault:      isDefault,
		OrganizationID: org,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	energy := &entities.EnergyPresetEntity{
		ID:                    id,
		OrganizationID:        org,
		EnergyCostPerKwh:      0.85,
		Currency:              "BRL",
		PeakHourMultiplier:    1.0,
		OffPeakHourMultiplier: 1.0,
	}
	require.NoError(t, repo.CreateEnergy(preset, energy))
	return id
}

// insertRawDefault inserts a base preset row directly (bypassing the repository's
// default-clearing), simulating legacy data with multiple defaults per type.
func insertRawDefault(t *testing.T, db *gorm.DB, org string, presetType entities.PresetType, updatedAt time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	m := &models.PresetModel{
		ID:             id,
		Name:           "legacy",
		Type:           string(presetType),
		IsActive:       true,
		IsDefault:      true,
		OrganizationID: org,
		CreatedAt:      updatedAt,
		UpdatedAt:      updatedAt,
	}
	require.NoError(t, db.Create(m).Error)
	// The UpdatedAt column is write-protected on create by the model's GORM tag,
	// so set it explicitly to control the dedupe tiebreaker deterministically.
	require.NoError(t, db.Model(&models.PresetModel{}).Where("id = ?", id).UpdateColumn("updated_at", updatedAt).Error)
	return id
}

func isDefault(t *testing.T, repo repositories.PresetRepository, id uuid.UUID, org string) bool {
	t.Helper()
	p, err := repo.GetByID(id, org)
	require.NoError(t, err)
	return p.IsDefault
}

// MigrateDefaults must keep only the most recently updated default per
// (organization, type) and then create a working partial unique index. Running
// it twice must be a no-op (idempotent).
func TestIntegration_MigrateDefaults_DedupeKeepsMostRecent(t *testing.T) {
	repo, db := setupRepo(t)

	now := time.Now()
	older := insertRawDefault(t, db, itOrgA, entities.PresetTypeEnergy, now.Add(-time.Hour))
	newer := insertRawDefault(t, db, itOrgA, entities.PresetTypeEnergy, now)
	// A different type in the same org must be left alone.
	otherType := insertRawDefault(t, db, itOrgA, entities.PresetTypeCost, now.Add(-2*time.Hour))
	// A different org must be left alone.
	otherOrg := insertRawDefault(t, db, itOrgB, entities.PresetTypeEnergy, now.Add(-3*time.Hour))

	require.NoError(t, repoimpl.MigrateDefaults(db))
	// Idempotent: a second run must not error nor change the outcome.
	require.NoError(t, repoimpl.MigrateDefaults(db))

	assert.False(t, isDefault(t, repo, older, itOrgA), "older duplicate default should be cleared")
	assert.True(t, isDefault(t, repo, newer, itOrgA), "most recent default should be kept")
	assert.True(t, isDefault(t, repo, otherType, itOrgA), "other type default untouched")
	assert.True(t, isDefault(t, repo, otherOrg, itOrgB), "other org default untouched")

	// The partial unique index must now reject a second live default of the same
	// (org, type) inserted directly.
	err := db.Create(&models.PresetModel{
		ID: uuid.New(), Name: "dup", Type: string(entities.PresetTypeEnergy),
		IsActive: true, IsDefault: true, OrganizationID: itOrgA,
		CreatedAt: now, UpdatedAt: now,
	}).Error
	assert.Error(t, err, "partial unique index should reject a second live default")
}

// Creating a second default of the same type through the repository must clear
// the first, so the partial unique index is never violated.
func TestIntegration_CreateDefault_ClearsSibling(t *testing.T) {
	repo, db := setupRepo(t)
	require.NoError(t, repoimpl.MigrateDefaults(db))

	first := seedEnergy(t, repo, itOrgA, true)
	second := seedEnergy(t, repo, itOrgA, true)

	assert.False(t, isDefault(t, repo, first, itOrgA), "first default should be cleared")
	assert.True(t, isDefault(t, repo, second, itOrgA), "second should become the default")
}

// SetDefault must atomically clear the sibling default and set the target.
func TestIntegration_SetDefault_ClearsSibling(t *testing.T) {
	repo, db := setupRepo(t)
	require.NoError(t, repoimpl.MigrateDefaults(db))

	current := seedEnergy(t, repo, itOrgA, true)
	other := seedEnergy(t, repo, itOrgA, false)

	updated, err := repo.SetDefault(other, itOrgA)
	require.NoError(t, err)
	assert.True(t, updated.IsDefault)

	assert.True(t, isDefault(t, repo, other, itOrgA))
	assert.False(t, isDefault(t, repo, current, itOrgA))

	// Cross-org SetDefault must not find the preset.
	_, err = repo.SetDefault(other, itOrgB)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// Duplicate must copy the base and the typed child, reset is_default, and stay
// within the organization.
func TestIntegration_Duplicate_CopiesBaseAndChild(t *testing.T) {
	repo, _ := setupRepo(t)

	orig := seedMachine(t, repo, itOrgA, true, true)

	copyPreset, err := repo.Duplicate(orig, itOrgA, "Printer (cópia)")
	require.NoError(t, err)
	assert.NotEqual(t, orig, copyPreset.ID)
	assert.Equal(t, "Printer (cópia)", copyPreset.Name)
	assert.False(t, copyPreset.IsDefault, "a duplicate must never be a default")

	// The machine child must have been copied under the new ID.
	child, err := repo.GetMachineByID(copyPreset.ID, itOrgA)
	require.NoError(t, err)
	assert.Equal(t, "Prusa", child.Brand)

	// Cross-org duplicate must not find the source preset.
	_, err = repo.Duplicate(orig, itOrgB, "hack (cópia)")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// CreateFromTemplate (use case over the real repository) must persist a preset
// built from the static template, with the overridden default flag.
func TestIntegration_CreateFromTemplate_Persists(t *testing.T) {
	repo, _ := setupRepo(t)
	createUC := usecases.NewCreatePresetUseCase(repo)

	preset, err := createUC.CreateFromTemplate("custo-hobby", usecases.FromTemplateOverrides{IsDefault: true}, itOrgA, nil)
	require.NoError(t, err)
	assert.Equal(t, entities.PresetTypeCost, preset.Type)
	assert.True(t, preset.IsDefault)

	cost, err := repo.GetCostByID(preset.ID, itOrgA)
	require.NoError(t, err)
	assert.InDelta(t, 20, cost.LaborCostPerHour, 0.001)
	assert.InDelta(t, 15, cost.ProfitMarginPercentage, 0.001)

	// Unknown template key surfaces a typed error.
	_, err = createUC.CreateFromTemplate("does-not-exist", usecases.FromTemplateOverrides{}, itOrgA, nil)
	assert.ErrorIs(t, err, entities.ErrTemplateNotFound)
}
