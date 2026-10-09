package repositories

import (
	"context"
	"time"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/notification/data/models"
	"github.com/RodolfoBonis/spooliq/features/notification/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/notification/domain/repositories"
	"gorm.io/gorm"
)

type notificationRepository struct {
	db *gorm.DB
}

// NewNotificationRepository creates the GORM notification repository.
func NewNotificationRepository(db *gorm.DB) repositories.NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Recipients(ctx context.Context, organizationID string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).
		Table("users").
		Where("organization_id = ? AND is_active = ? AND deleted_at IS NULL", organizationID, true).
		Pluck("keycloak_user_id", &ids).Error
	return ids, err
}

func (r *notificationRepository) RecentlyNotified(ctx context.Context, organizationID, dedupeKey string, since time.Time) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.NotificationModel{}).
		Where("organization_id = ? AND dedupe_key = ? AND created_at >= ?", organizationID, dedupeKey, since).
		Limit(1).
		Count(&count).Error
	return count > 0, err
}

func (r *notificationRepository) CreateForUsers(ctx context.Context, organizationID string, userIDs []string, n entities.NewNotification) error {
	if len(userIDs) == 0 {
		return nil
	}
	rows := make([]models.NotificationModel, 0, len(userIDs))
	for _, userID := range userIDs {
		rows = append(rows, models.NotificationModel{
			OrganizationID: organizationID,
			UserID:         userID,
			Type:           string(n.Type),
			Title:          n.Title,
			Body:           n.Body,
			Link:           n.Link,
			DedupeKey:      n.DedupeKey,
		})
	}
	return r.db.WithContext(ctx).Create(&rows).Error
}

func (r *notificationRepository) List(ctx context.Context, userID string, unreadOnly bool, q helpers.ListQuery) ([]entities.Notification, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.NotificationModel{}).Where("user_id = ?", userID)
	if unreadOnly {
		query = query.Where("read_at IS NULL")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []models.NotificationModel
	if err := query.Order("created_at DESC").Offset(q.Offset()).Limit(q.Limit()).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]entities.Notification, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToEntity())
	}
	return out, total, nil
}

func (r *notificationRepository) UnreadCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.NotificationModel{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Count(&count).Error
	return count, err
}

func (r *notificationRepository) MarkRead(ctx context.Context, userID, id string) (bool, error) {
	res := r.db.WithContext(ctx).
		Model(&models.NotificationModel{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("read_at", gorm.Expr("COALESCE(read_at, NOW())"))
	return res.RowsAffected > 0, res.Error
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Model(&models.NotificationModel{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", time.Now()).Error
}
