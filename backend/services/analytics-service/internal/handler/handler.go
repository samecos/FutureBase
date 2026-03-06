package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/archplatform/analytics-service/internal/models"
	"github.com/archplatform/analytics-service/internal/storage"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handler handles HTTP requests
type Handler struct {
	storage *storage.PostgresStorage
}

// NewHandler creates a new handler
func NewHandler(storage *storage.PostgresStorage) *Handler {
	return &Handler{storage: storage}
}

// RegisterRoutes registers HTTP routes
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/health", h.HealthCheck).Methods("GET")
	r.HandleFunc("/events", h.TrackEvent).Methods("POST")
	r.HandleFunc("/events", h.GetEvents).Methods("GET")
	r.HandleFunc("/reports", h.CreateReport).Methods("POST")
	r.HandleFunc("/reports", h.GetReports).Methods("GET")
	r.HandleFunc("/reports/{id}", h.GetReport).Methods("GET")
	r.HandleFunc("/metrics/{name}", h.GetMetrics).Methods("GET")
	r.HandleFunc("/query", h.QueryAnalytics).Methods("POST")
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// TrackEvent tracks an analytics event
func (h *Handler) TrackEvent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID   uuid.UUID       `json:"tenant_id"`
		UserID     uuid.UUID       `json:"user_id"`
		ProjectID  *uuid.UUID      `json:"project_id,omitempty"`
		EventType  string          `json:"event_type"`
		EntityType string          `json:"entity_type"`
		EntityID   string          `json:"entity_id"`
		Action     string          `json:"action"`
		Metadata   map[string]any  `json:"metadata"`
		SessionID  string          `json:"session_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	event := &models.Event{
		ID:         uuid.New(),
		TenantID:   req.TenantID,
		UserID:     req.UserID,
		ProjectID:  req.ProjectID,
		EventType:  req.EventType,
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
		Action:     req.Action,
		Metadata:   req.Metadata,
		SessionID:  req.SessionID,
		IP:         r.RemoteAddr,
		UserAgent:  r.UserAgent(),
		Timestamp:  time.Now(),
	}

	if err := h.storage.CreateEvent(r.Context(), event); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}

// GetEvents retrieves analytics events
func (h *Handler) GetEvents(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	tid := uuid.MustParse(tenantID)

	// Default to last 7 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)

	events, err := h.storage.GetEvents(r.Context(), tid, startDate, endDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(events)
}

// CreateReport creates a new analytics report
func (h *Handler) CreateReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID   uuid.UUID     `json:"tenant_id"`
		Name       string        `json:"name"`
		Type       string        `json:"type"`
		StartDate  time.Time     `json:"start_date"`
		EndDate    time.Time     `json:"end_date"`
		Format     string        `json:"format"`
		CreatedBy  uuid.UUID     `json:"created_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	report := &models.Report{
		ID:        uuid.New(),
		TenantID:  req.TenantID,
		Name:      req.Name,
		Type:      models.ReportType(req.Type),
		Status:    models.ReportStatusPending,
		Parameters: models.ReportParameters{
			StartDate: req.StartDate,
			EndDate:   req.EndDate,
			Format:    req.Format,
		},
		CreatedBy: req.CreatedBy,
		CreatedAt: time.Now(),
	}

	if err := h.storage.CreateReport(r.Context(), report); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(report)
}

// GetReport retrieves a report
func (h *Handler) GetReport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	report, err := h.storage.GetReport(r.Context(), id)
	if err != nil {
		http.Error(w, "Report not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(report)
}

// GetReports lists reports for a tenant
func (h *Handler) GetReports(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	tid := uuid.MustParse(tenantID)
	reports, err := h.storage.GetReportsByTenant(r.Context(), tid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(reports)
}

// GetMetrics returns analytics metrics
func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	metricName := vars["name"]

	// Return mock metrics for now
	response := models.AnalyticsResponse{
		Metric: metricName,
		Labels: []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"},
		Data: []models.DataPoint{
			{Timestamp: time.Now().AddDate(0, 0, -30), Value: 100},
			{Timestamp: time.Now().AddDate(0, 0, -25), Value: 120},
			{Timestamp: time.Now().AddDate(0, 0, -20), Value: 115},
			{Timestamp: time.Now().AddDate(0, 0, -15), Value: 140},
			{Timestamp: time.Now().AddDate(0, 0, -10), Value: 160},
			{Timestamp: time.Now().AddDate(0, 0, -5), Value: 180},
		},
		Summary: models.SummaryStats{
			Total:   815,
			Average: 135.8,
			Min:     100,
			Max:     180,
			Count:   6,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// QueryAnalytics queries analytics data
func (h *Handler) QueryAnalytics(w http.ResponseWriter, r *http.Request) {
	var req models.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return mock response
	response := models.AnalyticsResponse{
		Metric: req.Metric,
		Data:   []models.DataPoint{},
		Summary: models.SummaryStats{
			Total:   1000,
			Average: 100,
			Min:     50,
			Max:     150,
			Count:   10,
		},
	}

	json.NewEncoder(w).Encode(response)
}
