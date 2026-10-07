package entities

import (
	"testing"
	"time"
)

// TestIsValidTransition is a table test over the full Phase 4B transition matrix.
func TestIsValidTransition(t *testing.T) {
	allowed := map[BudgetStatus]map[BudgetStatus]bool{
		StatusDraft:     {StatusSent: true, StatusCancelled: true},
		StatusSent:      {StatusApproved: true, StatusRejected: true, StatusExpired: true, StatusCancelled: true},
		StatusApproved:  {StatusPrinting: true, StatusCancelled: true},
		StatusRejected:  {StatusDraft: true},
		StatusExpired:   {StatusDraft: true},
		StatusCancelled: {StatusDraft: true},
		StatusPrinting:  {StatusCompleted: true},
		StatusCompleted: {},
	}

	all := KnownStatuses()
	for _, from := range all {
		for _, to := range all {
			want := allowed[from][to]
			b := &BudgetEntity{Status: from}
			if got := b.IsValidTransition(to); got != want {
				t.Errorf("transition %s -> %s: got %v, want %v", from, to, got, want)
			}
		}
	}
}

func TestKnownStatusesIncludesExpiredAndCancelled(t *testing.T) {
	if !IsKnownStatus("expired") || !IsKnownStatus("cancelled") {
		t.Fatal("expired and cancelled must be known statuses")
	}
}

func TestExpiryHelpers(t *testing.T) {
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	sentPast := &BudgetEntity{Status: StatusSent, ValidUntil: &past}
	if !sentPast.IsExpired(now) {
		t.Error("sent budget with past valid_until should be expired")
	}
	if sentPast.EffectiveStatus(now) != StatusExpired {
		t.Error("effective status of overdue sent budget should be expired")
	}
	if sentPast.CanRespond(now) {
		t.Error("overdue sent budget should not be answerable")
	}

	sentFuture := &BudgetEntity{Status: StatusSent, ValidUntil: &future}
	if sentFuture.IsExpired(now) {
		t.Error("sent budget with future valid_until should not be expired")
	}
	if !sentFuture.CanRespond(now) {
		t.Error("non-expired sent budget should be answerable")
	}

	sentNoValidity := &BudgetEntity{Status: StatusSent}
	if sentNoValidity.IsExpired(now) {
		t.Error("sent budget with no valid_until should never be expired")
	}
	if !sentNoValidity.CanRespond(now) {
		t.Error("sent budget with no valid_until should be answerable")
	}

	draft := &BudgetEntity{Status: StatusDraft}
	if draft.CanRespond(now) {
		t.Error("draft budget should not be answerable")
	}
}
