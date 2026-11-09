package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PaymentWebhookEventModel is the durable log of every webhook payload received, so
// the reconcile job can reprocess events whose confirmation did not complete in the
// inline goroutine (process restart, transient error). organization_id is the path
// org; it is validated against the token before the event is trusted.
type PaymentWebhookEventModel struct {
	ID             uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID string          `gorm:"type:varchar(255);not null;index:idx_payment_webhook_events_org" json:"organization_id"`
	Payload        json.RawMessage `gorm:"type:jsonb" json:"payload"`
	ReceivedAt     time.Time       `gorm:"type:timestamptz;not null;index:idx_payment_webhook_events_received" json:"received_at"`
	ProcessedAt    *time.Time      `gorm:"type:timestamptz" json:"processed_at"`
	Attempts       int             `gorm:"type:integer;not null;default:0" json:"attempts"`
	LastError      string          `gorm:"type:text" json:"last_error"`
}

// TableName specifies the table name for GORM.
func (PaymentWebhookEventModel) TableName() string {
	return "payment_webhook_events"
}
