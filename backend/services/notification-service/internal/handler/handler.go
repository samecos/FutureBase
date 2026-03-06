package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/archplatform/notification-service/internal/models"
	"github.com/archplatform/notification-service/internal/storage"
	"github.com/archplatform/notification-service/pkg/email"
	"github.com/archplatform/notification-service/pkg/websocket"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handler handles HTTP requests
type Handler struct {
	storage   *storage.PostgresStorage
	wsManager *websocket.Manager
	emailSender *email.Sender
}

// NewHandler creates a new handler
func NewHandler(storage *storage.PostgresStorage, wsManager *websocket.Manager, emailSender *email.Sender) *Handler {
	return &Handler{
		storage:     storage,
		wsManager:   wsManager,
		emailSender: emailSender,
	}
}

// RegisterRoutes registers HTTP routes
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/health", h.HealthCheck).Methods("GET")
	r.HandleFunc("/notifications", h.GetNotifications).Methods("GET")
	r.HandleFunc("/notifications", h.CreateNotification).Methods("POST")
	r.HandleFunc("/notifications/{id}/read", h.MarkAsRead).Methods("POST")
	r.HandleFunc("/notifications/read-all", h.MarkAllAsRead).Methods("POST")
	r.HandleFunc("/notifications/unread", h.GetUnreadCount).Methods("GET")
	r.HandleFunc("/ws", h.WebSocketHandler).Methods("GET")
	r.HandleFunc("/preferences", h.GetPreferences).Methods("GET")
	r.HandleFunc("/preferences", h.UpdatePreferences).Methods("PUT")
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]any{
		"status":            "healthy",
		"connected_clients": h.wsManager.GetConnectedCount(),
	})
}

// GetNotifications retrieves notifications for the user
func (h *Handler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	uid := uuid.MustParse(userID)
	
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	notifications, err := h.storage.GetNotificationsByUser(r.Context(), uid, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(notifications)
}

// CreateNotification creates a new notification
func (h *Handler) CreateNotification(w http.ResponseWriter, r *http.Request) {
	var req models.CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dataJSON, _ := json.Marshal(req.Data)
	
	notification := &models.Notification{
		ID:         uuid.New(),
		TenantID:   req.TenantID,
		UserID:     req.UserID,
		Type:       req.Type,
		Title:      req.Title,
		Content:    req.Content,
		Data:       string(dataJSON),
		Priority:   req.Priority,
		Status:     models.NotificationStatusPending,
		Channels:   req.Channels,
		ActionURL:  req.ActionURL,
		ImageURL:   req.ImageURL,
		SourceType: req.SourceType,
		SourceID:   req.SourceID,
		CreatedAt:  time.Now(),
	}

	if err := h.storage.CreateNotification(r.Context(), notification); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send via WebSocket if enabled
	if contains(notification.Channels, "websocket") {
		h.wsManager.SendNotification(req.UserID, notification)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(notification)
}

// MarkAsRead marks a notification as read
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.storage.MarkNotificationAsRead(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// MarkAllAsRead marks all notifications as read for a user
func (h *Handler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	uid := uuid.MustParse(userID)
	if err := h.storage.MarkAllNotificationsAsRead(r.Context(), uid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetUnreadCount gets the count of unread notifications
func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	uid := uuid.MustParse(userID)
	notifications, err := h.storage.GetUnreadNotificationsByUser(r.Context(), uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]int{"unread_count": len(notifications)})
}

// WebSocketHandler handles WebSocket connections
func (h *Handler) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	uid := uuid.MustParse(userID)
	h.wsManager.HandleConnection(w, r, uid)
}

// GetPreferences retrieves notification preferences for a user
func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	uid := uuid.MustParse(userID)
	prefs, err := h.storage.GetNotificationPreference(r.Context(), uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(prefs)
}

// UpdatePreferences updates notification preferences
func (h *Handler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	var prefs models.NotificationPreference
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	prefs.ID = uuid.New()
	prefs.CreatedAt = time.Now()
	prefs.UpdatedAt = time.Now()

	if err := h.storage.SaveNotificationPreference(r.Context(), &prefs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(prefs)
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

