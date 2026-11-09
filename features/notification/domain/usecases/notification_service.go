package usecases

import (
	"context"
	"net/http"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	coreerrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/notification/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/notification/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DedupeWindow is how long a DedupeKey suppresses repeated notifications.
const DedupeWindow = 7 * 24 * time.Hour

// INotificationService creates and serves in-app notifications.
type INotificationService interface {
	// Notify fans a notification out to every active user of the organization.
	// It runs in the background and never fails the caller.
	Notify(organizationID string, n entities.NewNotification)
	List(c *gin.Context)
	UnreadCount(c *gin.Context)
	MarkRead(c *gin.Context)
	MarkAllRead(c *gin.Context)
}

// NotificationService implements INotificationService.
type NotificationService struct {
	repository repositories.NotificationRepository
	logger     logger.Logger
	now        func() time.Time
	// async runs Notify's work; tests replace it to run synchronously.
	async func(func())
}

// NewNotificationService creates the notification service.
func NewNotificationService(repository repositories.NotificationRepository, log logger.Logger) INotificationService {
	return &NotificationService{
		repository: repository,
		logger:     log,
		now:        time.Now,
		async:      func(f func()) { go f() },
	}
}

// Notify fans out asynchronously so producers (HTTP handlers, jobs) are
// never slowed down or failed by notifications.
func (s *NotificationService) Notify(organizationID string, n entities.NewNotification) {
	s.async(func() { s.deliver(context.Background(), organizationID, n) })
}

func (s *NotificationService) deliver(ctx context.Context, organizationID string, n entities.NewNotification) {
	fields := logger.Fields{"organization_id": organizationID, "type": string(n.Type)}
	if n.DedupeKey != "" {
		recent, err := s.repository.RecentlyNotified(ctx, organizationID, n.DedupeKey, s.now().Add(-DedupeWindow))
		if err != nil {
			s.logger.Error(ctx, "notification dedupe check failed", logger.Fields{"error": err.Error(), "type": string(n.Type)})
			return
		}
		if recent {
			return
		}
	}
	recipients, err := s.repository.Recipients(ctx, organizationID)
	if err != nil {
		fields["error"] = err.Error()
		s.logger.Error(ctx, "failed to load notification recipients", fields)
		return
	}
	if err := s.repository.CreateForUsers(ctx, organizationID, recipients, n); err != nil {
		fields["error"] = err.Error()
		s.logger.Error(ctx, "failed to create notifications", fields)
	}
}

// List returns the user's notifications, newest first.
// @Summary List notifications
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param unread query bool false "Only unread"
// @Success 200 {object} helpers.Page[entities.Notification]
// @Failure 401 {object} errors.HTTPError
// @Router /notifications [get]
func (s *NotificationService) List(c *gin.Context) {
	userID := helpers.GetUserID(c)
	q := helpers.ParseListQuery(c, helpers.ListQueryOptions{DefaultPageSize: 20})
	items, total, err := s.repository.List(c.Request.Context(), userID, c.Query("unread") == "true", q)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, helpers.NewPage(items, total, q))
}

// UnreadCount returns how many notifications the user hasn't read.
// @Summary Unread notifications count
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.UnreadCountResponse
// @Failure 401 {object} errors.HTTPError
// @Router /notifications/unread-count [get]
func (s *NotificationService) UnreadCount(c *gin.Context) {
	count, err := s.repository.UnreadCount(c.Request.Context(), helpers.GetUserID(c))
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, entities.UnreadCountResponse{Count: count})
}

// MarkRead marks one notification as read.
// @Summary Mark notification as read
// @Tags notifications
// @Security BearerAuth
// @Param id path string true "Notification ID"
// @Success 204 "Marked as read"
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Router /notifications/{id}/read [post]
func (s *NotificationService) MarkRead(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		coreerrors.Respond(c, coreerrors.BadRequest("invalid_notification_id", "ID de notificação inválido"))
		return
	}
	found, err := s.repository.MarkRead(c.Request.Context(), helpers.GetUserID(c), id)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}
	if !found {
		coreerrors.Respond(c, coreerrors.NotFoundErr("notification_not_found", "Notificação não encontrada"))
		return
	}
	c.Status(http.StatusNoContent)
}

// MarkAllRead marks every notification of the user as read.
// @Summary Mark all notifications as read
// @Tags notifications
// @Security BearerAuth
// @Success 204 "Marked as read"
// @Router /notifications/read-all [post]
func (s *NotificationService) MarkAllRead(c *gin.Context) {
	if err := s.repository.MarkAllRead(c.Request.Context(), helpers.GetUserID(c)); err != nil {
		coreerrors.Respond(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
