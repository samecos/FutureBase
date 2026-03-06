package models

import (
	"time"

	"github.com/google/uuid"
)

// Event represents an analytics event
type Event struct {
	ID          uuid.UUID         `json:"id"`
	TenantID    uuid.UUID         `json:"tenant_id"`
	UserID      uuid.UUID         `json:"user_id"`
	ProjectID   *uuid.UUID        `json:"project_id,omitempty"`
	EventType   string            `json:"event_type"`
	EntityType  string            `json:"entity_type"`
	EntityID    string            `json:"entity_id"`
	Action      string            `json:"action"`
	Metadata    map[string]any    `json:"metadata"`
	SessionID   string            `json:"session_id"`
	IP          string            `json:"ip"`
	UserAgent   string            `json:"user_agent"`
	Timestamp   time.Time         `json:"timestamp"`
}

// Metric represents a time-series metric
type Metric struct {
	Name       string            `json:"name"`
	TenantID   uuid.UUID         `json:"tenant_id"`
	ProjectID  *uuid.UUID        `json:"project_id,omitempty"`
	Value      float64           `json:"value"`
	Labels     map[string]string `json:"labels"`
	Timestamp  time.Time         `json:"timestamp"`
}

// Report represents an analytics report
type Report struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	Name        string          `json:"name"`
	Type        ReportType      `json:"type"`
	Status      ReportStatus    `json:"status"`
	Parameters  ReportParameters `json:"parameters"`
	Result      *ReportResult   `json:"result,omitempty"`
	CreatedBy   uuid.UUID       `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
}

type ReportType string

const (
	ReportTypeProjectUsage      ReportType = "project_usage"
	ReportTypeUserActivity      ReportType = "user_activity"
	ReportTypeDesignMetrics     ReportType = "design_metrics"
	ReportTypeStorageUsage      ReportType = "storage_usage"
	ReportTypeCollaborationStats ReportType = "collaboration_stats"
)

type ReportStatus string

const (
	ReportStatusPending    ReportStatus = "pending"
	ReportStatusProcessing ReportStatus = "processing"
	ReportStatusCompleted  ReportStatus = "completed"
	ReportStatusFailed     ReportStatus = "failed"
)

type ReportParameters struct {
	ProjectID *uuid.UUID `json:"project_id,omitempty"`
	StartDate time.Time  `json:"start_date"`
	EndDate   time.Time  `json:"end_date"`
	Format    string     `json:"format"`
}

type ReportResult struct {
	DownloadURL string `json:"download_url"`
	Size        int64  `json:"size"`
	Rows        int64  `json:"rows"`
}

// Dashboard represents an analytics dashboard
type Dashboard struct {
	ID          uuid.UUID          `json:"id"`
	TenantID    uuid.UUID          `json:"tenant_id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Widgets     []DashboardWidget  `json:"widgets"`
	CreatedBy   uuid.UUID          `json:"created_by"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type DashboardWidget struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Title    string                 `json:"title"`
	Config   map[string]any         `json:"config"`
	Position WidgetPosition         `json:"position"`
}

type WidgetPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// AnalyticsResponse represents an analytics query response
type AnalyticsResponse struct {
	Metric    string          `json:"metric"`
	Labels    []string        `json:"labels"`
	Data      []DataPoint     `json:"data"`
	Summary   SummaryStats    `json:"summary"`
}

type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label,omitempty"`
}

type SummaryStats struct {
	Total   float64 `json:"total"`
	Average float64 `json:"average"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Count   int64   `json:"count"`
}

// QueryRequest represents an analytics query request
type QueryRequest struct {
	Metric     string    `json:"metric"`
	TenantID   uuid.UUID `json:"tenant_id"`
	ProjectID  *uuid.UUID `json:"project_id,omitempty"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	Interval   string    `json:"interval"` // hour, day, week, month
	GroupBy    []string  `json:"group_by"`
	Filters    map[string]string `json:"filters"`
}
