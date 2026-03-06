package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/archplatform/script-service/internal/models"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type PostgresStorage struct {
	db *sql.DB
}

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

func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// Script operations
func (s *PostgresStorage) CreateScript(ctx context.Context, script *models.Script) error {
	query := `
		INSERT INTO scripts.scripts (id, tenant_id, project_id, name, description, code, language, version, status, tags,
			input_schema, output_schema, dependencies, timeout_seconds, max_memory_mb, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`
	
	tagsJSON, _ := json.Marshal(script.Tags)
	depsJSON, _ := json.Marshal(script.Dependencies)
	
	_, err := s.db.ExecContext(ctx, query,
		script.ID, script.TenantID, script.ProjectID, script.Name, script.Description,
		script.Code, script.Language, script.Version, script.Status, string(tagsJSON),
		script.InputSchema, script.OutputSchema, string(depsJSON),
		script.TimeoutSeconds, script.MaxMemoryMB, script.CreatedBy,
		script.CreatedAt, script.UpdatedAt)
	
	return err
}

func (s *PostgresStorage) GetScript(ctx context.Context, id uuid.UUID) (*models.Script, error) {
	query := `
		SELECT id, tenant_id, project_id, name, description, code, language, version, status, tags,
			input_schema, output_schema, dependencies, timeout_seconds, max_memory_mb,
			created_by, updated_by, created_at, updated_at, deleted_at
		FROM scripts.scripts WHERE id = $1 AND deleted_at IS NULL
	`
	
	var script models.Script
	var tagsJSON, depsJSON string
	
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&script.ID, &script.TenantID, &script.ProjectID, &script.Name, &script.Description,
		&script.Code, &script.Language, &script.Version, &script.Status, &tagsJSON,
		&script.InputSchema, &script.OutputSchema, &depsJSON,
		&script.TimeoutSeconds, &script.MaxMemoryMB,
		&script.CreatedBy, &script.UpdatedBy, &script.CreatedAt, &script.UpdatedAt, &script.DeletedAt)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("script not found")
	}
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal([]byte(tagsJSON), &script.Tags)
	json.Unmarshal([]byte(depsJSON), &script.Dependencies)
	
	return &script, nil
}

func (s *PostgresStorage) GetScriptsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.Script, error) {
	query := `
		SELECT id, tenant_id, project_id, name, description, code, language, version, status, tags,
			input_schema, output_schema, dependencies, timeout_seconds, max_memory_mb,
			created_by, updated_by, created_at, updated_at
		FROM scripts.scripts WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY updated_at DESC
	`
	
	return s.queryScripts(ctx, query, tenantID)
}

func (s *PostgresStorage) UpdateScript(ctx context.Context, script *models.Script) error {
	query := `
		UPDATE scripts.scripts SET
			name = $2, description = $3, code = $4, status = $5, tags = $6,
			input_schema = $7, output_schema = $8, dependencies = $9,
			timeout_seconds = $10, max_memory_mb = $11, updated_by = $12, updated_at = $13
		WHERE id = $1
	`
	
	tagsJSON, _ := json.Marshal(script.Tags)
	depsJSON, _ := json.Marshal(script.Dependencies)
	
	_, err := s.db.ExecContext(ctx, query,
		script.ID, script.Name, script.Description, script.Code, script.Status, string(tagsJSON),
		script.InputSchema, script.OutputSchema, string(depsJSON),
		script.TimeoutSeconds, script.MaxMemoryMB, script.UpdatedBy, script.UpdatedAt)
	
	return err
}

func (s *PostgresStorage) DeleteScript(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE scripts.scripts SET deleted_at = $2 WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id, time.Now())
	return err
}

// Execution operations
func (s *PostgresStorage) CreateExecution(ctx context.Context, exec *models.ScriptExecution) error {
	query := `
		INSERT INTO scripts.executions (id, script_id, tenant_id, version, status, input, output, error, logs,
			execution_time_ms, memory_usage_bytes, started_at, completed_at, created_by, cache_hit, workflow_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`
	
	_, err := s.db.ExecContext(ctx, query,
		exec.ID, exec.ScriptID, exec.TenantID, exec.Version, exec.Status,
		exec.Input, exec.Output, exec.Error, exec.Logs,
		exec.ExecutionTime, exec.MemoryUsage, exec.StartedAt, exec.CompletedAt,
		exec.CreatedBy, exec.CacheHit, exec.WorkflowID)
	
	return err
}

func (s *PostgresStorage) GetExecution(ctx context.Context, id uuid.UUID) (*models.ScriptExecution, error) {
	query := `
		SELECT id, script_id, tenant_id, version, status, input, output, error, logs,
			execution_time_ms, memory_usage_bytes, started_at, completed_at, created_by, cache_hit, workflow_id
		FROM scripts.executions WHERE id = $1
	`
	
	var exec models.ScriptExecution
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&exec.ID, &exec.ScriptID, &exec.TenantID, &exec.Version, &exec.Status,
		&exec.Input, &exec.Output, &exec.Error, &exec.Logs,
		&exec.ExecutionTime, &exec.MemoryUsage, &exec.StartedAt, &exec.CompletedAt,
		&exec.CreatedBy, &exec.CacheHit, &exec.WorkflowID)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found")
	}
	
	return &exec, err
}

func (s *PostgresStorage) UpdateExecution(ctx context.Context, exec *models.ScriptExecution) error {
	query := `
		UPDATE scripts.executions SET
			status = $2, output = $3, error = $4, logs = $5,
			execution_time_ms = $6, memory_usage_bytes = $7, completed_at = $8
		WHERE id = $1
	`
	
	_, err := s.db.ExecContext(ctx, query,
		exec.ID, exec.Status, exec.Output, exec.Error, exec.Logs,
		exec.ExecutionTime, exec.MemoryUsage, exec.CompletedAt)
	
	return err
}

func (s *PostgresStorage) GetExecutionsByScript(ctx context.Context, scriptID uuid.UUID, limit int) ([]*models.ScriptExecution, error) {
	query := `
		SELECT id, script_id, tenant_id, version, status, input, output, error, logs,
			execution_time_ms, memory_usage_bytes, started_at, completed_at, created_by, cache_hit, workflow_id
		FROM scripts.executions WHERE script_id = $1
		ORDER BY started_at DESC LIMIT $2
	`
	
	rows, err := s.db.QueryContext(ctx, query, scriptID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	return s.scanExecutions(rows)
}

// Helper methods
func (s *PostgresStorage) queryScripts(ctx context.Context, query string, args ...any) ([]*models.Script, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var scripts []*models.Script
	for rows.Next() {
		var script models.Script
		var tagsJSON, depsJSON string
		
		err := rows.Scan(
			&script.ID, &script.TenantID, &script.ProjectID, &script.Name, &script.Description,
			&script.Code, &script.Language, &script.Version, &script.Status, &tagsJSON,
			&script.InputSchema, &script.OutputSchema, &depsJSON,
			&script.TimeoutSeconds, &script.MaxMemoryMB,
			&script.CreatedBy, &script.UpdatedBy, &script.CreatedAt, &script.UpdatedAt)
		
		if err != nil {
			return nil, err
		}
		
		json.Unmarshal([]byte(tagsJSON), &script.Tags)
		json.Unmarshal([]byte(depsJSON), &script.Dependencies)
		scripts = append(scripts, &script)
	}
	
	return scripts, rows.Err()
}

func (s *PostgresStorage) scanExecutions(rows *sql.Rows) ([]*models.ScriptExecution, error) {
	var executions []*models.ScriptExecution
	
	for rows.Next() {
		var exec models.ScriptExecution
		err := rows.Scan(
			&exec.ID, &exec.ScriptID, &exec.TenantID, &exec.Version, &exec.Status,
			&exec.Input, &exec.Output, &exec.Error, &exec.Logs,
			&exec.ExecutionTime, &exec.MemoryUsage, &exec.StartedAt, &exec.CompletedAt,
			&exec.CreatedBy, &exec.CacheHit, &exec.WorkflowID)
		
		if err != nil {
			return nil, err
		}
		executions = append(executions, &exec)
	}
	
	return executions, rows.Err()
}
