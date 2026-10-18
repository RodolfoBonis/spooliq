package repositories

import (
	"context"
	"time"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/notification/domain/entities"
)

// NotificationRepository persists notifications.
type NotificationRepository interface {
	// Recipients returns the Keycloak IDs of the organization's active users.
	Recipients(ctx context.Context, organizationID string) ([]string, error)
	// RecentlyNotified reports whether dedupeKey was used in the organization
	// since the given time.
	RecentlyNotified(ctx context.Context, organizationID, dedupeKey string, since time.Time) (bool, error)
	CreateForUsers(ctx context.Context, organizationID string, userIDs []string, n entities.NewNotification) error
	List(ctx context.Context, userID string, unreadOnly bool, q helpers.ListQuery) ([]entities.Notification, int64, error)
	UnreadCount(ctx context.Context, userID string) (int64, error)
	// MarkRead returns false when the notification doesn't belong to the user.
	MarkRead(ctx context.Context, userID, id string) (bool, error)
	MarkAllRead(ctx context.Context, userID string) error
}
