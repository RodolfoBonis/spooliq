package repositories_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/preset/data/models"
	repoimpl "github.com/RodolfoBonis/spooliq/features/preset/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// These tests exercise the REAL GORM repository against a real PostgreSQL, so the
// actual SQL WHERE clauses that enforce tenant isolation and soft-delete are
// verified — something the in-memory fake (which reimplements the contract) cannot
// prove. They are gated by TEST_DATABASE_URL and skip when it is unset.
//
// To run locally:
//
//	make infrastructure/raise   # starts postgres on :5432 (user/password/spooliq_db)
//	TEST_DATABASE_URL='postgres://user:password@localhost:5432/spooliq_db?sslmode=disable' \
//	    go test ./features/preset/data/repositories/...
//
// Each run creates an isolated schema and drops it on cleanup, so parallel runs
// and leftover state never interfere. Foreign-key constraint creation is disabled
// during migration (DisableForeignKeyConstraintWhenMigrating), so the companies
// table is not required: org scoping is enforced by WHERE clauses, which is
// exactly what we are testing.

const (
	itOrgA = "org-a"
	itOrgB = "org-b"
)

// setupRepo connects to TEST_DATABASE_URL, creates an isolated schema, migrates the
// preset models into it and returns a repository bound to that schema. It skips the
// test when the env var is absent.
func setupRepo(t *testing.T) (repositories.PresetRepository, *gorm.DB) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping preset repository integration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "connect to TEST_DATABASE_URL")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	// Pin to a single connection so SET search_path persists for the whole test.
	sqlDB.SetMaxOpenConns(1)

	schema := fmt.Sprintf("preset_it_%s", uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)

	t.Cleanup(func() {
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})

	require.NoError(t, db.AutoMigrate(
		&models.PresetModel{},
		&models.MachinePresetModel{},
		&models.EnergyPresetModel{},
		&models.CostPresetModel{},
	), "migrate preset models")

	return repoimpl.NewPresetRepository(db), db
}

// seedMachine inserts a machine preset (base + child) for org via the repository.
func seedMachine(t *testing.T, repo repositories.PresetRepository, org string, active, isDefault bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	preset := &entities.PresetEntity{
		ID:             id,
		Name:           "Printer",
		Type:           entities.PresetTypeMachine,
		IsActive:       active,
		IsDefault:      isDefault,
		OrganizationID: org,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	machine := &entities.MachinePresetEntity{
		ID:               id,
		OrganizationID:   org,
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
	require.NoError(t, repo.CreateMachine(preset, machine))
	return id
}

func TestIntegration_GetByID_CrossOrgReturnsNotFound(t *testing.T) {
	repo, _ := setupRepo(t)

	id := seedMachine(t, repo, itOrgB, true, false)

	// Owner reads it.
	got, err := repo.GetByID(id, itOrgB)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)

	// Another org gets a not-found (no leak), proving the org_id WHERE clause.
	_, err = repo.GetByID(id, itOrgA)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestIntegration_Update_CrossOrgReturnsNotFound(t *testing.T) {
	repo, _ := setupRepo(t)

	id := seedMachine(t, repo, itOrgB, true, false)

	// Update attempt scoped to the wrong org must not affect any row.
	preset := &entities.PresetEntity{
		ID:             id,
		Name:           "Hacked",
		Type:           entities.PresetTypeMachine,
		IsActive:       true,
		OrganizationID: itOrgA, // attacker's org
		UpdatedAt:      time.Now(),
	}
	err := repo.Update(preset)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Original row is untouched.
	stored, err := repo.GetByID(id, itOrgB)
	require.NoError(t, err)
	assert.Equal(t, "Printer", stored.Name)
}

func TestIntegration_Delete_CrossOrgReturnsNotFound(t *testing.T) {
	repo, _ := setupRepo(t)

	id := seedMachine(t, repo, itOrgB, true, false)

	err := repo.Delete(id, itOrgA)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Still readable by its real owner (not soft-deleted).
	_, err = repo.GetByID(id, itOrgB)
	require.NoError(t, err)
}

func TestIntegration_SoftDeleteExcludedFromReads(t *testing.T) {
	repo, _ := setupRepo(t)

	id := seedMachine(t, repo, itOrgA, true, false)

	require.NoError(t, repo.Delete(id, itOrgA))

	// Base read excludes soft-deleted rows.
	_, err := repo.GetByID(id, itOrgA)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Typed getter (joined on presets.deleted_at IS NULL) also excludes it.
	_, err = repo.GetMachineByID(id, itOrgA)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// List excludes it too.
	list, err := repo.ListPresets(itOrgA, entities.PresetFilters{})
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestIntegration_ListPresets_FiltersAndOrgScope(t *testing.T) {
	repo, _ := setupRepo(t)

	// org-a: one active+default machine, one inactive non-default machine.
	activeDefault := seedMachine(t, repo, itOrgA, true, true)
	seedMachine(t, repo, itOrgA, false, false)
	// org-b: one machine that must never appear for org-a.
	seedMachine(t, repo, itOrgB, true, false)

	// Org scope: org-a sees only its two presets.
	all, err := repo.ListPresets(itOrgA, entities.PresetFilters{})
	require.NoError(t, err)
	assert.Len(t, all, 2)
	for _, p := range all {
		assert.Equal(t, itOrgA, p.OrganizationID)
	}

	// Combinable filters: type=machine AND active AND default -> only activeDefault.
	machineType := entities.PresetTypeMachine
	filtered, err := repo.ListPresets(itOrgA, entities.PresetFilters{
		Type:        &machineType,
		ActiveOnly:  true,
		DefaultOnly: true,
	})
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	assert.Equal(t, activeDefault, filtered[0].ID)
}

func TestIntegration_TypedGetters_ExcludeSoftDeletedAndCrossOrg(t *testing.T) {
	repo, _ := setupRepo(t)

	id := seedMachine(t, repo, itOrgB, true, false)

	// Cross-org typed getter -> not found.
	_, err := repo.GetMachineByID(id, itOrgA)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// GetMachinePresets (optimized) is org-scoped and active-only.
	resA, _, err := repo.GetMachinePresets(itOrgA, helpers.ListQuery{})
	require.NoError(t, err)
	assert.Empty(t, resA)

	resB, _, err := repo.GetMachinePresets(itOrgB, helpers.ListQuery{})
	require.NoError(t, err)
	assert.Len(t, resB, 1)
}

func TestIntegration_UpdateMachineWithPreset_AtomicAndScoped(t *testing.T) {
	repo, _ := setupRepo(t)

	id := seedMachine(t, repo, itOrgA, true, false)

	preset, err := repo.GetByID(id, itOrgA)
	require.NoError(t, err)
	machine, err := repo.GetMachineByID(id, itOrgA)
	require.NoError(t, err)

	preset.Name = "Updated"
	machine.Brand = "Bambu"
	require.NoError(t, repo.UpdateMachineWithPreset(preset, machine))

	gotPreset, err := repo.GetByID(id, itOrgA)
	require.NoError(t, err)
	assert.Equal(t, "Updated", gotPreset.Name)
	gotMachine, err := repo.GetMachineByID(id, itOrgA)
	require.NoError(t, err)
	assert.Equal(t, "Bambu", gotMachine.Brand)

	// Wrong-org atomic update must roll back entirely (base row unchanged).
	preset.OrganizationID = itOrgB
	preset.Name = "Hacked"
	machine.OrganizationID = itOrgB
	err = repo.UpdateMachineWithPreset(preset, machine)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	unchanged, err := repo.GetByID(id, itOrgA)
	require.NoError(t, err)
	assert.Equal(t, "Updated", unchanged.Name)
}
