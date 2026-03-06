package minio

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/archplatform/file-service/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Client wraps MinIO client
type Client struct {
	client *minio.Client
	bucket string
}

// NewClient creates a new MinIO client
func NewClient(cfg config.MinIOConfig) (*Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// Create bucket if it doesn't exist
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &Client{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

// Upload uploads a file to MinIO
func (c *Client) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := c.client.PutObject(ctx, c.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

// Download downloads a file from MinIO
func (c *Client) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	object, err := c.client.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return object, nil
}

// Delete deletes a file from MinIO
func (c *Client) Delete(ctx context.Context, key string) error {
	return c.client.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
}

// GetPresignedUploadURL generates a presigned URL for upload
func (c *Client) GetPresignedUploadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	url, err := c.client.PresignedPutObject(ctx, c.bucket, key, expiry)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

// GetPresignedDownloadURL generates a presigned URL for download
func (c *Client) GetPresignedDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	reqParams := make(map[string]string)
	url, err := c.client.PresignedGetObject(ctx, c.bucket, key, expiry, reqParams)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

// Stat gets object info
func (c *Client) Stat(ctx context.Context, key string) (minio.ObjectInfo, error) {
	return c.client.StatObject(ctx, c.bucket, key, minio.StatObjectOptions{})
}
