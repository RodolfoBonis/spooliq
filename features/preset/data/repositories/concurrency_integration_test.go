package repositories_test

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

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

// setupConcurrentRepo builds a repository backed by a MULTI-connection pool bound
// to an isolated schema (via the search_path connection parameter so it applies
// to every pooled connection). This is required to exercise genuinely concurrent
// transactions, unlike the single-connection setupRepo used elsewhere.
func setupConcurrentRepo(t *testing.T) (repositories.PresetRepository, *gorm.DB) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping preset concurrency integration test")
	}

	schema := fmt.Sprintf("preset_cc_%s", uuid.New().String()[:8])

	// Admin connection to create/drop the schema.
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	adminSQL, err := admin.DB()
	require.NoError(t, err)
	require.NoError(t, admin.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)

	// Pool connection with search_path pinned to the schema for every connection.
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	poolDSN := dsn + sep + "search_path=" + schema

	db, err := gorm.Open(postgres.Open(poolDSN), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(8)

	t.Cleanup(func() {
		_ = sqlDB.Close()
		_ = admin.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = adminSQL.Close()
	})

	require.NoError(t, db.AutoMigrate(
		&models.PresetModel{},
		&models.MachinePresetModel{},
		&models.EnergyPresetModel{},
		&models.CostPresetModel{},
	))
	require.NoError(t, repoimpl.MigrateDefaults(db))

	return repoimpl.NewPresetRepository(db), db
}

// Two goroutines setting different presets as the default for the same
// (org, type) concurrently must both succeed (serialized by the advisory lock),
// leaving exactly one default and never surfacing a 23505 / 500.
func TestIntegration_SetDefault_ConcurrentSingleWinner(t *testing.T) {
	repo, db := setupConcurrentRepo(t)

	const org = "org-cc"
	a := seedEnergyPool(t, repo, org)
	b := seedEnergyPool(t, repo, org)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	ids := []uuid.UUID{a, b}
	wg.Add(2)
	for i := range ids {
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = repo.SetDefault(ids[idx], org)
		}(i)
	}
	wg.Wait()

	// The advisory lock serializes the two transactions, so BOTH must succeed
	// (the second clears the first's default before setting its own). Without the
	// lock the loser would hit the partial unique index; even then it must be the
	// friendly domain error, never a raw 23505 / 500.
	for _, err := range errs {
		if err != nil {
			assert.ErrorIs(t, err, entities.ErrDefaultConflict)
		}
	}
	assert.NoError(t, errs[0], "advisory lock should let the first set-default succeed")
	assert.NoError(t, errs[1], "advisory lock should let the second set-default succeed")

	var defaults int64
	require.NoError(t, db.Model(&models.PresetModel{}).
		Where("organization_id = ? AND type = ? AND is_default = ? AND deleted_at IS NULL", org, string(entities.PresetTypeEnergy), true).
		Count(&defaults).Error)
	assert.Equal(t, int64(1), defaults, "exactly one default must remain after concurrent set-default")
}

// seedEnergyPool inserts a non-default energy preset via the repository.
func seedEnergyPool(t *testing.T, repo repositories.PresetRepository, org string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	preset := &entities.PresetEntity{
		ID: id, Name: "Tarifa", Type: entities.PresetTypeEnergy,
		IsActive: true, OrganizationID: org, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	energy := &entities.EnergyPresetEntity{
		ID: id, OrganizationID: org, EnergyCostPerKwh: 0.85, Currency: "BRL",
		PeakHourMultiplier: 1.0, OffPeakHourMultiplier: 1.0,
	}
	require.NoError(t, repo.CreateEnergy(preset, energy))
	return id
}
