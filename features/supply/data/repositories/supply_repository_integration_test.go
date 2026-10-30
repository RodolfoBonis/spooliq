package repositories_test

import (
	"fmt"
	"os"
	"testing"

	repoimpl "github.com/RodolfoBonis/spooliq/features/supply/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/supply/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/supply/domain/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// These tests exercise the REAL GORM supply repository against a real PostgreSQL,
// focused on the behaviors the in-memory fake cannot prove: the list query is
// organization-scoped, honors the is_active filter, and a PUT persists explicit zero
// values (unit_cost: 0, is_active: false). Gated by TEST_DATABASE_URL.

const (
	itOrgA = "sup-org-a"
	itOrgB = "sup-org-b"
)

func setupSupplyRepo(t *testing.T) repositories.SupplyRepository {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping supply repository integration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "connect to TEST_DATABASE_URL")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	schema := fmt.Sprintf("sup_it_%s", uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)

	t.Cleanup(func() {
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})

	require.NoError(t, db.Exec(`
		CREATE TABLE supplies (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			name varchar(255) NOT NULL,
			unit varchar(20) NOT NULL,
			unit_cost bigint NOT NULL DEFAULT 0,
			description text,
			is_active boolean NOT NULL DEFAULT true,
			created_at timestamptz,
			updated_at timestamptz,
			deleted_at timestamptz
		)`).Error)

	return repoimpl.NewSupplyRepository(db)
}

// TestSupplyUpdatePersistsZeroAndFalse verifies a PUT lowering unit_cost to 0 and
// is_active to false is actually written (the Select-based Update), rather than
// dropped as GORM zero-values.
func TestSupplyUpdatePersistsZeroAndFalse(t *testing.T) {
	repo := setupSupplyRepo(t)

	s := &entities.SupplyEntity{
		ID:             uuid.New(),
		OrganizationID: itOrgA,
		Name:           "LED",
		Unit:           "un",
		UnitCost:       150,
		IsActive:       true,
	}
	require.NoError(t, repo.Create(s))

	s.UnitCost = 0
	s.IsActive = false
	require.NoError(t, repo.Update(s))

	got, err := repo.FindByID(s.ID, itOrgA)
	require.NoError(t, err)
	assert.Equal(t, int64(0), got.UnitCost, "unit_cost: 0 must persist")
	assert.False(t, got.IsActive, "is_active: false must persist")
}

// TestSupplyListOrgScopedSearchAndActiveFilter verifies org scoping, the name search
// and the is_active filter.
func TestSupplyListOrgScopedSearchAndActiveFilter(t *testing.T) {
	repo := setupSupplyRepo(t)

	require.NoError(t, repo.Create(&entities.SupplyEntity{ID: uuid.New(), OrganizationID: itOrgA, Name: "LED", Unit: "un", IsActive: true}))
	require.NoError(t, repo.Create(&entities.SupplyEntity{ID: uuid.New(), OrganizationID: itOrgA, Name: "Creme", Unit: "ml", IsActive: false}))
	require.NoError(t, repo.Create(&entities.SupplyEntity{ID: uuid.New(), OrganizationID: itOrgB, Name: "Chaveiro", Unit: "un", IsActive: true}))

	// Org A sees only its two supplies.
	rows, total, err := repo.FindAll(itOrgA, "", nil, "lower(name) asc", 20, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, rows, 2)
	assert.Equal(t, "Creme", rows[0].Name)
	assert.Equal(t, "LED", rows[1].Name)

	// is_active=true filter.
	active := true
	rows, total, err = repo.FindAll(itOrgA, "", &active, "lower(name) asc", 20, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, "LED", rows[0].Name)

	// is_active=false filter.
	inactive := false
	rows, total, err = repo.FindAll(itOrgA, "", &inactive, "lower(name) asc", 20, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, "Creme", rows[0].Name)

	// Name search.
	rows, total, err = repo.FindAll(itOrgA, "led", nil, "lower(name) asc", 20, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, "LED", rows[0].Name)
}
