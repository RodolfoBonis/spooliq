package usecases

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	coreerrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/activity/domain/repositories"
	"github.com/gin-gonic/gin"
)

// IActivityService defines the interface for recording and querying activities.
type IActivityService interface {
	Record(ctx context.Context, activity entities.ActivityEntity)
	ListActivities(c *gin.Context)
	FindRecentByOrganization(organizationID string, limit int) ([]entities.ActivityEntity, error)
}

// ActivityService implements IActivityService with repository persistence.
type ActivityService struct {
	repository repositories.ActivityRepository
	logger     logger.Logger
}

// NewActivityService creates a new ActivityService instance.
func NewActivityService(repository repositories.ActivityRepository, logger logger.Logger) IActivityService {
	return &ActivityService{
		repository: repository,
		logger:     logger,
	}
}

// Record persists an activity asynchronously in a fire-and-forget goroutine.
func (s *ActivityService) Record(ctx context.Context, activity entities.ActivityEntity) {
	go func() {
		bgCtx := context.Background()

		if activity.Metadata != nil {
			if _, err := json.Marshal(activity.Metadata); err != nil {
				s.logger.Error(bgCtx, "failed to marshal activity metadata", logger.Fields{
					"error":       err.Error(),
					"entity_type": string(activity.EntityType),
					"entity_id":   activity.EntityID,
				})
				activity.Metadata = nil
			}
		}

		if err := s.repository.Create(&activity); err != nil {
			s.logger.Error(bgCtx, "failed to record activity", logger.Fields{
				"error":       err.Error(),
				"entity_type": string(activity.EntityType),
				"entity_id":   activity.EntityID,
				"action":      string(activity.Action),
			})
		}
	}()
}

// ListActivities handles HTTP GET requests to list activities for an organization.
// @Summary List activities
// @Tags Activities
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param entity_type query string false "Filter by entity type"
// @Param action query string false "Filter by action"
// @Success 200 {object} helpers.Page[entities.ActivityEntity]
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Security BearerAuth
// @Router /activities [get]
func (s *ActivityService) ListActivities(c *gin.Context) {
	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		coreerrors.Respond(c, coreerrors.Unauthorized("organization_id_missing", "Organização não encontrada no contexto"))
		return
	}

	q := helpers.ParseListQuery(c, helpers.ListQueryOptions{DefaultPageSize: 20})

	filter := &entities.ActivityFilter{
		OrganizationID: organizationID,
		Page:           q.Page,
		PageSize:       q.PageSize,
	}

	if et := c.Query("entity_type"); et != "" {
		entityType := entities.ActivityEntityType(et)
		filter.EntityType = &entityType
	}

	if a := c.Query("action"); a != "" {
		action := entities.ActivityAction(a)
		filter.Action = &action
	}

	result, err := s.repository.FindByOrganization(filter)
	if err != nil {
		s.logger.Error(c.Request.Context(), "failed to list activities", logger.Fields{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		coreerrors.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, helpers.NewPage(result.Activities, result.Total, q))
}

// FindRecentByOrganization returns the most recent activities for an organization.
func (s *ActivityService) FindRecentByOrganization(organizationID string, limit int) ([]entities.ActivityEntity, error) {
	return s.repository.FindRecentByOrganization(organizationID, limit)
}
