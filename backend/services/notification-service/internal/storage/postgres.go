package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/archplatform/notification-service/internal/models"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// PostgresStorage handles database operations
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage creates a new PostgreSQL storage
func NewPostgresStorage(connectionString string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresStorage{db: db}, nil
}

// Close closes the database connection
func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// CreateNotification creates a new notification
func (s *PostgresStorage) CreateNotification(ctx context.Context, n *models.Notification) error {
	query := `
		INSERT INTO notifications.notifications (id, tenant_id, user_id, type, title, content, data, priority, status,
			channels, action_url, image_url, source_type, source_id, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	channelsJSON, _ := json.Marshal(n.Channels)

	_, err := s.db.ExecContext(ctx, query,
		n.ID, n.TenantID, n.UserID, n.Type, n.Title, n.Content, n.Data, n.Priority, n.Status,
		string(channelsJSON), n.ActionURL, n.ImageURL, n.SourceType, n.SourceID, n.CreatedAt, n.ExpiresAt)

	return err
}

// GetNotification retrieves a notification by ID
func (s *PostgresStorage) GetNotification(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	query := `
		SELECT id, tenant_id, user_id, type, title, content, data, priority, status,
			channels, read_at, action_url, image_url, source_type, source_id, created_at, expires_at
		FROM notifications.notifications WHERE id = $1
	`

	var n models.Notification
	var channelsJSON string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&n.ID, &n.TenantID, &n.UserID, &n.Type, &n.Title, &n.Content, &n.Data, &n.Priority, &n.Status,
		&channelsJSON, &n.ReadAt, &n.ActionURL, &n.ImageURL, &n.SourceType, &n.SourceID, &n.CreatedAt, &n.ExpiresAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("notification not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(channelsJSON), &n.Channels)
	return &n, nil
}

// GetNotificationsByUser retrieves notifications for a user
func (s *PostgresStorage) GetNotificationsByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Notification, error) {
	query := `
		SELECT id, tenant_id, user_id, type, title, content, data, priority, status,
			channels, read_at, action_url, image_url, source_type, source_id, created_at
		FROM notifications.notifications WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`

	return s.queryNotifications(ctx, query, userID, limit, offset)
}

// GetUnreadNotificationsByUser retrieves unread notifications for a user
func (s *PostgresStorage) GetUnreadNotificationsByUser(ctx context.Context, userID uuid.UUID) ([]*models.Notification, error) {
	query := `
		SELECT id, tenant_id, user_id, type, title, content, data, priority, status,
			channels, read_at, action_url, image_url, source_type, source_id, created_at
		FROM notifications.notifications WHERE user_id = $1 AND read_at IS NULL
		ORDER BY created_at DESC
	`

	return s.queryNotifications(ctx, query, userID)
}

// MarkNotificationAsRead marks a notification as read
func (s *PostgresStorage) MarkNotificationAsRead(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE notifications.notifications SET read_at = $2, status = 'read' WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id, time.Now())
	return err
}

// MarkAllNotificationsAsRead marks all notifications for a user as read
func (s *PostgresStorage) MarkAllNotificationsAsRead(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE notifications.notifications 
		SET read_at = $2, status = 'read' 
		WHERE user_id = $1 AND read_at IS NULL
	`
	_, err := s.db.ExecContext(ctx, query, userID, time.Now())
	return err
}

// GetNotificationPreference retrieves notification preferences for a user
func (s *PostgresStorage) GetNotificationPreference(ctx context.Context, userID uuid.UUID) (*models.NotificationPreference, error) {
	query := `
		SELECT id, user_id, tenant_id, email_enabled, push_enabled, in_app_enabled, webhook_enabled,
			project_updates, mentions, invites, system_alerts, marketing,
			quiet_hours_start, quiet_hours_end, timezone, webhook_url, created_at, updated_at
		FROM notifications.preferences WHERE user_id = $1
	`

	var pref models.NotificationPreference
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&pref.ID, &pref.UserID, &pref.TenantID, &pref.EmailEnabled, &pref.PushEnabled,
		&pref.InAppEnabled, &pref.WebhookEnabled, &pref.ProjectUpdates, &pref.Mentions,
		&pref.Invites, &pref.SystemAlerts, &pref.Marketing,
		&pref.QuietHoursStart, &pref.QuietHoursEnd, &pref.Timezone, &pref.WebhookURL,
		&pref.CreatedAt, &pref.UpdatedAt)

	if err == sql.ErrNoRows {
		// Return default preferences
		return &models.NotificationPreference{
			UserID:         userID,
			EmailEnabled:   true,
			PushEnabled:    true,
			InAppEnabled:   true,
			ProjectUpdates: true,
			Mentions:       true,
			Invites:        true,
			SystemAlerts:   true,
			Marketing:      false,
		}, nil
	}

	return &pref, err
}

// SaveNotificationPreference saves notification preferences
func (s *PostgresStorage) SaveNotificationPreference(ctx context.Context, pref *models.NotificationPreference) error {
	query := `
		INSERT INTO notifications.preferences (id, user_id, tenant_id, email_enabled, push_enabled, in_app_enabled, webhook_enabled,
			project_updates, mentions, invites, system_alerts, marketing,
			quiet_hours_start, quiet_hours_end, timezone, webhook_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (user_id) DO UPDATE SET
			email_enabled = EXCLUDED.email_enabled,
			push_enabled = EXCLUDED.push_enabled,
			in_app_enabled = EXCLUDED.in_app_enabled,
			webhook_enabled = EXCLUDED.webhook_enabled,
			project_updates = EXCLUDED.project_updates,
			mentions = EXCLUDED.mentions,
			invites = EXCLUDED.invites,
			system_alerts = EXCLUDED.system_alerts,
			marketing = EXCLUDED.marketing,
			quiet_hours_start = EXCLUDED.quiet_hours_start,
			quiet_hours_end = EXCLUDED.quiet_hours_end,
			timezone = EXCLUDED.timezone,
			webhook_url = EXCLUDED.webhook_url,
			updated_at = EXCLUDED.updated_at
	`

	_, err := s.db.ExecContext(ctx, query,
		pref.ID, pref.UserID, pref.TenantID, pref.EmailEnabled, pref.PushEnabled,
		pref.InAppEnabled, pref.WebhookEnabled, pref.ProjectUpdates, pref.Mentions,
		pref.Invites, pref.SystemAlerts, pref.Marketing,
		pref.QuietHoursStart, pref.QuietHoursEnd, pref.Timezone, pref.WebhookURL,
		pref.CreatedAt, pref.UpdatedAt)

	return err
}

// CreateWebhookDelivery creates a webhook delivery record
func (s *PostgresStorage) CreateWebhookDelivery(ctx context.Context, delivery *models.WebhookDelivery) error {
	query := `
		INSERT INTO notifications.webhook_deliveries (id, notification_id, url, status, status_code, response, attempt, error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := s.db.ExecContext(ctx, query,
		delivery.ID, delivery.NotificationID, delivery.URL, delivery.Status,
		delivery.StatusCode, delivery.Response, delivery.Attempt, delivery.Error, delivery.CreatedAt)

	return err
}

// DeleteOldNotifications deletes notifications older than retention period
func (s *PostgresStorage) DeleteOldNotifications(ctx context.Context, retentionDays int) (int64, error) {
	query := `DELETE FROM notifications.notifications WHERE created_at < $1`
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	
	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}
	
	return result.RowsAffected()
}

// Helper methods
func (s *PostgresStorage) queryNotifications(ctx context.Context, query string, args ...any) ([]*models.Notification, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*models.Notification
	for rows.Next() {
		var n models.Notification
		var channelsJSON string

		err := rows.Scan(
			&n.ID, &n.TenantID, &n.UserID, &n.Type, &n.Title, &n.Content, &n.Data, &n.Priority, &n.Status,
			&channelsJSON, &n.ReadAt, &n.ActionURL, &n.ImageURL, &n.SourceType, &n.SourceID, &n.CreatedAt)

		if err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(channelsJSON), &n.Channels)
		notifications = append(notifications, &n)
	}

	return notifications, rows.Err()
}
