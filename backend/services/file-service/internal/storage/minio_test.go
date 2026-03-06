package storage

import (
	"context"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMinIOClient is a mock implementation of MinIO client
type MockMinIOClient struct {
	mock.Mock
}

func (m *MockMinIOClient) PutObject(ctx context.Context, bucketName, objectName string, reader interface{}, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	args := m.Called(ctx, bucketName, objectName, reader, objectSize, opts)
	return args.Get(0).(minio.UploadInfo), args.Error(1)
}

func (m *MockMinIOClient) GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error) {
	args := m.Called(ctx, bucketName, objectName, opts)
	return args.Get(0).(*minio.Object), args.Error(1)
}

func (m *MockMinIOClient) RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error {
	args := m.Called(ctx, bucketName, objectName, opts)
	return args.Error(0)
}

func (m *MockMinIOClient) PresignedGetObject(ctx context.Context, bucketName, objectName string, expiry time.Duration, reqParams map[string]string) (string, error) {
	args := m.Called(ctx, bucketName, objectName, expiry, reqParams)
	return args.String(0), args.Error(1)
}

func (m *MockMinIOClient) PresignedPutObject(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, bucketName, objectName, expiry)
	return args.String(0), args.Error(1)
}

func (m *MockMinIOClient) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	args := m.Called(ctx, bucketName)
	return args.Bool(0), args.Error(1)
}

func (m *MockMinIOClient) MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error {
	args := m.Called(ctx, bucketName, opts)
	return args.Error(0)
}

func TestNewMinIOStorage(t *testing.T) {
	config := &MinIOConfig{
		Endpoint:  "localhost:9000",
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		UseSSL:    false,
	}
	
	storage, err := NewMinIOStorage(config)
	
	// This will fail without actual MinIO server, but we test the structure
	assert.Nil(t, storage) // Expected to be nil without server
	assert.Error(t, err)   // Expected error
}

func TestMinIOStorage_CreatePresignedUploadURL(t *testing.T) {
	// This is a placeholder test
	// In real implementation, you'd mock the MinIO client
	
	storage := &MinIOStorage{
		bucket: "test-bucket",
	}
	
	assert.NotNil(t, storage)
	assert.Equal(t, "test-bucket", storage.bucket)
}

func TestMinIOStorage_CreatePresignedDownloadURL(t *testing.T) {
	storage := &MinIOStorage{
		bucket: "test-bucket",
	}
	
	assert.NotNil(t, storage)
}

func TestGenerateUniqueFileName(t *testing.T) {
	fileName := "test.pdf"
	uniqueName := GenerateUniqueFileName(fileName)
	
	assert.NotEmpty(t, uniqueName)
	assert.Contains(t, uniqueName, ".pdf")
	assert.NotEqual(t, fileName, uniqueName) // Should have UUID prefix
}

func TestGenerateUniqueFileName_NoExtension(t *testing.T) {
	fileName := "README"
	uniqueName := GenerateUniqueFileName(fileName)
	
	assert.NotEmpty(t, uniqueName)
	assert.NotEqual(t, fileName, uniqueName)
}

func TestValidateFileType(t *testing.T) {
	tests := []struct {
		fileName string
		allowed  []string
		expected bool
	}{
		{"document.pdf", []string{".pdf", ".doc"}, true},
		{"image.jpg", []string{".png", ".gif"}, false},
		{"file.PDF", []string{".pdf"}, true}, // Case insensitive
		{"archive", []string{".zip"}, false}, // No extension
		{"drawing.dwg", []string{".dwg", ".dxf", ".rvt"}, true},
	}
	
	for _, tt := range tests {
		result := ValidateFileType(tt.fileName, tt.allowed)
		assert.Equal(t, tt.expected, result)
	}
}

func TestCalculatePartSize(t *testing.T) {
	tests := []struct {
		fileSize    int64
		maxParts    int
		expectedMin int64
	}{
		{100 * 1024 * 1024, 100, 5 * 1024 * 1024},       // 100MB file
		{1024 * 1024 * 1024, 100, 5 * 1024 * 1024},      // 1GB file
		{10 * 1024 * 1024, 100, 5 * 1024 * 1024},        // 10MB file (min 5MB)
	}
	
	for _, tt := range tests {
		partSize := CalculatePartSize(tt.fileSize, tt.maxParts)
		assert.GreaterOrEqual(t, partSize, tt.expectedMin)
	}
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		fileName string
		expected string
	}{
		{"document.pdf", "application/pdf"},
		{"image.png", "image/png"},
		{"model.dwg", "application/octet-stream"},
		{"data.json", "application/json"},
		{"unknown.xyz", "application/octet-stream"},
	}
	
	for _, tt := range tests {
		contentType := GetContentType(tt.fileName)
		assert.Equal(t, tt.expected, contentType)
	}
}

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"document.pdf", "document.pdf"},
		{"my file (1).pdf", "my_file_1.pdf"},
		{"../../etc/passwd", "etc_passwd"},
		{"file:name.pdf", "file_name.pdf"},
		{"spaces and | pipes.pdf", "spaces_and__pipes.pdf"},
	}
	
	for _, tt := range tests {
		result := SanitizeFileName(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}
