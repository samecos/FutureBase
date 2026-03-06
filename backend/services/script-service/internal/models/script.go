package models

import (
	"time"

	"github.com/google/uuid"
)

// Script represents a Python script
type Script struct {
	ID            uuid.UUID     `json:"id" db:"id"`
	TenantID      uuid.UUID     `json:"tenant_id" db:"tenant_id"`
	ProjectID     *uuid.UUID    `json:"project_id,omitempty" db:"project_id"`
	Name          string        `json:"name" db:"name"`
	Description   string        `json:"description" db:"description"`
	Code          string        `json:"code" db:"code"`
	Language      string        `json:"language" db:"language"`
	Version       int           `json:"version" db:"version"`
	Status        ScriptStatus  `json:"status" db:"status"`
	Tags          []string      `json:"tags" db:"tags"`
	InputSchema   string        `json:"input_schema" db:"input_schema"`
	OutputSchema  string        `json:"output_schema" db:"output_schema"`
	Dependencies  []string      `json:"dependencies" db:"dependencies"`
	TimeoutSeconds int          `json:"timeout_seconds" db:"timeout_seconds"`
	MaxMemoryMB   int           `json:"max_memory_mb" db:"max_memory_mb"`
	CreatedBy     uuid.UUID     `json:"created_by" db:"created_by"`
	UpdatedBy     *uuid.UUID    `json:"updated_by,omitempty" db:"updated_by"`
	CreatedAt     time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at" db:"updated_at"`
	DeletedAt     *time.Time    `json:"deleted_at,omitempty" db:"deleted_at"`
}

type ScriptStatus string

const (
	ScriptStatusDraft     ScriptStatus = "draft"
	ScriptStatusActive    ScriptStatus = "active"
	ScriptStatusArchived  ScriptStatus = "archived"
)

// ScriptExecution represents an execution of a script
type ScriptExecution struct {
	ID            uuid.UUID        `json:"id" db:"id"`
	ScriptID      uuid.UUID        `json:"script_id" db:"script_id"`
	TenantID      uuid.UUID        `json:"tenant_id" db:"tenant_id"`
	Version       int              `json:"version" db:"version"`
	Status        ExecutionStatus  `json:"status" db:"status"`
	Input         string           `json:"input" db:"input"`
	Output        string           `json:"output" db:"output"`
	Error         string           `json:"error" db:"error"`
	Logs          string           `json:"logs" db:"logs"`
	ExecutionTime int              `json:"execution_time_ms" db:"execution_time_ms"`
	MemoryUsage   int64            `json:"memory_usage_bytes" db:"memory_usage_bytes"`
	StartedAt     time.Time        `json:"started_at" db:"started_at"`
	CompletedAt   *time.Time       `json:"completed_at,omitempty" db:"completed_at"`
	CreatedBy     uuid.UUID        `json:"created_by" db:"created_by"`
	CacheHit      bool             `json:"cache_hit" db:"cache_hit"`
	WorkflowID    string           `json:"workflow_id,omitempty" db:"workflow_id"`
}

type ExecutionStatus string

const (
	ExecutionStatusPending    ExecutionStatus = "pending"
	ExecutionStatusRunning    ExecutionStatus = "running"
	ExecutionStatusCompleted  ExecutionStatus = "completed"
	ExecutionStatusFailed     ExecutionStatus = "failed"
	ExecutionStatusCancelled  ExecutionStatus = "cancelled"
	ExecutionStatusTimeout    ExecutionStatus = "timeout"
)

// ScriptVersion represents a version of a script
type ScriptVersion struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	ScriptID     uuid.UUID  `json:"script_id" db:"script_id"`
	Version      int        `json:"version" db:"version"`
	Code         string     `json:"code" db:"code"`
	ChangeLog    string     `json:"change_log" db:"change_log"`
	CreatedBy    uuid.UUID  `json:"created_by" db:"created_by"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// ScriptTemplate represents a reusable script template
type ScriptTemplate struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	TenantID    uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	Category    string     `json:"category" db:"category"`
	Code        string     `json:"code" db:"code"`
	InputSchema string     `json:"input_schema" db:"input_schema"`
	IsSystem    bool       `json:"is_system" db:"is_system"`
	CreatedBy   uuid.UUID  `json:"created_by" db:"created_by"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// ExecutionRequest represents a request to execute a script
type ExecutionRequest struct {
	ScriptID   uuid.UUID       `json:"script_id"`
	Input      map[string]any  `json:"input"`
	Async      bool            `json:"async"`
	CacheKey   string          `json:"cache_key,omitempty"`
	Timeout    int             `json:"timeout_seconds,omitempty"`
}

// ExecutionResponse represents the response from executing a script
type ExecutionResponse struct {
	ExecutionID   uuid.UUID       `json:"execution_id"`
	Status        ExecutionStatus `json:"status"`
	Output        any             `json:"output,omitempty"`
	Error         string          `json:"error,omitempty"`
	Logs          string          `json:"logs,omitempty"`
	ExecutionTime int             `json:"execution_time_ms"`
	CacheHit      bool            `json:"cache_hit"`
}
