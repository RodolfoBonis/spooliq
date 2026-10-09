package models

import (
	"time"

	"github.com/google/uuid"
)

// DashboardGoalModel stores a user-defined monthly target per organization and
// metric (see entities.GoalMetric).
type DashboardGoalModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_dashboard_goals_org_metric,priority:1"`
	Metric         string    `gorm:"type:varchar(30);not null;uniqueIndex:idx_dashboard_goals_org_metric,priority:2"`
	Period         string    `gorm:"type:varchar(20);not null;default:'monthly'"`
	Target         float64   `gorm:"type:numeric(16,2);not null"`
	UpdatedBy      string    `gorm:"type:varchar(255)"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name.
func (DashboardGoalModel) TableName() string { return "dashboard_goals" }
