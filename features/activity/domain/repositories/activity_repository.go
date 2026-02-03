package repositories

import (
	"github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
)

// ActivityRepository defines the interface for activity data operations.
type ActivityRepository interface {
	Create(activity *entities.ActivityEntity) error
	FindByOrganization(filter *entities.ActivityFilter) (*entities.PaginatedActivities, error)
	FindRecentByOrganization(organizationID string, limit int) ([]entities.ActivityEntity, error)
}
