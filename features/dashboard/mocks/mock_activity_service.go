package mocks

import (
	"context"

	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
)

// MockActivityService is a mock implementation of IActivityService
type MockActivityService struct {
	mock.Mock
}

// NewMockActivityService creates a new mock activity service.
func NewMockActivityService() *MockActivityService {
	return &MockActivityService{}
}

// Record mocks the Record method.
func (m *MockActivityService) Record(ctx context.Context, activity activityEntities.ActivityEntity) {
	m.Called(ctx, activity)
}

// ListActivities mocks the ListActivities method.
func (m *MockActivityService) ListActivities(c *gin.Context) {
	m.Called(c)
}

// FindRecentByOrganization mocks the FindRecentByOrganization method.
func (m *MockActivityService) FindRecentByOrganization(organizationID string, limit int) ([]activityEntities.ActivityEntity, error) {
	args := m.Called(organizationID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]activityEntities.ActivityEntity), args.Error(1)
}
