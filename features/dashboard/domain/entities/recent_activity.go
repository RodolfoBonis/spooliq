package entities

import "time"

// RecentActivityItem represents a single activity item.
type RecentActivityItem struct {
	ID          string         `json:"id"`
	UserID      string         `json:"user_id"`
	Action      string         `json:"action"`
	EntityType  string         `json:"entity_type"`
	EntityID    string         `json:"entity_id"`
	EntityName  string         `json:"entity_name"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// RecentActivityResponse contains recent activity items.
type RecentActivityResponse struct {
	Activities []RecentActivityItem `json:"activities"`
	Total      int                  `json:"total"`
}
