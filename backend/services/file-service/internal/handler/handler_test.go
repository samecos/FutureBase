package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockStorage implements storage interface for testing
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Upload(ctx interface{}, file interface{}, filename string) (string, error) {
	args := m.Called(ctx, file, filename)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) Download(ctx interface{}, fileID string) (interface{}, error) {
	args := m.Called(ctx, fileID)
	return args.Get(0), args.Error(1)
}

func (m *MockStorage) Delete(ctx interface{}, fileID string) error {
	args := m.Called(ctx, fileID)
	return args.Error(0)
}

func (m *MockStorage) GetPresignedURL(ctx interface{}, fileID string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, fileID, expiry)
	return args.String(0), args.Error(1)
}

func TestFileHandler_UploadFile(t *testing.T) {
	mockStorage := new(MockStorage)
	handler := NewFileHandler(mockStorage)

	// Create multipart form data
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("test content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/files/upload", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()

	mockStorage.On("Upload", mock.Anything, mock.Anything, mock.Anything).Return("file-123", nil)

	handler.UploadFile(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var response map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &response)
	assert.Equal(t, "file-123", response["fileId"])
}

func TestFileHandler_GetFile(t *testing.T) {
	mockStorage := new(MockStorage)
	handler := NewFileHandler(mockStorage)

	fileID := uuid.New().String()
	req := httptest.NewRequest("GET", "/api/v1/files/"+fileID, nil)
	rr := httptest.NewRecorder()

	mockStorage.On("GetPresignedURL", mock.Anything, fileID, mock.Anything).Return("http://presigned-url", nil)

	handler.GetFile(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &response)
	assert.Equal(t, "http://presigned-url", response["url"])
}

func TestFileHandler_DeleteFile(t *testing.T) {
	mockStorage := new(MockStorage)
	handler := NewFileHandler(mockStorage)

	fileID := uuid.New().String()
	req := httptest.NewRequest("DELETE", "/api/v1/files/"+fileID, nil)
	rr := httptest.NewRecorder()

	mockStorage.On("Delete", mock.Anything, fileID).Return(nil)

	handler.DeleteFile(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestFileHandler_GetUploadURL(t *testing.T) {
	mockStorage := new(MockStorage)
	handler := NewFileHandler(mockStorage)

	reqBody := map[string]interface{}{
		"filename": "test.txt",
		"contentType": "text/plain",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/files/upload-url", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	mockStorage.On("GetPresignedURL", mock.Anything, mock.Anything, mock.Anything).Return("http://upload-url", nil)

	handler.GetUploadURL(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &response)
	assert.Equal(t, "http://upload-url", response["uploadUrl"])
}

func TestFileHandler_ValidateFileType(t *testing.T) {
	tests := []struct {
		filename string
		allowed  []string
		expected bool
	}{
		{"document.pdf", []string{".pdf", ".doc"}, true},
		{"image.jpg", []string{".png", ".gif"}, false},
		{"file.PDF", []string{".pdf"}, true},
		{"noextension", []string{".txt"}, false},
	}

	for _, tt := range tests {
		result := ValidateFileType(tt.filename, tt.allowed)
		assert.Equal(t, tt.expected, result)
	}
}

func TestFileHandler_CalculatePartSize(t *testing.T) {
	tests := []struct {
		fileSize int64
		expected int64
	}{
		{100 * 1024 * 1024, 5 * 1024 * 1024},       // 100MB -> 5MB parts
		{1024 * 1024 * 1024, 10 * 1024 * 1024},     // 1GB -> 10MB parts
		{10 * 1024 * 1024, 5 * 1024 * 1024},        // 10MB -> minimum 5MB
	}

	for _, tt := range tests {
		partSize := CalculatePartSize(tt.fileSize, 100)
		assert.GreaterOrEqual(t, partSize, tt.expected)
	}
}

func TestFileHandler_GenerateUniqueFileName(t *testing.T) {
	filename := "document.pdf"
	uniqueName := GenerateUniqueFileName(filename)

	assert.NotEmpty(t, uniqueName)
	assert.Contains(t, uniqueName, ".pdf")
	assert.NotEqual(t, filename, uniqueName)
}
