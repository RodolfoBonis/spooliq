package models

import (
	"time"

	companyModels "github.com/RodolfoBonis/spooliq/features/company/data/models"
	"github.com/RodolfoBonis/spooliq/features/notification/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationModel is the database model for in-app notifications. One row
// per recipient user, so read state is simply read_at.
type NotificationModel struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID string     `gorm:"type:varchar(255);not null;index:idx_notifications_org_dedupe,priority:1"`
	UserID         string     `gorm:"type:varchar(255);not null;index:idx_notifications_user,priority:1"`
	Type           string     `gorm:"type:varchar(40);not null"`
	Title          string     `gorm:"type:varchar(255);not null"`
	Body           string     `gorm:"type:text"`
	Link           string     `gorm:"type:varchar(255)"`
	DedupeKey      string     `gorm:"type:varchar(255);index:idx_notifications_org_dedupe,priority:2"`
	ReadAt         *time.Time `gorm:"index:idx_notifications_user,priority:2"`
	CreatedAt      time.Time  `gorm:"autoCreateTime;index:idx_notifications_user,priority:3,sort:desc"`

	Organization *companyModels.CompanyModel `gorm:"foreignKey:OrganizationID;references:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

// TableName returns the table name.
func (NotificationModel) TableName() string { return "notifications" }

// BeforeCreate assigns an ID when missing.
func (n *NotificationModel) BeforeCreate(*gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}

// ToEntity converts the model to the domain entity.
func (n *NotificationModel) ToEntity() entities.Notification {
	return entities.Notification{
		ID:             n.ID.String(),
		OrganizationID: n.OrganizationID,
		UserID:         n.UserID,
		Type:           entities.NotificationType(n.Type),
		Title:          n.Title,
		Body:           n.Body,
		Link:           n.Link,
		ReadAt:         n.ReadAt,
		CreatedAt:      n.CreatedAt,
	}
}
