package repositories_test

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	repoimpl "github.com/RodolfoBonis/spooliq/features/filament/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/filament/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/filament/domain/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// These tests exercise the REAL GORM filament repository against a real
// PostgreSQL. They prove list/search are organization-scoped, that the free-text
// search matches filament/brand/material names, and — crucially — that the
// per-row brand/material info is resolved via a single batch query each (the
// N+1 fix) regardless of page size. Gated by TEST_DATABASE_URL.
//
//	make infrastructure/raise
//	TEST_DATABASE_URL='postgres://user:password@localhost:5432/spooliq_db?sslmode=disable' \
//	    go test ./features/filament/data/repositories/...

const (
	itOrgA = "fil-org-a"
	itOrgB = "fil-org-b"
)

type queryCounter struct{ n int64 }

func (q *queryCounter) reset()       { atomic.StoreInt64(&q.n, 0) }
func (q *queryCounter) count() int64 { return atomic.LoadInt64(&q.n) }

func (q *queryCounter) register(db *gorm.DB) {
	inc := func(*gorm.DB) { atomic.AddInt64(&q.n, 1) }
	_ = db.Callback().Query().After("gorm:query").Register("test:count_query", inc)
	_ = db.Callback().Raw().After("gorm:raw").Register("test:count_raw", inc)
	_ = db.Callback().Row().After("gorm:row").Register("test:count_row", inc)
}

func setupFilamentRepo(t *testing.T) (repositories.FilamentRepository, *gorm.DB, *queryCounter) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping filament repository integration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "connect to TEST_DATABASE_URL")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	schema := fmt.Sprintf("fil_it_%s", uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)

	t.Cleanup(func() {
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})

	require.NoError(t, db.Exec(`
		CREATE TABLE brands (
			id uuid PRIMARY KEY,
			organization_id varchar(255),
			name varchar(255),
			description text,
			created_at timestamptz,
			updated_at timestamptz,
			deleted_at timestamptz
		)`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE materials (
			id uuid PRIMARY KEY,
			organization_id varchar(255),
			name varchar(255),
			description text,
			temp_table real,
			temp_extruder real,
			created_at timestamptz,
			updated_at timestamptz,
			deleted_at timestamptz
		)`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE filaments (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			name varchar(255) NOT NULL,
			description text,
			brand_id uuid NOT NULL,
			material_id uuid NOT NULL,
			color varchar(100),
			color_hex varchar(7),
			color_type varchar(20),
			color_data text,
			color_preview text,
			diameter numeric NOT NULL,
			weight numeric,
			price_per_kg numeric NOT NULL,
			url text,
			owner_user_id varchar(255),
			is_active boolean,
			print_temperature integer,
			bed_temperature integer,
			created_at timestamptz,
			updated_at timestamptz,
			deleted_at timestamptz
		)`).Error)

	counter := &queryCounter{}
	counter.register(db)

	return repoimpl.NewFilamentRepository(db), db, counter
}

func seedBrand(t *testing.T, db *gorm.DB, org, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO brands (id, organization_id, name, description) VALUES (?, ?, ?, ?)`,
		id, org, name, "",
	).Error)
	return id
}

func seedMaterial(t *testing.T, db *gorm.DB, org, name string, tempTable, tempExtruder float32) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO materials (id, organization_id, name, description, temp_table, temp_extruder) VALUES (?, ?, ?, ?, ?, ?)`,
		id, org, name, "", tempTable, tempExtruder,
	).Error)
	return id
}

func seedFilament(t *testing.T, repo repositories.FilamentRepository, org, name string, brandID, materialID uuid.UUID, price float64) {
	t.Helper()
	owner := "owner"
	err := repo.Create(context.Background(), &entities.FilamentEntity{
		ID:             uuid.New(),
		OrganizationID: org,
		Name:           name,
		BrandID:        brandID,
		MaterialID:     materialID,
		Color:          "Preto",
		Diameter:       1.75,
		PricePerKg:     price,
		OwnerUserID:    &owner,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})
	require.NoError(t, err)
}

func TestFilamentListAndSearch(t *testing.T) {
	repo, db, _ := setupFilamentRepo(t)
	ctx := context.Background()

	brandA := seedBrand(t, db, itOrgA, "Voolt")
	brandB := seedBrand(t, db, itOrgA, "Prusament")
	mat := seedMaterial(t, db, itOrgA, "PLA", 60, 210)
	otherBrand := seedBrand(t, db, itOrgB, "OtherBrand")
	otherMat := seedMaterial(t, db, itOrgB, "ABS", 100, 250)

	seedFilament(t, repo, itOrgA, "Silk Gold", brandA, mat, 120.0)
	seedFilament(t, repo, itOrgA, "Matte Black", brandB, mat, 90.0)
	seedFilament(t, repo, itOrgB, "Hidden", otherBrand, otherMat, 50.0)

	// org scoping
	rows, total, err := repo.FindAll(ctx, itOrgA, "", "filaments.created_at desc", 20, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, rows, 2)

	// search matches filament name
	rows, total, err = repo.SearchFilaments(ctx, itOrgA, nil, "silk", "filaments.created_at desc", 20, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, "Silk Gold", rows[0].Name)

	// search matches the related brand name
	rows, total, err = repo.SearchFilaments(ctx, itOrgA, nil, "prusa", "filaments.created_at desc", 20, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, "Matte Black", rows[0].Name)

	// structured filter by brand_id
	rows, total, err = repo.SearchFilaments(ctx, itOrgA, map[string]interface{}{"brand_id": brandA}, "", "filaments.price_per_kg asc", 20, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, "Silk Gold", rows[0].Name)
}

// TestFilamentBatchInfoIsConstantQueries proves the brand/material info for a
// whole page is loaded with a single query each regardless of row count, which
// is the N+1 fix for the list/search endpoints.
func TestFilamentBatchInfoIsConstantQueries(t *testing.T) {
	repo, db, counter := setupFilamentRepo(t)
	ctx := context.Background()

	brandA := seedBrand(t, db, itOrgA, "Voolt")
	brandB := seedBrand(t, db, itOrgA, "Prusament")
	matA := seedMaterial(t, db, itOrgA, "PLA", 60, 210)
	matB := seedMaterial(t, db, itOrgA, "PETG", 80, 240)

	seedFilament(t, repo, itOrgA, "F1", brandA, matA, 100)
	seedFilament(t, repo, itOrgA, "F2", brandB, matB, 110)
	seedFilament(t, repo, itOrgA, "F3", brandA, matB, 120)

	rows, _, err := repo.FindAll(ctx, itOrgA, "", "filaments.created_at desc", 20, 0)
	require.NoError(t, err)
	require.Len(t, rows, 3)

	brandIDs := make([]uuid.UUID, 0, len(rows))
	materialIDs := make([]uuid.UUID, 0, len(rows))
	for _, f := range rows {
		brandIDs = append(brandIDs, f.BrandID)
		materialIDs = append(materialIDs, f.MaterialID)
	}

	counter.reset()
	brands, err := repo.GetBrandsInfo(ctx, brandIDs, itOrgA)
	require.NoError(t, err)
	assert.Equal(t, int64(1), counter.count(), "brand info for the whole page must be a single query")
	assert.Len(t, brands, 2) // two distinct brands across three rows

	counter.reset()
	materials, err := repo.GetMaterialsInfo(ctx, materialIDs, itOrgA)
	require.NoError(t, err)
	assert.Equal(t, int64(1), counter.count(), "material info for the whole page must be a single query")
	assert.Len(t, materials, 2)

	// Verify the material temperatures round-trip (temp_table/temp_extruder).
	assert.Equal(t, float32(60), materials[matA].TempTable)
	assert.Equal(t, float32(210), materials[matA].TempExtruder)

	// FindAll itself is a fixed 2 queries (count + select); combined with the two
	// batch lookups a full page costs exactly 4 queries, independent of row count.
	counter.reset()
	_, _, err = repo.FindAll(ctx, itOrgA, "", "filaments.created_at desc", 20, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), counter.count(), "FindAll must be count+select only")
}

// TestFilamentCrossOrgNoLeak proves the organization-scoped relationship
// helpers never resolve a brand/material from another tenant: a filament in
// org A that points at a brand id belonging to org B must not leak that brand's
// name, and the single-lookup form returns ErrRecordNotFound (which the create
// use case maps to a 400).
func TestFilamentCrossOrgNoLeak(t *testing.T) {
	repo, db, _ := setupFilamentRepo(t)
	ctx := context.Background()

	// Brand + material owned by org B.
	foreignBrand := seedBrand(t, db, itOrgB, "SecretBrand")
	foreignMat := seedMaterial(t, db, itOrgB, "SecretMat", 60, 210)

	// A filament in org A that (wrongly) references org B's brand/material.
	seedFilament(t, repo, itOrgA, "Leaky", foreignBrand, foreignMat, 100)

	// Batch lookup scoped to org A must not return org B's brand/material.
	brands, err := repo.GetBrandsInfo(ctx, []uuid.UUID{foreignBrand}, itOrgA)
	require.NoError(t, err)
	if _, ok := brands[foreignBrand]; ok {
		t.Error("brand from another org leaked through GetBrandsInfo")
	}
	materials, err := repo.GetMaterialsInfo(ctx, []uuid.UUID{foreignMat}, itOrgA)
	require.NoError(t, err)
	if _, ok := materials[foreignMat]; ok {
		t.Error("material from another org leaked through GetMaterialsInfo")
	}

	// Single lookup scoped to org A returns not found (-> create maps to 400).
	_, err = repo.GetBrandInfo(ctx, foreignBrand, itOrgA)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	_, err = repo.GetMaterialInfo(ctx, foreignMat, itOrgA)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// But the owner org still resolves them.
	_, err = repo.GetBrandInfo(ctx, foreignBrand, itOrgB)
	require.NoError(t, err)

	// A free-text search in org A for the org B brand name must not surface the
	// leaky filament (the join is matched on the same organization_id).
	rows, total, err := repo.SearchFilaments(ctx, itOrgA, nil, "SecretBrand", "filaments.created_at desc", 20, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, rows, 0)
}
