package models

import (
	"time"

	"github.com/google/uuid"
)

// File represents a stored file
type File struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	TenantID     uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	ProjectID    *uuid.UUID `json:"project_id,omitempty" db:"project_id"`
	DesignID     *uuid.UUID `json:"design_id,omitempty" db:"design_id"`
	Name         string     `json:"name" db:"name"`
	OriginalName string     `json:"original_name" db:"original_name"`
	ContentType  string     `json:"content_type" db:"content_type"`
	Size         int64      `json:"size" db:"size"`
	StorageKey   string     `json:"storage_key" db:"storage_key"`
	StorageType  string     `json:"storage_type" db:"storage_type"`
	Checksum     string     `json:"checksum" db:"checksum"`
	Status       FileStatus `json:"status" db:"status"`
	Version      int        `json:"version" db:"version"`
	Tags         []string   `json:"tags" db:"tags"`
	Metadata     string     `json:"metadata" db:"metadata"`
	UploadedBy   uuid.UUID  `json:"uploaded_by" db:"uploaded_by"`
	UploadedAt   time.Time  `json:"uploaded_at" db:"uploaded_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

type FileStatus string

const (
	FileStatusUploading   FileStatus = "uploading"
	FileStatusProcessing  FileStatus = "processing"
	FileStatusActive      FileStatus = "active"
	FileStatusArchived    FileStatus = "archived"
	FileStatusError       FileStatus = "error"
)

// FileVersion represents a version of a file
type FileVersion struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	FileID     uuid.UUID  `json:"file_id" db:"file_id"`
	Version    int        `json:"version" db:"version"`
	StorageKey string     `json:"storage_key" db:"storage_key"`
	Size       int64      `json:"size" db:"size"`
	Checksum   string     `json:"checksum" db:"checksum"`
	CreatedBy  uuid.UUID  `json:"created_by" db:"created_by"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// Thumbnail represents a generated thumbnail
type Thumbnail struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	FileID     uuid.UUID  `json:"file_id" db:"file_id"`
	Width      int        `json:"width" db:"width"`
	Height     int        `json:"height" db:"height"`
	Format     string     `json:"format" db:"format"`
	Size       int64      `json:"size" db:"size"`
	StorageKey string     `json:"storage_key" db:"storage_key"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// ConversionJob represents a file format conversion job
type ConversionJob struct {
	ID           uuid.UUID     `json:"id" db:"id"`
	SourceFileID uuid.UUID     `json:"source_file_id" db:"source_file_id"`
	TargetFormat string        `json:"target_format" db:"target_format"`
	Status       JobStatus     `json:"status" db:"status"`
	OutputFileID *uuid.UUID    `json:"output_file_id,omitempty" db:"output_file_id"`
	Error        string        `json:"error" db:"error"`
	Progress     int           `json:"progress" db:"progress"`
	CreatedAt    time.Time     `json:"created_at" db:"created_at"`
	CompletedAt  *time.Time    `json:"completed_at,omitempty" db:"completed_at"`
}

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// UploadRequest represents a file upload request
type UploadRequest struct {
	TenantID     uuid.UUID       `json:"tenant_id"`
	ProjectID    *uuid.UUID      `json:"project_id,omitempty"`
	DesignID     *uuid.UUID      `json:"design_id,omitempty"`
	Name         string          `json:"name"`
	ContentType  string          `json:"content_type"`
	Size         int64           `json:"size"`
	Tags         []string        `json:"tags"`
	Metadata     map[string]any  `json:"metadata"`
}

// UploadResponse represents a file upload response
type UploadResponse struct {
	FileID      uuid.UUID  `json:"file_id"`
	StorageKey  string     `json:"storage_key"`
	UploadURL   string     `json:"upload_url,omitempty"`
	Status      string     `json:"status"`
}

// FileInfo represents file information for download
type FileInfo struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	OriginalName string    `json:"original_name"`
	ContentType  string    `json:"content_type"`
	Size         int64     `json:"size"`
	DownloadURL  string    `json:"download_url"`
	UploadedAt   time.Time `json:"uploaded_at"`
}
