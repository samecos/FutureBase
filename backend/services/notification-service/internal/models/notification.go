package models

import (
	"time"

	"github.com/google/uuid"
)

// Notification represents a notification message
type Notification struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	TenantID    uuid.UUID         `json:"tenant_id" db:"tenant_id"`
	UserID      uuid.UUID         `json:"user_id" db:"user_id"`
	Type        NotificationType  `json:"type" db:"type"`
	Title       string            `json:"title" db:"title"`
	Content     string            `json:"content" db:"content"`
	Data        string            `json:"data" db:"data"` // JSON data
	Priority    Priority          `json:"priority" db:"priority"`
	Status      NotificationStatus `json:"status" db:"status"`
	Channels    []string          `json:"channels" db:"channels"`
	ReadAt      *time.Time        `json:"read_at,omitempty" db:"read_at"`
	ActionURL   string            `json:"action_url" db:"action_url"`
	ImageURL    string            `json:"image_url" db:"image_url"`
	SourceType  string            `json:"source_type" db:"source_type"`
	SourceID    string            `json:"source_id" db:"source_id"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty" db:"expires_at"`
}

type NotificationType string

const (
	NotificationTypeInfo      NotificationType = "info"
	NotificationTypeSuccess   NotificationType = "success"
	NotificationTypeWarning   NotificationType = "warning"
	NotificationTypeError     NotificationType = "error"
	NotificationTypeSystem    NotificationType = "system"
	NotificationTypeMention   NotificationType = "mention"
	NotificationTypeInvite    NotificationType = "invite"
	NotificationTypeUpdate    NotificationType = "update"
)

type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityNormal   Priority = "normal"
	PriorityHigh     Priority = "high"
	PriorityUrgent   Priority = "urgent"
)

type NotificationStatus string

const (
	NotificationStatusPending     NotificationStatus = "pending"
	NotificationStatusSent        NotificationStatus = "sent"
	NotificationStatusDelivered   NotificationStatus = "delivered"
	NotificationStatusRead        NotificationStatus = "read"
	NotificationStatusFailed      NotificationStatus = "failed"
)

// NotificationPreference represents user notification preferences
type NotificationPreference struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	TenantID  uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	
	// Channel preferences
	EmailEnabled      bool   `json:"email_enabled" db:"email_enabled"`
	PushEnabled       bool   `json:"push_enabled" db:"push_enabled"`
	InAppEnabled      bool   `json:"in_app_enabled" db:"in_app_enabled"`
	WebhookEnabled    bool   `json:"webhook_enabled" db:"webhook_enabled"`
	
	// Type preferences
	ProjectUpdates    bool   `json:"project_updates" db:"project_updates"`
	Mentions          bool   `json:"mentions" db:"mentions"`
	Invites           bool   `json:"invites" db:"invites"`
	SystemAlerts      bool   `json:"system_alerts" db:"system_alerts"`
	Marketing         bool   `json:"marketing" db:"marketing"`
	
	// Schedule
	QuietHoursStart   *int   `json:"quiet_hours_start,omitempty" db:"quiet_hours_start"`
	QuietHoursEnd     *int   `json:"quiet_hours_end,omitempty" db:"quiet_hours_end"`
	Timezone          string `json:"timezone" db:"timezone"`
	
	WebhookURL        string `json:"webhook_url" db:"webhook_url"`
	
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// NotificationTemplate represents a reusable notification template
type NotificationTemplate struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	TenantID    uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	Category    string     `json:"category" db:"category"`
	Subject     string     `json:"subject" db:"subject"`
	Body        string     `json:"body" db:"body"`
	DataSchema  string     `json:"data_schema" db:"data_schema"`
	Channels    []string   `json:"channels" db:"channels"`
	IsSystem    bool       `json:"is_system" db:"is_system"`
	CreatedBy   uuid.UUID  `json:"created_by" db:"created_by"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// NotificationBatch represents a batch of notifications
type NotificationBatch struct {
	ID            uuid.UUID     `json:"id" db:"id"`
	TenantID      uuid.UUID     `json:"tenant_id" db:"tenant_id"`
	Status        string        `json:"status" db:"status"`
	TotalCount    int           `json:"total_count" db:"total_count"`
	SuccessCount  int           `json:"success_count" db:"success_count"`
	FailedCount   int           `json:"failed_count" db:"failed_count"`
	CreatedAt     time.Time     `json:"created_at" db:"created_at"`
	CompletedAt   *time.Time    `json:"completed_at,omitempty" db:"completed_at"`
}

// WebhookDelivery represents a webhook delivery attempt
type WebhookDelivery struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	NotificationID uuid.UUID `json:"notification_id" db:"notification_id"`
	URL          string     `json:"url" db:"url"`
	Status       string     `json:"status" db:"status"`
	StatusCode   int        `json:"status_code" db:"status_code"`
	Response     string     `json:"response" db:"response"`
	Attempt      int        `json:"attempt" db:"attempt"`
	Error        string     `json:"error" db:"error"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// CreateNotificationRequest represents a request to create a notification
type CreateNotificationRequest struct {
	TenantID   uuid.UUID         `json:"tenant_id"`
	UserID     uuid.UUID         `json:"user_id"`
	Type       NotificationType  `json:"type"`
	Title      string            `json:"title"`
	Content    string            `json:"content"`
	Data       map[string]any    `json:"data,omitempty"`
	Priority   Priority          `json:"priority"`
	Channels   []string          `json:"channels"`
	ActionURL  string            `json:"action_url"`
	ImageURL   string            `json:"image_url"`
	SourceType string            `json:"source_type"`
	SourceID   string            `json:"source_id"`
}

// NotificationResponse represents a notification response
type NotificationResponse struct {
	ID          uuid.UUID         `json:"id"`
	Type        NotificationType  `json:"type"`
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	Data        map[string]any    `json:"data,omitempty"`
	Priority    Priority          `json:"priority"`
	Status      NotificationStatus `json:"status"`
	Read        bool              `json:"read"`
	ActionURL   string            `json:"action_url"`
	ImageURL    string            `json:"image_url"`
	CreatedAt   time.Time         `json:"created_at"`
}

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
