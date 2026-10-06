package repositories_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	repoimpl "github.com/RodolfoBonis/spooliq/features/slicer/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/suggest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// This test exercises the REAL GORM catalog repository against a real PostgreSQL,
// proving candidate loading is organization-scoped (never leaks another org's
// filaments), joins the material name, and excludes inactive/soft-deleted rows.
// It is gated by TEST_DATABASE_URL and skips when unset.
//
//	make infrastructure/raise
//	TEST_DATABASE_URL='postgres://user:password@localhost:5432/spooliq_db?sslmode=disable' \
//	    go test ./features/slicer/data/repositories/...

const (
	orgA = "slicer-org-a"
	orgB = "slicer-org-b"
)

func setupCatalogDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping slicer catalog integration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "connect to TEST_DATABASE_URL")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1) // pin connection so SET search_path persists

	schema := fmt.Sprintf("slicer_it_%s", uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)

	t.Cleanup(func() {
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})

	require.NoError(t, db.Exec(`
		CREATE TABLE materials (
			id uuid PRIMARY KEY,
			name varchar(255) NOT NULL
		)`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE filaments (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			name varchar(255) NOT NULL,
			color_hex varchar(7),
			material_id uuid,
			is_active boolean NOT NULL DEFAULT true,
			deleted_at timestamptz
		)`).Error)

	return db
}

func insertMaterial(t *testing.T, db *gorm.DB, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO materials (id, name) VALUES (?, ?)`, id, name).Error)
	return id
}

func insertFilament(t *testing.T, db *gorm.DB, org, name, colorHex string, materialID uuid.UUID, active bool, deleted bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if deleted {
		require.NoError(t, db.Exec(
			`INSERT INTO filaments (id, organization_id, name, color_hex, material_id, is_active, deleted_at) VALUES (?, ?, ?, ?, ?, ?, now())`,
			id, org, name, colorHex, materialID, active).Error)
	} else {
		require.NoError(t, db.Exec(
			`INSERT INTO filaments (id, organization_id, name, color_hex, material_id, is_active) VALUES (?, ?, ?, ?, ?, ?)`,
			id, org, name, colorHex, materialID, active).Error)
	}
	return id
}

func TestCatalogRepository_OrgIsolation(t *testing.T) {
	db := setupCatalogDB(t)
	repo := repoimpl.NewFilamentCatalogRepository(db)
	ctx := context.Background()

	plaA := insertMaterial(t, db, "PLA")
	plaB := insertMaterial(t, db, "PLA")

	redA := insertFilament(t, db, orgA, "Red PLA", "#FF0000", plaA, true, false)
	insertFilament(t, db, orgA, "Green PLA", "#00FF00", plaA, true, false)
	insertFilament(t, db, orgA, "Inactive PLA", "#123456", plaA, false, false) // excluded (inactive)
	insertFilament(t, db, orgA, "Deleted PLA", "#654321", plaA, true, true)    // excluded (soft-deleted)
	blueB := insertFilament(t, db, orgB, "Blue PLA", "#0000FF", plaB, true, false)

	// Org A sees only its own two active, non-deleted filaments.
	candA, err := repo.LoadCandidates(ctx, orgA)
	require.NoError(t, err)
	require.Len(t, candA, 2, "org A must see exactly its 2 active filaments")

	ids := map[string]bool{}
	for _, c := range candA {
		ids[c.FilamentID] = true
		assert.Equal(t, "PLA", c.Material, "material name must be joined")
	}
	assert.True(t, ids[redA.String()])
	assert.False(t, ids[blueB.String()], "org A must NOT see org B's filament")

	// Suggestions over org A's catalog match red exactly and never return org B's blue.
	s := suggest.Match("#FF0000", "PLA", candA)
	require.NotNil(t, s)
	assert.Equal(t, redA.String(), s.FilamentID)
	assert.Equal(t, entities.ConfidenceExact, s.Confidence)

	// Org B sees only its own filament.
	candB, err := repo.LoadCandidates(ctx, orgB)
	require.NoError(t, err)
	require.Len(t, candB, 1)
	assert.Equal(t, blueB.String(), candB[0].FilamentID)
}
