package repositories_test

import (
	"context"
	"sync"
	"testing"
	"time"

	repoimpl "github.com/RodolfoBonis/spooliq/features/budget/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// createPhase4BSchema creates the minimal companies + budgets tables (with the
// Phase 4B columns and the partial unique indexes) the integration tests below
// exercise. It is self-contained (no FKs).
func createPhase4BSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE companies (
			organization_id varchar(255) PRIMARY KEY,
			next_quote_number integer NOT NULL DEFAULT 1,
			deleted_at timestamptz
		)`,
		`CREATE TABLE budgets (
			id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			organization_id varchar(255) NOT NULL,
			name varchar(255) NOT NULL DEFAULT '',
			quote_number integer,
			public_token varchar(43),
			status varchar(20) NOT NULL DEFAULT 'draft',
			valid_until timestamptz,
			customer_response_at timestamptz,
			customer_response_name varchar(120),
			customer_response_ip varchar(45),
			customer_response_user_agent varchar(255),
			rejection_reason text,
			approved_at timestamptz,
			completed_at timestamptz,
			created_at timestamptz NOT NULL DEFAULT now(),
			updated_at timestamptz NOT NULL DEFAULT now(),
			deleted_at timestamptz
		)`,
		`CREATE TABLE budget_status_history (
			id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			budget_id uuid NOT NULL,
			organization_id varchar(255) NOT NULL,
			previous_status varchar(20) NOT NULL,
			new_status varchar(20) NOT NULL,
			changed_by varchar(255) NOT NULL,
			notes text,
			created_at timestamptz NOT NULL DEFAULT now()
		)`,
		`CREATE UNIQUE INDEX uq_budgets_org_quote_number ON budgets(organization_id, quote_number) WHERE deleted_at IS NULL AND quote_number IS NOT NULL`,
		`CREATE UNIQUE INDEX uq_budgets_public_token ON budgets(public_token) WHERE public_token IS NOT NULL`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error, s)
	}
}

// TestAllocateQuoteNumber_ConcurrentSequentialPerOrg proves the atomic allocation is
// race-safe (distinct, contiguous numbers under concurrency) and independent per org.
func TestAllocateQuoteNumber_ConcurrentSequentialPerOrg(t *testing.T) {
	db := openTestDB(t, "qalloc")
	createPhase4BSchema(t, db)
	require.NoError(t, db.Exec(`INSERT INTO companies (organization_id) VALUES ('org-a'), ('org-b')`).Error)

	repo := repoimpl.NewBudgetRepository(db)
	ctx := context.Background()

	const n = 50
	results := make([]int, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			v, err := repo.AllocateQuoteNumber(ctx, "org-a")
			results[idx] = v
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		require.NoErrorf(t, err, "allocation %d", i)
	}

	seen := make(map[int]bool, n)
	for _, v := range results {
		require.Falsef(t, seen[v], "duplicate quote number allocated: %d", v)
		seen[v] = true
	}
	for i := 1; i <= n; i++ {
		require.Truef(t, seen[i], "missing quote number %d in 1..%d", i, n)
	}

	// org-b is an independent sequence starting at 1.
	v, err := repo.AllocateQuoteNumber(ctx, "org-b")
	require.NoError(t, err)
	require.Equal(t, 1, v)

	// The counter persists: org-a's next allocation is n+1.
	next, err := repo.AllocateQuoteNumber(ctx, "org-a")
	require.NoError(t, err)
	require.Equal(t, n+1, next)
}

// TestBackfillQuoteNumber_SQL exercises the one-time backfill + counter-init SQL.
func TestBackfillQuoteNumber_SQL(t *testing.T) {
	db := openTestDB(t, "qbackfill")
	createPhase4BSchema(t, db)
	require.NoError(t, db.Exec(`INSERT INTO companies (organization_id) VALUES ('org-a'), ('org-b')`).Error)

	base := time.Now().Add(-time.Hour)
	insert := func(org string, offset time.Duration) {
		require.NoError(t, db.Exec(
			`INSERT INTO budgets (id, organization_id, status, created_at) VALUES (?, ?, 'draft', ?)`,
			uuid.New(), org, base.Add(offset),
		).Error)
	}
	// org-a: three budgets in a known created_at order.
	insert("org-a", 0)
	insert("org-a", time.Minute)
	insert("org-a", 2*time.Minute)
	// org-b: two budgets.
	insert("org-b", 0)
	insert("org-b", time.Minute)

	backfill := `UPDATE budgets b SET quote_number = n.rn
		 FROM (
			SELECT id, ROW_NUMBER() OVER (PARTITION BY organization_id ORDER BY created_at, id) AS rn
			FROM budgets
			WHERE quote_number IS NULL
		 ) n
		 WHERE b.id = n.id AND b.quote_number IS NULL`
	require.NoError(t, db.Exec(backfill).Error)

	initCounter := `UPDATE companies c SET next_quote_number = sub.max_qn + 1
		 FROM (
			SELECT organization_id, MAX(quote_number) AS max_qn
			FROM budgets WHERE quote_number IS NOT NULL GROUP BY organization_id
		 ) sub
		 WHERE c.organization_id = sub.organization_id AND sub.max_qn >= c.next_quote_number`
	require.NoError(t, db.Exec(initCounter).Error)

	// org-a numbered 1,2,3 in created_at order.
	var nums []int
	require.NoError(t, db.Raw(`SELECT quote_number FROM budgets WHERE organization_id = 'org-a' ORDER BY created_at`).Scan(&nums).Error)
	require.Equal(t, []int{1, 2, 3}, nums)

	// counters set to max+1.
	var nextA, nextB int
	require.NoError(t, db.Raw(`SELECT next_quote_number FROM companies WHERE organization_id = 'org-a'`).Scan(&nextA).Error)
	require.NoError(t, db.Raw(`SELECT next_quote_number FROM companies WHERE organization_id = 'org-b'`).Scan(&nextB).Error)
	require.Equal(t, 4, nextA)
	require.Equal(t, 3, nextB)
}

// TestExpireOverdue_SQL verifies only overdue sent budgets are expired.
func TestExpireOverdue_SQL(t *testing.T) {
	db := openTestDB(t, "qexpire")
	createPhase4BSchema(t, db)
	repo := repoimpl.NewBudgetRepository(db)

	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	overdue := uuid.New()
	notDue := uuid.New()
	noValidity := uuid.New()
	approvedOverdue := uuid.New()

	require.NoError(t, db.Exec(`INSERT INTO budgets (id, organization_id, status, valid_until) VALUES
		(?, 'org-a', 'sent', ?),
		(?, 'org-a', 'sent', ?),
		(?, 'org-a', 'sent', NULL),
		(?, 'org-a', 'approved', ?)`,
		overdue, past, notDue, future, noValidity, approvedOverdue, past).Error)

	expired, err := repo.ExpireOverdue(context.Background())
	require.NoError(t, err)
	require.Len(t, expired, 1)
	require.Equal(t, overdue, expired[0].ID)
	require.Equal(t, "org-a", expired[0].OrganizationID)

	status := func(id uuid.UUID) string {
		var s string
		require.NoError(t, db.Raw(`SELECT status FROM budgets WHERE id = ?`, id).Scan(&s).Error)
		return s
	}
	require.Equal(t, "expired", status(overdue))

	// The expiration is recorded in the status history (funnel/response times).
	var history []struct{ PreviousStatus, NewStatus, ChangedBy string }
	require.NoError(t, db.Raw(`SELECT previous_status, new_status, changed_by FROM budget_status_history WHERE budget_id = ?`, overdue).Scan(&history).Error)
	require.Len(t, history, 1)
	require.Equal(t, "sent", history[0].PreviousStatus)
	require.Equal(t, "expired", history[0].NewStatus)
	require.Equal(t, "system", history[0].ChangedBy)
	require.Equal(t, "sent", status(notDue))
	require.Equal(t, "sent", status(noValidity))
	require.Equal(t, "approved", status(approvedOverdue))
}

// TestRespondToPublicBudget_Race proves two concurrent approves yield exactly one
// success (the race-safe conditional UPDATE).
func TestRespondToPublicBudget_Race(t *testing.T) {
	db := openTestDB(t, "qrace")
	createPhase4BSchema(t, db)
	repo := repoimpl.NewBudgetRepository(db)

	id := uuid.New()
	future := time.Now().Add(24 * time.Hour)
	require.NoError(t, db.Exec(
		`INSERT INTO budgets (id, organization_id, status, valid_until) VALUES (?, 'org-a', 'sent', ?)`,
		id, future,
	).Error)

	const racers = 8
	rowsOut := make([]int64, racers)
	errsOut := make([]error, racers)
	var wg sync.WaitGroup
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			rows, err := repo.RespondToPublicBudget(context.Background(), id, entities.StatusApproved, "Alice", "1.2.3.4", "ua", nil, time.Now())
			rowsOut[idx] = rows
			errsOut[idx] = err
		}(i)
	}
	wg.Wait()

	successes := 0
	for i := 0; i < racers; i++ {
		require.NoErrorf(t, errsOut[i], "racer %d", i)
		if rowsOut[i] == 1 {
			successes++
		}
	}
	require.Equal(t, 1, successes, "exactly one concurrent approve must win")

	var status string
	require.NoError(t, db.Raw(`SELECT status FROM budgets WHERE id = ?`, id).Scan(&status).Error)
	require.Equal(t, "approved", status)
}

// TestUpdateStatus_LifecycleDates checks approved_at/completed_at are stamped on the
// matching transitions and cleared when the budget is reopened as a draft.
func TestUpdateStatus_LifecycleDates(t *testing.T) {
	db := openTestDB(t, "qlifecycle")
	createPhase4BSchema(t, db)
	repo := repoimpl.NewBudgetRepository(db)
	ctx := context.Background()

	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO budgets (id, organization_id, status) VALUES (?, 'org-a', 'sent')`, id).Error)

	dates := func() (approved, completed *time.Time) {
		var row struct{ ApprovedAt, CompletedAt *time.Time }
		require.NoError(t, db.Raw(`SELECT approved_at, completed_at FROM budgets WHERE id = ?`, id).Scan(&row).Error)
		return row.ApprovedAt, row.CompletedAt
	}

	require.NoError(t, repo.UpdateStatus(ctx, id, "org-a", entities.StatusSent, entities.StatusApproved))
	approved, completed := dates()
	require.NotNil(t, approved)
	require.Nil(t, completed)

	require.NoError(t, repo.UpdateStatus(ctx, id, "org-a", entities.StatusApproved, entities.StatusPrinting))
	require.NoError(t, repo.UpdateStatus(ctx, id, "org-a", entities.StatusPrinting, entities.StatusCompleted))
	approvedAfter, completed := dates()
	require.Equal(t, approved.Unix(), approvedAfter.Unix(), "approval date is kept through production")
	require.NotNil(t, completed)

	// Reopening (cancelled -> draft) starts a new sale: dates are cleared.
	require.NoError(t, db.Exec(`UPDATE budgets SET status = 'cancelled' WHERE id = ?`, id).Error)
	require.NoError(t, repo.UpdateStatus(ctx, id, "org-a", entities.StatusCancelled, entities.StatusDraft))
	approved, completed = dates()
	require.Nil(t, approved)
	require.Nil(t, completed)
}

// TestRespondToPublicBudget_ApprovalDate checks a customer approval stamps approved_at.
func TestRespondToPublicBudget_ApprovalDate(t *testing.T) {
	db := openTestDB(t, "qpublicdate")
	createPhase4BSchema(t, db)
	repo := repoimpl.NewBudgetRepository(db)

	approved, rejected := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO budgets (id, organization_id, status) VALUES (?, 'org-a', 'sent'), (?, 'org-a', 'sent')`, approved, rejected).Error)
	at := time.Now().Truncate(time.Second)

	_, err := repo.RespondToPublicBudget(context.Background(), approved, entities.StatusApproved, "Ana", "1.1.1.1", "ua", nil, at)
	require.NoError(t, err)
	reason := "caro"
	_, err = repo.RespondToPublicBudget(context.Background(), rejected, entities.StatusRejected, "Ana", "1.1.1.1", "ua", &reason, at)
	require.NoError(t, err)

	approvedAt := func(id uuid.UUID) *time.Time {
		var row struct{ ApprovedAt *time.Time }
		require.NoError(t, db.Raw(`SELECT approved_at FROM budgets WHERE id = ?`, id).Scan(&row).Error)
		return row.ApprovedAt
	}
	require.NotNil(t, approvedAt(approved))
	require.Equal(t, at.Unix(), approvedAt(approved).Unix())
	require.Nil(t, approvedAt(rejected))
}
