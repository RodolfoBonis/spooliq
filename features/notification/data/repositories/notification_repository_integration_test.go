package repositories_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/notification/data/models"
	"github.com/RodolfoBonis/spooliq/features/notification/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/notification/domain/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// openDB connects to TEST_DATABASE_URL in a throwaway schema (skips when unset):
//
//	TEST_DATABASE_URL='postgres://user:pass@localhost:5432/db?sslmode=disable' go test ./features/notification/...
func openDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping notification integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	schema := fmt.Sprintf("notif_%s", uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)
	t.Cleanup(func() {
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})

	require.NoError(t, db.Exec(`CREATE TABLE users (
		id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
		organization_id varchar(255) NOT NULL,
		keycloak_user_id varchar(255) NOT NULL,
		is_active boolean NOT NULL DEFAULT true,
		deleted_at timestamptz)`).Error)
	require.NoError(t, db.AutoMigrate(&models.NotificationModel{}))
	return db
}

func TestNotificationRepository(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()
	repo := repositories.NewNotificationRepository(db)

	require.NoError(t, db.Exec(`INSERT INTO users (organization_id, keycloak_user_id, is_active, deleted_at) VALUES
		('org', 'u1', true, NULL), ('org', 'u2', true, NULL),
		('org', 'off', false, NULL), ('org', 'gone', true, now()), ('other', 'x', true, NULL)`).Error)

	recipients, err := repo.Recipients(ctx, "org")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"u1", "u2"}, recipients)

	n := entities.NewNotification{Type: entities.TypeLowStock, Title: "Estoque baixo", DedupeKey: "low_stock:f1"}
	require.NoError(t, repo.CreateForUsers(ctx, "org", recipients, n))
	require.NoError(t, repo.CreateForUsers(ctx, "org", []string{"u1"}, entities.NewNotification{Type: entities.TypeBudgetApproved, Title: "Aprovado"}))

	recent, err := repo.RecentlyNotified(ctx, "org", "low_stock:f1", time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.True(t, recent)
	recent, err = repo.RecentlyNotified(ctx, "other", "low_stock:f1", time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.False(t, recent)

	q := helpers.ListQuery{Page: 1, PageSize: 20}
	items, total, err := repo.List(ctx, "u1", false, q)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Equal(t, "Aprovado", items[0].Title, "newest first")

	count, err := repo.UnreadCount(ctx, "u1")
	require.NoError(t, err)
	require.EqualValues(t, 2, count)

	found, err := repo.MarkRead(ctx, "u2", items[0].ID)
	require.NoError(t, err)
	require.False(t, found, "users can't mark other users' notifications")

	found, err = repo.MarkRead(ctx, "u1", items[0].ID)
	require.NoError(t, err)
	require.True(t, found)
	unread, _, err := repo.List(ctx, "u1", true, q)
	require.NoError(t, err)
	require.Len(t, unread, 1)

	require.NoError(t, repo.MarkAllRead(ctx, "u1"))
	count, err = repo.UnreadCount(ctx, "u1")
	require.NoError(t, err)
	require.Zero(t, count)

	count, err = repo.UnreadCount(ctx, "u2")
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
}
