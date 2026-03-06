package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/archplatform/file-service/internal/models"
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

// CreateFile creates a new file record
func (s *PostgresStorage) CreateFile(ctx context.Context, file *models.File) error {
	query := `
		INSERT INTO files.files (id, tenant_id, project_id, design_id, name, original_name, content_type, size, 
			storage_key, storage_type, checksum, status, version, tags, metadata, uploaded_by, uploaded_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	tagsJSON, _ := json.Marshal(file.Tags)

	_, err := s.db.ExecContext(ctx, query,
		file.ID, file.TenantID, file.ProjectID, file.DesignID, file.Name, file.OriginalName,
		file.ContentType, file.Size, file.StorageKey, file.StorageType, file.Checksum,
		file.Status, file.Version, string(tagsJSON), file.Metadata,
		file.UploadedBy, file.UploadedAt, file.UpdatedAt)

	return err
}

// GetFile retrieves a file by ID
func (s *PostgresStorage) GetFile(ctx context.Context, id uuid.UUID) (*models.File, error) {
	query := `
		SELECT id, tenant_id, project_id, design_id, name, original_name, content_type, size,
			storage_key, storage_type, checksum, status, version, tags, metadata,
			uploaded_by, uploaded_at, updated_at, deleted_at
		FROM files.files WHERE id = $1 AND deleted_at IS NULL
	`

	var file models.File
	var tagsJSON string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&file.ID, &file.TenantID, &file.ProjectID, &file.DesignID, &file.Name, &file.OriginalName,
		&file.ContentType, &file.Size, &file.StorageKey, &file.StorageType, &file.Checksum,
		&file.Status, &file.Version, &tagsJSON, &file.Metadata,
		&file.UploadedBy, &file.UploadedAt, &file.UpdatedAt, &file.DeletedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(tagsJSON), &file.Tags)
	return &file, nil
}

// GetFilesByProject retrieves files for a project
func (s *PostgresStorage) GetFilesByProject(ctx context.Context, projectID uuid.UUID) ([]*models.File, error) {
	query := `
		SELECT id, tenant_id, project_id, design_id, name, original_name, content_type, size,
			storage_key, storage_type, checksum, status, version, tags, metadata,
			uploaded_by, uploaded_at, updated_at
		FROM files.files WHERE project_id = $1 AND deleted_at IS NULL
		ORDER BY uploaded_at DESC
	`

	return s.queryFiles(ctx, query, projectID)
}

// UpdateFile updates a file record
func (s *PostgresStorage) UpdateFile(ctx context.Context, file *models.File) error {
	query := `
		UPDATE files.files SET
			name = $2, content_type = $3, status = $4, tags = $5, metadata = $6, updated_at = $7
		WHERE id = $1
	`

	tagsJSON, _ := json.Marshal(file.Tags)

	_, err := s.db.ExecContext(ctx, query,
		file.ID, file.Name, file.ContentType, file.Status, string(tagsJSON), file.Metadata, time.Now())

	return err
}

// DeleteFile soft deletes a file
func (s *PostgresStorage) DeleteFile(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE files.files SET deleted_at = $2 WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id, time.Now())
	return err
}

// CreateThumbnail creates a thumbnail record
func (s *PostgresStorage) CreateThumbnail(ctx context.Context, thumb *models.Thumbnail) error {
	query := `
		INSERT INTO files.thumbnails (id, file_id, width, height, format, size, storage_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := s.db.ExecContext(ctx, query,
		thumb.ID, thumb.FileID, thumb.Width, thumb.Height, thumb.Format, thumb.Size, thumb.StorageKey, thumb.CreatedAt)

	return err
}

// GetThumbnailByFile retrieves thumbnail for a file
func (s *PostgresStorage) GetThumbnailByFile(ctx context.Context, fileID uuid.UUID) (*models.Thumbnail, error) {
	query := `
		SELECT id, file_id, width, height, format, size, storage_key, created_at
		FROM files.thumbnails WHERE file_id = $1
	`

	var thumb models.Thumbnail
	err := s.db.QueryRowContext(ctx, query, fileID).Scan(
		&thumb.ID, &thumb.FileID, &thumb.Width, &thumb.Height,
		&thumb.Format, &thumb.Size, &thumb.StorageKey, &thumb.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &thumb, err
}

// CreateConversionJob creates a conversion job
func (s *PostgresStorage) CreateConversionJob(ctx context.Context, job *models.ConversionJob) error {
	query := `
		INSERT INTO files.conversion_jobs (id, source_file_id, target_format, status, progress, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.db.ExecContext(ctx, query,
		job.ID, job.SourceFileID, job.TargetFormat, job.Status, job.Progress, job.CreatedAt)

	return err
}

// GetConversionJob retrieves a conversion job
func (s *PostgresStorage) GetConversionJob(ctx context.Context, id uuid.UUID) (*models.ConversionJob, error) {
	query := `
		SELECT id, source_file_id, target_format, status, output_file_id, error, progress, created_at, completed_at
		FROM files.conversion_jobs WHERE id = $1
	`

	var job models.ConversionJob
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&job.ID, &job.SourceFileID, &job.TargetFormat, &job.Status,
		&job.OutputFileID, &job.Error, &job.Progress, &job.CreatedAt, &job.CompletedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found")
	}
	return &job, err
}

// UpdateConversionJob updates a conversion job
func (s *PostgresStorage) UpdateConversionJob(ctx context.Context, job *models.ConversionJob) error {
	query := `
		UPDATE files.conversion_jobs SET
			status = $2, output_file_id = $3, error = $4, progress = $5, completed_at = $6
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query,
		job.ID, job.Status, job.OutputFileID, job.Error, job.Progress, job.CompletedAt)

	return err
}

// Helper methods
func (s *PostgresStorage) queryFiles(ctx context.Context, query string, args ...any) ([]*models.File, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*models.File
	for rows.Next() {
		var file models.File
		var tagsJSON string

		err := rows.Scan(
			&file.ID, &file.TenantID, &file.ProjectID, &file.DesignID, &file.Name, &file.OriginalName,
			&file.ContentType, &file.Size, &file.StorageKey, &file.StorageType, &file.Checksum,
			&file.Status, &file.Version, &tagsJSON, &file.Metadata,
			&file.UploadedBy, &file.UploadedAt, &file.UpdatedAt)

		if err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(tagsJSON), &file.Tags)
		files = append(files, &file)
	}

	return files, rows.Err()
}
