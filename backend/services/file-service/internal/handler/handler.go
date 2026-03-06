package handler

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/archplatform/file-service/internal/models"
	"github.com/archplatform/file-service/internal/storage"
	"github.com/archplatform/file-service/internal/thumbnail"
	"github.com/archplatform/file-service/pkg/minio"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handler handles HTTP requests
type Handler struct {
	storage   *storage.PostgresStorage
	minio     *minio.Client
	thumbnail *thumbnail.Generator
}

// NewHandler creates a new handler
func NewHandler(storage *storage.PostgresStorage, minioClient *minio.Client, thumbGen *thumbnail.Generator) *Handler {
	return &Handler{
		storage:   storage,
		minio:     minioClient,
		thumbnail: thumbGen,
	}
}

// RegisterRoutes registers HTTP routes
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/health", h.HealthCheck).Methods("GET")
	r.HandleFunc("/files", h.ListFiles).Methods("GET")
	r.HandleFunc("/files", h.UploadFile).Methods("POST")
	r.HandleFunc("/files/{id}", h.GetFile).Methods("GET")
	r.HandleFunc("/files/{id}/download", h.DownloadFile).Methods("GET")
	r.HandleFunc("/files/{id}/thumbnail", h.GetThumbnail).Methods("GET")
	r.HandleFunc("/files/{id}", h.DeleteFile).Methods("DELETE")
	r.HandleFunc("/upload-url", h.GetUploadURL).Methods("POST")
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// UploadFile handles file upload
func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(100 << 20) // 100MB max
	if err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get metadata from form
	tenantID := r.FormValue("tenant_id")
	projectID := r.FormValue("project_id")
	userID := r.FormValue("user_id")

	if tenantID == "" || userID == "" {
		http.Error(w, "tenant_id and user_id required", http.StatusBadRequest)
		return
	}

	// Calculate checksum
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		http.Error(w, "Failed to calculate checksum", http.StatusInternalServerError)
		return
	}
	checksum := hex.EncodeToString(hash.Sum(nil))

	// Reset file reader
	file.Seek(0, io.SeekStart)

	// Generate storage key
	storageKey := fmt.Sprintf("%s/%s/%s", tenantID, uuid.New().String(), header.Filename)

	// Upload to MinIO
	if err := h.minio.Upload(r.Context(), storageKey, file, header.Size, header.Header.Get("Content-Type")); err != nil {
		http.Error(w, "Failed to upload file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create file record
	fileRecord := &models.File{
		ID:           uuid.New(),
		TenantID:     uuid.MustParse(tenantID),
		Name:         header.Filename,
		OriginalName: header.Filename,
		ContentType:  header.Header.Get("Content-Type"),
		Size:         header.Size,
		StorageKey:   storageKey,
		StorageType:  "minio",
		Checksum:     checksum,
		Status:       models.FileStatusActive,
		Version:      1,
		UploadedBy:   uuid.MustParse(userID),
		UploadedAt:   time.Now(),
		UpdatedAt:    time.Now(),
	}

	if projectID != "" {
		pid := uuid.MustParse(projectID)
		fileRecord.ProjectID = &pid
	}

	if err := h.storage.CreateFile(r.Context(), fileRecord); err != nil {
		// Try to delete from MinIO on error
		h.minio.Delete(r.Context(), storageKey)
		http.Error(w, "Failed to save file record: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate thumbnail for images
	if h.thumbnail.CanThumbnail(fileRecord.ContentType) {
		go h.generateThumbnail(fileRecord)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(fileRecord)
}

// GetFile retrieves file metadata
func (h *Handler) GetFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	file, err := h.storage.GetFile(r.Context(), id)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(file)
}

// DownloadFile handles file download
func (h *Handler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	file, err := h.storage.GetFile(r.Context(), id)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Generate presigned URL
	url, err := h.minio.GetPresignedDownloadURL(r.Context(), file.StorageKey, 15*time.Minute)
	if err != nil {
		http.Error(w, "Failed to generate download URL", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"download_url": url,
		"filename":     file.OriginalName,
	})
}

// GetThumbnail retrieves thumbnail for an image
func (h *Handler) GetThumbnail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	thumb, err := h.storage.GetThumbnailByFile(r.Context(), id)
	if err != nil {
		http.Error(w, "Thumbnail not found", http.StatusNotFound)
		return
	}

	if thumb == nil {
		http.Error(w, "Thumbnail not available", http.StatusNotFound)
		return
	}

	// Generate presigned URL for thumbnail
	url, err := h.minio.GetPresignedDownloadURL(r.Context(), thumb.StorageKey, 15*time.Minute)
	if err != nil {
		http.Error(w, "Failed to generate thumbnail URL", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"thumbnail_url": url,
		"width":         strconv.Itoa(thumb.Width),
		"height":        strconv.Itoa(thumb.Height),
	})
}

// DeleteFile handles file deletion
func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	file, err := h.storage.GetFile(r.Context(), id)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Delete from MinIO
	if err := h.minio.Delete(r.Context(), file.StorageKey); err != nil {
		// Log error but continue
		fmt.Printf("Failed to delete from MinIO: %v\n", err)
	}

	// Soft delete from database
	if err := h.storage.DeleteFile(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListFiles lists files for a project
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id required", http.StatusBadRequest)
		return
	}

	pid := uuid.MustParse(projectID)
	files, err := h.storage.GetFilesByProject(r.Context(), pid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(files)
}

// GetUploadURL generates a presigned upload URL
func (h *Handler) GetUploadURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID    string `json:"tenant_id"`
		FileName    string `json:"file_name"`
		ContentType string `json:"content_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	storageKey := fmt.Sprintf("%s/%s/%s", req.TenantID, uuid.New().String(), req.FileName)

	url, err := h.minio.GetPresignedUploadURL(r.Context(), storageKey, 15*time.Minute)
	if err != nil {
		http.Error(w, "Failed to generate upload URL", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"upload_url": url,
		"storage_key": storageKey,
	})
}

// generateThumbnail generates a thumbnail for an image file
func (h *Handler) generateThumbnail(file *models.File) {
	ctx := context.Background()

	// Download file from MinIO
	reader, err := h.minio.Download(ctx, file.StorageKey)
	if err != nil {
		fmt.Printf("Failed to download file for thumbnail: %v\n", err)
		return
	}
	defer reader.Close()

	// Generate thumbnail
	data, width, height, err := h.thumbnail.Generate(reader)
	if err != nil {
		fmt.Printf("Failed to generate thumbnail: %v\n", err)
		return
	}

	// Upload thumbnail to MinIO
	thumbKey := fmt.Sprintf("thumbnails/%s/%s", file.TenantID, file.ID.String())
	if err := h.minio.Upload(ctx, thumbKey, bytes.NewReader(data), int64(len(data)), "image/jpeg"); err != nil {
		fmt.Printf("Failed to upload thumbnail: %v\n", err)
		return
	}

	// Save thumbnail record
	thumb := &models.Thumbnail{
		ID:         uuid.New(),
		FileID:     file.ID,
		Width:      width,
		Height:     height,
		Format:     "jpeg",
		Size:       int64(len(data)),
		StorageKey: thumbKey,
		CreatedAt:  time.Now(),
	}

	if err := h.storage.CreateThumbnail(ctx, thumb); err != nil {
		fmt.Printf("Failed to save thumbnail record: %v\n", err)
	}
}


