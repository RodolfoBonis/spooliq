package repositories_test

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	repoimpl "github.com/RodolfoBonis/spooliq/features/customer/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// These tests exercise the REAL GORM customer repository against a real
// PostgreSQL to prove the list queries are organization-scoped and, crucially,
// that per-row budget stats are resolved in a single grouped query (the N+1
// fix) regardless of how many customers are on the page. They are gated by
// TEST_DATABASE_URL and skip when it is unset.
//
//	make infrastructure/raise
//	TEST_DATABASE_URL='postgres://user:password@localhost:5432/spooliq_db?sslmode=disable' \
//	    go test ./features/customer/data/repositories/...

const (
	itOrgA = "cust-org-a"
	itOrgB = "cust-org-b"
)

// queryCounter counts SQL statements executed on a *gorm.DB via callbacks so a
// test can assert the number of round-trips is constant per page.
type queryCounter struct{ n int64 }

func (q *queryCounter) reset()       { atomic.StoreInt64(&q.n, 0) }
func (q *queryCounter) count() int64 { return atomic.LoadInt64(&q.n) }

func (q *queryCounter) register(db *gorm.DB) {
	inc := func(*gorm.DB) { atomic.AddInt64(&q.n, 1) }
	_ = db.Callback().Query().After("gorm:query").Register("test:count_query", inc)
	_ = db.Callback().Raw().After("gorm:raw").Register("test:count_raw", inc)
	_ = db.Callback().Row().After("gorm:row").Register("test:count_row", inc)
}

func setupCustomerRepo(t *testing.T) (repositories.CustomerRepository, *gorm.DB, *queryCounter) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping customer repository integration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "connect to TEST_DATABASE_URL")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1) // pin connection so SET search_path persists

	schema := fmt.Sprintf("cust_it_%s", uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)

	t.Cleanup(func() {
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})

	// Minimal DDL for exactly the columns the repository reads. Created by hand
	// (not AutoMigrate) to avoid pulling in cross-feature model associations.
	require.NoError(t, db.Exec(`
		CREATE TABLE customers (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			name varchar(255) NOT NULL,
			email varchar(255),
			phone varchar(50),
			document varchar(50),
			address varchar(500),
			city varchar(255),
			state varchar(100),
			zip_code varchar(20),
			notes text,
			owner_user_id varchar(255),
			is_active boolean DEFAULT true,
			created_at timestamptz,
			updated_at timestamptz,
			deleted_at timestamptz
		)`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE budgets (
			id uuid PRIMARY KEY,
			customer_id uuid NOT NULL,
			name varchar(255),
			status varchar(50),
			total_cost bigint,
			created_at timestamptz,
			deleted_at timestamptz
		)`).Error)

	counter := &queryCounter{}
	counter.register(db)

	return repoimpl.NewCustomerRepository(db), db, counter
}

func seedCustomer(t *testing.T, repo repositories.CustomerRepository, org, name, email string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	e := email
	err := repo.Create(context.Background(), &entities.CustomerEntity{
		ID:             id,
		OrganizationID: org,
		Name:           name,
		Email:          &e,
		OwnerUserID:    "owner",
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})
	require.NoError(t, err)
	return id
}

func seedBudget(t *testing.T, db *gorm.DB, customerID uuid.UUID, status string, total int64) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO budgets (id, customer_id, name, status, total_cost, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		uuid.New(), customerID, "B", status, total, time.Now(),
	).Error)
}

func TestCustomerListIsOrgScopedAndPaginated(t *testing.T) {
	repo, _, _ := setupCustomerRepo(t)
	ctx := context.Background()

	seedCustomer(t, repo, itOrgA, "Alpha", "alpha@x.com")
	seedCustomer(t, repo, itOrgA, "Beta", "beta@x.com")
	seedCustomer(t, repo, itOrgB, "Gamma", "gamma@x.com")

	// org scoping
	rows, total, err := repo.FindAll(ctx, itOrgA, "", "created_at desc", 20, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, rows, 2)

	// q search across name/email
	rows, total, err = repo.FindAll(ctx, itOrgA, "alpha", "created_at desc", 20, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, "Alpha", rows[0].Name)

	// pagination limit/offset
	rows, _, err = repo.FindAll(ctx, itOrgA, "", "lower(name) asc", 1, 1)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Beta", rows[0].Name)
}

// TestCustomerBudgetStatsIsConstantQueries proves GetBudgetStatsByCustomers
// issues exactly ONE query regardless of how many customers (and budgets) are
// involved — the core of the N+1 fix for the list endpoints.
func TestCustomerBudgetStatsIsConstantQueries(t *testing.T) {
	repo, db, counter := setupCustomerRepo(t)
	ctx := context.Background()

	c1 := seedCustomer(t, repo, itOrgA, "C1", "c1@x.com")
	c2 := seedCustomer(t, repo, itOrgA, "C2", "c2@x.com")
	c3 := seedCustomer(t, repo, itOrgA, "C3", "c3@x.com")

	// c1: 2 budgets (one counted, one not); c2: 1 counted budget; c3: none.
	seedBudget(t, db, c1, "completed", 1000)
	seedBudget(t, db, c1, "draft", 500)
	seedBudget(t, db, c2, "printing", 2000)

	statuses := []string{"printing", "completed"}

	counter.reset()
	stats, err := repo.GetBudgetStatsByCustomers(ctx, []uuid.UUID{c1, c2, c3}, statuses)
	require.NoError(t, err)
	assert.Equal(t, int64(1), counter.count(), "stats for a whole page must be a single query")

	assert.Equal(t, int64(2), stats[c1].Count)
	assert.Equal(t, int64(1000), stats[c1].Total) // only the completed budget counts
	assert.Equal(t, int64(1), stats[c2].Count)
	assert.Equal(t, int64(2000), stats[c2].Total)
	_, hasC3 := stats[c3]
	assert.False(t, hasC3, "customers with no budgets are absent from the map")

	// Adding more customers to the same call must not add queries.
	c4 := seedCustomer(t, repo, itOrgA, "C4", "c4@x.com")
	seedBudget(t, db, c4, "completed", 300)
	counter.reset()
	_, err = repo.GetBudgetStatsByCustomers(ctx, []uuid.UUID{c1, c2, c3, c4}, statuses)
	require.NoError(t, err)
	assert.Equal(t, int64(1), counter.count(), "query count stays constant as rows grow")
}
