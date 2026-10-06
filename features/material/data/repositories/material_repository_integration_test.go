package repositories_test

import (
	"fmt"
	"os"
	"testing"

	repoimpl "github.com/RodolfoBonis/spooliq/features/material/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/material/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/material/domain/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// These tests exercise the REAL GORM material repository against a real
// PostgreSQL, focused on two behaviors the in-memory fake cannot prove: the
// list query is organization-scoped, and a PUT persists explicit zero values
// (temp_table: 0) rather than skipping them. Gated by TEST_DATABASE_URL.

const (
	itOrgA = "mat-org-a"
	itOrgB = "mat-org-b"
)

func setupMaterialRepo(t *testing.T) (repositories.MaterialRepository, *gorm.DB) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping material repository integration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "connect to TEST_DATABASE_URL")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	schema := fmt.Sprintf("mat_it_%s", uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)

	t.Cleanup(func() {
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})

	require.NoError(t, db.Exec(`
		CREATE TABLE materials (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			name varchar(255) NOT NULL,
			description text,
			temp_table real,
			temp_extruder real,
			created_at timestamptz,
			updated_at timestamptz,
			deleted_at timestamptz
		)`).Error)

	return repoimpl.NewMaterialRepository(db), db
}

// TestMaterialUpdatePersistsZero verifies a PUT that sets temp_table to 0 is
// actually written (the Select-based Update), rather than being dropped as a
// GORM zero-value.
func TestMaterialUpdatePersistsZero(t *testing.T) {
	repo, _ := setupMaterialRepo(t)

	m := &entities.MaterialEntity{
		ID:             uuid.New(),
		OrganizationID: itOrgA,
		Name:           "PLA",
		TempTable:      60,
		TempExtruder:   210,
	}
	require.NoError(t, repo.Create(m))

	// Full PUT lowering both temperatures to zero.
	m.TempTable = 0
	m.TempExtruder = 0
	require.NoError(t, repo.Update(m))

	got, err := repo.FindByID(m.ID, itOrgA)
	require.NoError(t, err)
	assert.Equal(t, float32(0), got.TempTable, "temp_table: 0 must persist")
	assert.Equal(t, float32(0), got.TempExtruder, "temp_extruder: 0 must persist")
}

// TestMaterialListOrgScopedAndSearch verifies org scoping, pagination and the
// case-insensitive name search.
func TestMaterialListOrgScopedAndSearch(t *testing.T) {
	repo, _ := setupMaterialRepo(t)

	require.NoError(t, repo.Create(&entities.MaterialEntity{ID: uuid.New(), OrganizationID: itOrgA, Name: "PLA"}))
	require.NoError(t, repo.Create(&entities.MaterialEntity{ID: uuid.New(), OrganizationID: itOrgA, Name: "PETG"}))
	require.NoError(t, repo.Create(&entities.MaterialEntity{ID: uuid.New(), OrganizationID: itOrgB, Name: "ABS"}))

	rows, total, err := repo.FindAll(itOrgA, "", "lower(name) asc", 20, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, rows, 2)
	assert.Equal(t, "PETG", rows[0].Name)
	assert.Equal(t, "PLA", rows[1].Name)

	rows, total, err = repo.FindAll(itOrgA, "pet", "lower(name) asc", 20, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, "PETG", rows[0].Name)
}
