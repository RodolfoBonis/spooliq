package repositories_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	presetModels "github.com/RodolfoBonis/spooliq/features/preset/data/models"
	presetRepoImpl "github.com/RodolfoBonis/spooliq/features/preset/data/repositories"
	presetEntities "github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	presetRepos "github.com/RodolfoBonis/spooliq/features/preset/domain/repositories"
	profileModels "github.com/RodolfoBonis/spooliq/features/profile/data/models"
	profileRepoImpl "github.com/RodolfoBonis/spooliq/features/profile/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/profile/domain/entities"
	profileRepos "github.com/RodolfoBonis/spooliq/features/profile/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/profile/domain/usecases"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// These tests exercise the REAL GORM profile repository (and the ProfileUseCase
// over the real preset repository) against a real PostgreSQL. They are gated by
// TEST_DATABASE_URL and skip when it is unset. Each run uses an isolated schema.

const (
	orgA = "org-a"
	orgB = "org-b"
)

func setup(t *testing.T) (*usecases.ProfileUseCase, profileRepos.PrintProfileRepository, presetRepos.PresetRepository, *gorm.DB) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping profile repository integration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	schema := fmt.Sprintf("profile_it_%s", uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)
	t.Cleanup(func() {
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})

	require.NoError(t, db.AutoMigrate(
		&presetModels.PresetModel{},
		&presetModels.MachinePresetModel{},
		&presetModels.EnergyPresetModel{},
		&presetModels.CostPresetModel{},
		&profileModels.PrintProfileModel{},
	))
	require.NoError(t, profileRepoImpl.MigrateDefaults(db))

	presetRepo := presetRepoImpl.NewPresetRepository(db)
	profileRepo := profileRepoImpl.NewPrintProfileRepository(db)
	uc := usecases.NewProfileUseCase(profileRepo, presetRepo)
	return uc, profileRepo, presetRepo, db
}

func seedBasePreset(t *testing.T, db *gorm.DB, org, name string, pt presetEntities.PresetType) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Create(&presetModels.PresetModel{
		ID: id, Name: name, Type: string(pt), IsActive: true, OrganizationID: org,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}).Error)
	return id
}

func TestIntegration_Profile_CreateAndValidateRefs(t *testing.T) {
	uc, _, _, db := setup(t)

	machine := seedBasePreset(t, db, orgA, "Bambu P1S", presetEntities.PresetTypeMachine)
	energy := seedBasePreset(t, db, orgA, "CEMIG/MG", presetEntities.PresetTypeEnergy)
	cost := seedBasePreset(t, db, orgA, "Hobby", presetEntities.PresetTypeCost)

	resp, err := uc.Create(&entities.CreateProfileRequest{
		MachinePresetID: machine,
		EnergyPresetID:  energy,
		CostPresetID:    &cost,
	}, orgA, "user-1")
	require.NoError(t, err)
	assert.Equal(t, "Bambu P1S · CEMIG/MG", resp.Name)
	require.NotNil(t, resp.CostPreset)
	assert.Equal(t, "Hobby", resp.CostPreset.Name)
}

func TestIntegration_Profile_RejectsWrongTypeAndCrossOrg(t *testing.T) {
	uc, _, _, db := setup(t)

	// Wrong type: an energy preset used as the machine reference.
	energyA := seedBasePreset(t, db, orgA, "E", presetEntities.PresetTypeEnergy)
	_, err := uc.Create(&entities.CreateProfileRequest{
		MachinePresetID: energyA,
		EnergyPresetID:  energyA,
	}, orgA, "u")
	assert.ErrorIs(t, err, entities.ErrInvalidMachinePreset)

	// Cross-org: presets belong to orgB; orgA cannot reference them.
	machineB := seedBasePreset(t, db, orgB, "M", presetEntities.PresetTypeMachine)
	energyB := seedBasePreset(t, db, orgB, "E", presetEntities.PresetTypeEnergy)
	_, err = uc.Create(&entities.CreateProfileRequest{
		MachinePresetID: machineB,
		EnergyPresetID:  energyB,
	}, orgA, "u")
	assert.ErrorIs(t, err, entities.ErrInvalidMachinePreset)
}

func TestIntegration_Profile_SingleDefaultEnforced(t *testing.T) {
	uc, profileRepo, _, db := setup(t)

	machine := seedBasePreset(t, db, orgA, "M", presetEntities.PresetTypeMachine)
	energy := seedBasePreset(t, db, orgA, "E", presetEntities.PresetTypeEnergy)

	first, err := uc.Create(&entities.CreateProfileRequest{MachinePresetID: machine, EnergyPresetID: energy, IsDefault: true}, orgA, "u")
	require.NoError(t, err)
	second, err := uc.Create(&entities.CreateProfileRequest{MachinePresetID: machine, EnergyPresetID: energy, IsDefault: true}, orgA, "u")
	require.NoError(t, err)

	gotFirst, err := profileRepo.GetByID(uuid.MustParse(first.ID), orgA)
	require.NoError(t, err)
	gotSecond, err := profileRepo.GetByID(uuid.MustParse(second.ID), orgA)
	require.NoError(t, err)
	assert.False(t, gotFirst.IsDefault, "creating a second default must clear the first")
	assert.True(t, gotSecond.IsDefault)

	// The partial unique index must reject a direct second live default.
	err = db.Create(&profileModels.PrintProfileModel{
		ID: uuid.New(), OrganizationID: orgA, Name: "dup",
		MachinePresetID: machine, EnergyPresetID: energy, IsDefault: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}).Error
	assert.Error(t, err, "partial unique index should reject a second live default profile")
}

func TestIntegration_Profile_DuplicateAndDelete(t *testing.T) {
	uc, _, _, db := setup(t)

	machine := seedBasePreset(t, db, orgA, "M", presetEntities.PresetTypeMachine)
	energy := seedBasePreset(t, db, orgA, "E", presetEntities.PresetTypeEnergy)
	created, err := uc.Create(&entities.CreateProfileRequest{Name: "Perfil", MachinePresetID: machine, EnergyPresetID: energy, IsDefault: true}, orgA, "u")
	require.NoError(t, err)

	// Default cannot be deleted.
	assert.ErrorIs(t, uc.Delete(uuid.MustParse(created.ID), orgA), entities.ErrCannotDeleteDefaultProfile)

	// Duplicate is a non-default copy named "<name> (cópia)".
	dup, err := uc.Duplicate(uuid.MustParse(created.ID), orgA)
	require.NoError(t, err)
	assert.Equal(t, "Perfil (cópia)", dup.Name)
	assert.False(t, dup.IsDefault)

	// The duplicate (non-default) can be deleted.
	require.NoError(t, uc.Delete(uuid.MustParse(dup.ID), orgA))
}
