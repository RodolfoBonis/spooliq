package repositories_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/RodolfoBonis/spooliq/core/database"
	repoimpl "github.com/RodolfoBonis/spooliq/features/model3d/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// These tests exercise the REAL GORM model3d repository against a real PostgreSQL,
// proving the list queries are organization-scoped with working filters/search/sort
// and that the partial unique index (organization_id, file_hash) WHERE deleted_at
// IS NULL dedups live rows while allowing re-upload after a soft delete. They are
// gated by TEST_DATABASE_URL and skip when it is unset.
//
//	make infrastructure/raise
//	TEST_DATABASE_URL='postgres://user:password@localhost:5432/spooliq_db?sslmode=disable' \
//	    go test ./features/model3d/data/repositories/...

const (
	itOrgA = "m3d-org-a"
	itOrgB = "m3d-org-b"
)

func setupModel3DRepo(t *testing.T) (repositories.Model3DRepository, *gorm.DB) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping model3d repository integration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "connect to TEST_DATABASE_URL")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1) // pin connection so SET search_path persists

	schema := fmt.Sprintf("m3d_it_%s", uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)

	t.Cleanup(func() {
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})

	require.NoError(t, db.Exec(`
		CREATE TABLE models_3d (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			customer_id uuid,
			name varchar(255) NOT NULL,
			description text,
			file_name varchar(255) NOT NULL,
			file_url varchar(1024) NOT NULL,
			file_format varchar(10) NOT NULL,
			file_size_bytes bigint NOT NULL,
			file_hash varchar(64) NOT NULL,
			thumbnail_url varchar(1024),
			notes text,
			tags text,
			owner_user_id varchar(255) NOT NULL,
			created_at timestamptz,
			updated_at timestamptz,
			deleted_at timestamptz
		)`).Error)

	require.NoError(t, db.Exec(`
		CREATE UNIQUE INDEX uq_models_3d_org_file_hash
		ON models_3d(organization_id, file_hash) WHERE deleted_at IS NULL`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE customers (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			name varchar(255) NOT NULL,
			deleted_at timestamptz
		)`).Error)

	return repoimpl.NewModel3DRepository(db), db
}

type seedOpts struct {
	name       string
	tags       string
	format     string
	sizeBytes  int64
	hash       string
	customerID *uuid.UUID
	createdAt  time.Time
}

func seedModel(t *testing.T, repo repositories.Model3DRepository, org string, o seedOpts) uuid.UUID {
	t.Helper()
	id := uuid.New()
	var tags *string
	if o.tags != "" {
		tags = &o.tags
	}
	if o.hash == "" {
		o.hash = uuid.New().String()
	}
	if o.createdAt.IsZero() {
		o.createdAt = time.Now()
	}
	err := repo.Create(context.Background(), &entities.Model3DEntity{
		ID:             id,
		OrganizationID: org,
		CustomerID:     o.customerID,
		Name:           o.name,
		FileName:       o.name + o.format,
		FileURL:        "https://cdn/" + id.String() + o.format,
		FileFormat:     o.format,
		FileSizeBytes:  o.sizeBytes,
		FileHash:       o.hash,
		Tags:           tags,
		OwnerUserID:    "owner",
		CreatedAt:      o.createdAt,
		UpdatedAt:      o.createdAt,
	})
	require.NoError(t, err)
	return id
}

func TestModel3DRepo_ListScopingAndFilters(t *testing.T) {
	repo, _ := setupModel3DRepo(t)
	ctx := context.Background()
	now := time.Now()

	cid := uuid.New()
	seedModel(t, repo, itOrgA, seedOpts{name: "Dragon", tags: "fantasy,toy", format: ".stl", sizeBytes: 300, createdAt: now.Add(-3 * time.Hour)})
	seedModel(t, repo, itOrgA, seedOpts{name: "Vase", tags: "home,decor", format: ".3mf", sizeBytes: 100, createdAt: now.Add(-2 * time.Hour), customerID: &cid})
	seedModel(t, repo, itOrgA, seedOpts{name: "Gear", tags: "mechanical", format: ".stl", sizeBytes: 200, createdAt: now.Add(-1 * time.Hour)})
	seedModel(t, repo, itOrgB, seedOpts{name: "Dragon B", tags: "fantasy", format: ".stl", sizeBytes: 999, createdAt: now})

	// Org scoping: only org A rows.
	all, total, err := repo.FindAll(ctx, itOrgA, repositories.Model3DFilters{}, "", "created_at desc, id asc", 20, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	assert.Len(t, all, 3)

	// Search by name.
	byName, total, err := repo.FindAll(ctx, itOrgA, repositories.Model3DFilters{}, "dragon", "created_at desc, id asc", 20, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, byName, 1)
	assert.Equal(t, "Dragon", byName[0].Name)

	// Search by tags.
	byTag, total, err := repo.FindAll(ctx, itOrgA, repositories.Model3DFilters{}, "mechanical", "", 20, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, byTag, 1)
	assert.Equal(t, "Gear", byTag[0].Name)

	// Filter by format.
	stls, total, err := repo.FindAll(ctx, itOrgA, repositories.Model3DFilters{Format: ".stl"}, "", "", 20, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	assert.Len(t, stls, 2)

	// Filter by customer_id.
	byCust, total, err := repo.FindAll(ctx, itOrgA, repositories.Model3DFilters{CustomerID: &cid}, "", "", 20, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, byCust, 1)
	assert.Equal(t, "Vase", byCust[0].Name)

	// Sort by file_size_bytes asc (tie-break id).
	bySize, _, err := repo.FindAll(ctx, itOrgA, repositories.Model3DFilters{}, "", "file_size_bytes asc, id asc", 20, 0)
	require.NoError(t, err)
	require.Len(t, bySize, 3)
	assert.Equal(t, "Vase", bySize[0].Name)   // 100
	assert.Equal(t, "Gear", bySize[1].Name)   // 200
	assert.Equal(t, "Dragon", bySize[2].Name) // 300

	// Search escapes LIKE metacharacters: a literal % must not act as a wildcard.
	none, total, err := repo.FindAll(ctx, itOrgA, repositories.Model3DFilters{}, "%", "", 20, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 0, total)
	assert.Empty(t, none)
}

func TestModel3DRepo_PartialUniqueIndex(t *testing.T) {
	repo, _ := setupModel3DRepo(t)
	ctx := context.Background()

	hash := "deadbeef"
	id1 := seedModel(t, repo, itOrgA, seedOpts{name: "A", format: ".stl", hash: hash})

	// Same (org, hash) while the first is live -> unique violation.
	err := repo.Create(ctx, &entities.Model3DEntity{
		ID: uuid.New(), OrganizationID: itOrgA, Name: "Dup", FileName: "d.stl",
		FileURL: "https://cdn/d.stl", FileFormat: ".stl", FileSizeBytes: 1, FileHash: hash,
		OwnerUserID: "owner", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	require.Error(t, err)
	assert.True(t, database.IsUniqueViolation(err), "expected unique violation, got %v", err)

	// A different org may reuse the same hash.
	require.NoError(t, repo.Create(ctx, &entities.Model3DEntity{
		ID: uuid.New(), OrganizationID: itOrgB, Name: "OtherOrg", FileName: "o.stl",
		FileURL: "https://cdn/o.stl", FileFormat: ".stl", FileSizeBytes: 1, FileHash: hash,
		OwnerUserID: "owner", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	// Soft-deleting the first frees the (org, hash) so the same content can be re-uploaded.
	require.NoError(t, repo.Delete(ctx, id1, itOrgA))
	require.NoError(t, repo.Create(ctx, &entities.Model3DEntity{
		ID: uuid.New(), OrganizationID: itOrgA, Name: "ReUpload", FileName: "r.stl",
		FileURL: "https://cdn/r.stl", FileFormat: ".stl", FileSizeBytes: 1, FileHash: hash,
		OwnerUserID: "owner", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
}

func TestModel3DRepo_FindByHashAndCustomerAndDeleteScope(t *testing.T) {
	repo, db := setupModel3DRepo(t)
	ctx := context.Background()

	cid := uuid.New()
	id := seedModel(t, repo, itOrgA, seedOpts{name: "X", format: ".stl", hash: "h1", customerID: &cid})

	// FindByHash: present in org, absent cross-org, nil when missing.
	got, err := repo.FindByHash(ctx, "h1", itOrgA)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, id, got.ID)

	cross, err := repo.FindByHash(ctx, "h1", itOrgB)
	require.NoError(t, err)
	assert.Nil(t, cross)

	missing, err := repo.FindByHash(ctx, "nope", itOrgA)
	require.NoError(t, err)
	assert.Nil(t, missing)

	// FindByCustomerID is org-scoped and flat.
	list, err := repo.FindByCustomerID(ctx, cid, itOrgA)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	empty, err := repo.FindByCustomerID(ctx, cid, itOrgB)
	require.NoError(t, err)
	assert.Empty(t, empty)

	// Delete is org-scoped: wrong org is a no-op; correct org soft-deletes.
	require.NoError(t, repo.Delete(ctx, id, itOrgB))
	_, err = repo.FindByID(ctx, id, itOrgA)
	require.NoError(t, err, "wrong-org delete must not remove the row")

	require.NoError(t, repo.Delete(ctx, id, itOrgA))
	_, err = repo.FindByID(ctx, id, itOrgA)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Confirm the row is only soft-deleted (still present with deleted_at set).
	var cnt int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM models_3d WHERE id = ? AND deleted_at IS NOT NULL`, id).Scan(&cnt).Error)
	assert.EqualValues(t, 1, cnt)
}

func TestModel3DRepo_CustomerExists(t *testing.T) {
	repo, db := setupModel3DRepo(t)
	ctx := context.Background()

	cid := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO customers (id, organization_id, name) VALUES (?, ?, ?)`, cid, itOrgA, "C").Error)

	ok, err := repo.CustomerExists(ctx, cid, itOrgA)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = repo.CustomerExists(ctx, cid, itOrgB)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestModel3DRepo_UpdatePersistsAndClearsCustomer(t *testing.T) {
	repo, _ := setupModel3DRepo(t)
	ctx := context.Background()

	cid := uuid.New()
	id := seedModel(t, repo, itOrgA, seedOpts{name: "Before", format: ".stl", hash: "h2", customerID: &cid})

	m, err := repo.FindByID(ctx, id, itOrgA)
	require.NoError(t, err)
	m.Name = "After"
	m.CustomerID = nil // clear
	require.NoError(t, repo.Update(ctx, m))

	reloaded, err := repo.FindByID(ctx, id, itOrgA)
	require.NoError(t, err)
	assert.Equal(t, "After", reloaded.Name)
	assert.Nil(t, reloaded.CustomerID, "explicit nil customer_id must be persisted as NULL")
}
