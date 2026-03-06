package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/archplatform/analytics-service/internal/models"
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

// CreateEvent creates a new analytics event
func (s *PostgresStorage) CreateEvent(ctx context.Context, event *models.Event) error {
	query := `
		INSERT INTO analytics.events (id, tenant_id, user_id, project_id, event_type, entity_type, entity_id, action, 
			metadata, session_id, ip, user_agent, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	metadataJSON, _ := json.Marshal(event.Metadata)

	_, err := s.db.ExecContext(ctx, query,
		event.ID, event.TenantID, event.UserID, event.ProjectID, event.EventType,
		event.EntityType, event.EntityID, event.Action, string(metadataJSON),
		event.SessionID, event.IP, event.UserAgent, event.Timestamp)

	return err
}

// GetEvents retrieves events for a tenant
func (s *PostgresStorage) GetEvents(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) ([]*models.Event, error) {
	query := `
		SELECT id, tenant_id, user_id, project_id, event_type, entity_type, entity_id, action, 
			metadata, session_id, ip, user_agent, timestamp
		FROM analytics.events 
		WHERE tenant_id = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp DESC
	`

	return s.queryEvents(ctx, query, tenantID, startDate, endDate)
}

// CreateReport creates a new report
func (s *PostgresStorage) CreateReport(ctx context.Context, report *models.Report) error {
	query := `
		INSERT INTO analytics.reports (id, tenant_id, name, type, status, parameters, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	paramsJSON, _ := json.Marshal(report.Parameters)

	_, err := s.db.ExecContext(ctx, query,
		report.ID, report.TenantID, report.Name, report.Type, report.Status,
		string(paramsJSON), report.CreatedBy, report.CreatedAt)

	return err
}

// GetReport retrieves a report by ID
func (s *PostgresStorage) GetReport(ctx context.Context, id uuid.UUID) (*models.Report, error) {
	query := `
		SELECT id, tenant_id, name, type, status, parameters, result, created_by, created_at, completed_at
		FROM analytics.reports WHERE id = $1
	`

	var report models.Report
	var paramsJSON string
	var resultJSON *string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&report.ID, &report.TenantID, &report.Name, &report.Type, &report.Status,
		&paramsJSON, &resultJSON, &report.CreatedBy, &report.CreatedAt, &report.CompletedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("report not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(paramsJSON), &report.Parameters)
	if resultJSON != nil {
		json.Unmarshal([]byte(*resultJSON), &report.Result)
	}

	return &report, nil
}

// UpdateReport updates a report
func (s *PostgresStorage) UpdateReport(ctx context.Context, report *models.Report) error {
	query := `
		UPDATE analytics.reports SET
			status = $2, result = $3, completed_at = $4
		WHERE id = $1
	`

	resultJSON, _ := json.Marshal(report.Result)

	_, err := s.db.ExecContext(ctx, query,
		report.ID, report.Status, string(resultJSON), report.CompletedAt)

	return err
}

// GetReportsByTenant retrieves reports for a tenant
func (s *PostgresStorage) GetReportsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.Report, error) {
	query := `
		SELECT id, tenant_id, name, type, status, parameters, created_by, created_at, completed_at
		FROM analytics.reports WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []*models.Report
	for rows.Next() {
		var report models.Report
		var paramsJSON string

		err := rows.Scan(
			&report.ID, &report.TenantID, &report.Name, &report.Type, &report.Status,
			&paramsJSON, &report.CreatedBy, &report.CreatedAt, &report.CompletedAt)

		if err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(paramsJSON), &report.Parameters)
		reports = append(reports, &report)
	}

	return reports, rows.Err()
}

// Helper methods
func (s *PostgresStorage) queryEvents(ctx context.Context, query string, args ...any) ([]*models.Event, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.Event
	for rows.Next() {
		var event models.Event
		var metadataJSON string

		err := rows.Scan(
			&event.ID, &event.TenantID, &event.UserID, &event.ProjectID, &event.EventType,
			&event.EntityType, &event.EntityID, &event.Action, &metadataJSON,
			&event.SessionID, &event.IP, &event.UserAgent, &event.Timestamp)

		if err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(metadataJSON), &event.Metadata)
		events = append(events, &event)
	}

	return events, rows.Err()
}
