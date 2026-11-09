package entities

import (
	"strconv"
	"time"
)

// NotificationType classifies a notification (drives the icon in clients).
type NotificationType string

// Known notification types.
const (
	TypeBudgetApproved NotificationType = "budget_approved"
	TypeBudgetRejected NotificationType = "budget_rejected"
	TypeBudgetExpired  NotificationType = "budget_expired"
	TypeLowStock       NotificationType = "low_stock"
	TypePaymentOverdue NotificationType = "payment_overdue"
	TypePaymentPaid    NotificationType = "payment_received"
)

// Notification is an in-app notification addressed to one user.
type Notification struct {
	ID             string           `json:"id"`
	OrganizationID string           `json:"-"`
	UserID         string           `json:"-"`
	Type           NotificationType `json:"type"`
	Title          string           `json:"title"`
	Body           string           `json:"body,omitempty"`
	// Link is an app route (e.g. "/budgets/<id>"), resolved by the clients.
	Link      string     `json:"link,omitempty"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// NewNotification is what producers send; it fans out to the org's users.
type NewNotification struct {
	Type  NotificationType
	Title string
	Body  string
	Link  string
	// DedupeKey, when set, skips the notification if one with the same key
	// was created for the organization within DedupeWindow.
	DedupeKey string
}

// UnreadCountResponse is the body of GET /notifications/unread-count.
type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

// LowStockNotification warns that a filament reached its low-stock threshold.
// It is deduplicated per filament (see DedupeKey) so repeated movements while
// the stock stays low don't spam the team.
func LowStockNotification(filamentID, name, color string, stockGrams int64) NewNotification {
	label := name
	if color != "" {
		label = name + " (" + color + ")"
	}
	return NewNotification{
		Type:      TypeLowStock,
		Title:     "Estoque baixo: " + label,
		Body:      "Restam " + strconv.FormatInt(stockGrams, 10) + " g. Hora de repor.",
		Link:      "/catalog/filaments?low_stock=1",
		DedupeKey: "low_stock:" + filamentID,
	}
}
