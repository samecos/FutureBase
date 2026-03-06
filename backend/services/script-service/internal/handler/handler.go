package handler

import (
	"encoding/json"
	"net/http"

	"github.com/archplatform/script-service/internal/engine"
	"github.com/archplatform/script-service/internal/models"
	"github.com/archplatform/script-service/internal/storage"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handler handles HTTP requests
type Handler struct {
	storage *storage.PostgresStorage
	engine  *engine.Engine
}

// NewHandler creates a new handler
func NewHandler(storage *storage.PostgresStorage, eng *engine.Engine) *Handler {
	return &Handler{
		storage: storage,
		engine:  eng,
	}
}

// RegisterRoutes registers HTTP routes
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/health", h.HealthCheck).Methods("GET")
	r.HandleFunc("/scripts", h.ListScripts).Methods("GET")
	r.HandleFunc("/scripts", h.CreateScript).Methods("POST")
	r.HandleFunc("/scripts/{id}", h.GetScript).Methods("GET")
	r.HandleFunc("/scripts/{id}", h.UpdateScript).Methods("PUT")
	r.HandleFunc("/scripts/{id}", h.DeleteScript).Methods("DELETE")
	r.HandleFunc("/scripts/{id}/execute", h.ExecuteScript).Methods("POST")
	r.HandleFunc("/scripts/{id}/validate", h.ValidateScript).Methods("POST")
	r.HandleFunc("/executions/{id}", h.GetExecution).Methods("GET")
	r.HandleFunc("/executions", h.ListExecutions).Methods("GET")
	r.HandleFunc("/packages", h.ListPackages).Methods("GET")
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// CreateScript handles script creation
func (h *Handler) CreateScript(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID       string   `json:"tenant_id"`
		ProjectID      string   `json:"project_id,omitempty"`
		Name           string   `json:"name"`
		Description    string   `json:"description"`
		Code           string   `json:"code"`
		Language       string   `json:"language"`
		Tags           []string `json:"tags"`
		Dependencies   []string `json:"dependencies"`
		TimeoutSeconds int      `json:"timeout_seconds"`
		MaxMemoryMB    int      `json:"max_memory_mb"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate code
	if err := h.engine.Validate(req.Code); err != nil {
		http.Error(w, "Invalid code: "+err.Error(), http.StatusBadRequest)
		return
	}

	script := &models.Script{
		ID:             uuid.New(),
		TenantID:       uuid.MustParse(req.TenantID),
		Name:           req.Name,
		Description:    req.Description,
		Code:           req.Code,
		Language:       req.Language,
		Version:        1,
		Status:         models.ScriptStatusDraft,
		Tags:           req.Tags,
		Dependencies:   req.Dependencies,
		TimeoutSeconds: req.TimeoutSeconds,
		MaxMemoryMB:    req.MaxMemoryMB,
	}

	if req.ProjectID != "" {
		pid := uuid.MustParse(req.ProjectID)
		script.ProjectID = &pid
	}

	if err := h.storage.CreateScript(r.Context(), script); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(script)
}

// GetScript handles getting a script
func (h *Handler) GetScript(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	script, err := h.storage.GetScript(r.Context(), id)
	if err != nil {
		http.Error(w, "Script not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(script)
}

// ListScripts handles listing scripts
func (h *Handler) ListScripts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	tid := uuid.MustParse(tenantID)
	scripts, err := h.storage.GetScriptsByTenant(r.Context(), tid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(scripts)
}

// UpdateScript handles updating a script
func (h *Handler) UpdateScript(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	script, err := h.storage.GetScript(r.Context(), id)
	if err != nil {
		http.Error(w, "Script not found", http.StatusNotFound)
		return
	}

	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Code        string   `json:"code"`
		Status      string   `json:"status"`
		Tags        []string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Code != "" {
		if err := h.engine.Validate(req.Code); err != nil {
			http.Error(w, "Invalid code: "+err.Error(), http.StatusBadRequest)
			return
		}
		script.Code = req.Code
		script.Version++
	}

	script.Name = req.Name
	script.Description = req.Description
	script.Status = models.ScriptStatus(req.Status)
	script.Tags = req.Tags

	if err := h.storage.UpdateScript(r.Context(), script); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(script)
}

// DeleteScript handles deleting a script
func (h *Handler) DeleteScript(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.storage.DeleteScript(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ExecuteScript handles script execution
func (h *Handler) ExecuteScript(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scriptID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	script, err := h.storage.GetScript(r.Context(), scriptID)
	if err != nil {
		http.Error(w, "Script not found", http.StatusNotFound)
		return
	}

	var req struct {
		Input  map[string]any `json:"input"`
		UserID string         `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Execute
	result, err := h.engine.Execute(r.Context(), script, req.Input)
	
	response := models.ExecutionResponse{
		Status:        models.ExecutionStatusCompleted,
		ExecutionTime: result.ExecutionTime,
	}

	if err != nil {
		response.Status = models.ExecutionStatusFailed
		response.Error = err.Error()
	} else if result.Error != "" {
		response.Status = models.ExecutionStatusFailed
		response.Error = result.Error
		response.Logs = result.Logs
	} else {
		response.Output = result.Output
		response.Logs = result.Logs
	}

	json.NewEncoder(w).Encode(response)
}

// ValidateScript validates script code
func (h *Handler) ValidateScript(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scriptID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	script, err := h.storage.GetScript(r.Context(), scriptID)
	if err != nil {
		http.Error(w, "Script not found", http.StatusNotFound)
		return
	}

	if err := h.engine.Validate(script.Code); err != nil {
		json.NewEncoder(w).Encode(map[string]any{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{"valid": true})
}

// GetExecution gets execution details
func (h *Handler) GetExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	execution, err := h.storage.GetExecution(r.Context(), id)
	if err != nil {
		http.Error(w, "Execution not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(execution)
}

// ListExecutions lists executions for a script
func (h *Handler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	scriptID := r.URL.Query().Get("script_id")
	if scriptID == "" {
		http.Error(w, "script_id required", http.StatusBadRequest)
		return
	}

	sid := uuid.MustParse(scriptID)
	executions, err := h.storage.GetExecutionsByScript(r.Context(), sid, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(executions)
}

// ListPackages lists installed Python packages
func (h *Handler) ListPackages(w http.ResponseWriter, r *http.Request) {
	packages, err := h.engine.GetInstalledPackages()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{"packages": packages})
}
