package usecases

import (
	"context"
	"testing"
	"time"

	otellogger "github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/notification/domain/entities"
	"github.com/stretchr/testify/assert"
)

type noopLogger struct{}

func (noopLogger) Debug(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Info(context.Context, string, ...otellogger.Fields)    {}
func (noopLogger) Warning(context.Context, string, ...otellogger.Fields) {}
func (noopLogger) Error(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Fatal(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Panic(context.Context, string, ...otellogger.Fields)   {}
func (n noopLogger) With(otellogger.Fields) otellogger.Logger            { return n }
func (noopLogger) LogError(context.Context, string, error)               {}

type fakeRepo struct {
	recipients []string
	recentKeys map[string]time.Time
	created    map[string][]string // title -> users
	since      time.Time
}

func (r *fakeRepo) Recipients(context.Context, string) ([]string, error) { return r.recipients, nil }

func (r *fakeRepo) RecentlyNotified(_ context.Context, _ string, key string, since time.Time) (bool, error) {
	r.since = since
	at, ok := r.recentKeys[key]
	return ok && !at.Before(since), nil
}

func (r *fakeRepo) CreateForUsers(_ context.Context, _ string, users []string, n entities.NewNotification) error {
	r.created[n.Title] = users
	if n.DedupeKey != "" {
		r.recentKeys[n.DedupeKey] = time.Now()
	}
	return nil
}

func (r *fakeRepo) List(context.Context, string, bool, helpers.ListQuery) ([]entities.Notification, int64, error) {
	return nil, 0, nil
}
func (r *fakeRepo) UnreadCount(context.Context, string) (int64, error)     { return 0, nil }
func (r *fakeRepo) MarkRead(context.Context, string, string) (bool, error) { return true, nil }
func (r *fakeRepo) MarkAllRead(context.Context, string) error              { return nil }

func newService(repo *fakeRepo) *NotificationService {
	s := NewNotificationService(repo, noopLogger{}).(*NotificationService)
	s.async = func(f func()) { f() } // synchronous for the test
	return s
}

func TestNotify_FansOutToEveryActiveUser(t *testing.T) {
	repo := &fakeRepo{recipients: []string{"u1", "u2"}, recentKeys: map[string]time.Time{}, created: map[string][]string{}}
	newService(repo).Notify("org", entities.NewNotification{Type: entities.TypeBudgetApproved, Title: "Aprovado"})

	assert.Equal(t, []string{"u1", "u2"}, repo.created["Aprovado"])
}

func TestNotify_DeduplicatesWithinWindow(t *testing.T) {
	repo := &fakeRepo{recipients: []string{"u1"}, recentKeys: map[string]time.Time{}, created: map[string][]string{}}
	s := newService(repo)
	low := entities.LowStockNotification("f1", "PLA", "Preto", 120)

	s.Notify("org", low)
	delete(repo.created, low.Title)
	s.Notify("org", low)

	assert.Empty(t, repo.created, "second low-stock warning is suppressed")
	assert.WithinDuration(t, time.Now().Add(-DedupeWindow), repo.since, time.Minute)
}

func TestLowStockNotification(t *testing.T) {
	n := entities.LowStockNotification("f1", "PLA Basic", "Preto", 120)
	assert.Equal(t, "Estoque baixo: PLA Basic (Preto)", n.Title)
	assert.Equal(t, "low_stock:f1", n.DedupeKey)
	assert.Contains(t, n.Body, "120 g")
}
