package minio

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config holds the MinIO connection parameters, typically fetched from the DB settings table.
type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// Client wraps the official MinIO SDK client with convenience methods for
// backup upload/download operations.
type Client struct {
	mc     *minio.Client
	bucket string
}

// NewClient creates a new MinIO SDK client from the given Config.
// It does NOT auto-create the bucket; call EnsureBucket separately if needed.
func NewClient(cfg *Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("MinIO endpoint belum dikonfigurasi")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("MinIO bucket belum dikonfigurasi")
	}

	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("gagal membuat MinIO client: %w", err)
	}

	return &Client{mc: mc, bucket: cfg.Bucket}, nil
}

// Ping verifies the connection by checking whether the configured bucket exists.
func (c *Client) Ping(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("gagal terhubung ke MinIO: %w", err)
	}
	if !exists {
		return fmt.Errorf("bucket '%s' tidak ditemukan", c.bucket)
	}
	return nil
}

// EnsureBucket creates the bucket if it does not exist yet.
func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("gagal cek bucket: %w", err)
	}
	if !exists {
		if err := c.mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("gagal membuat bucket '%s': %w", c.bucket, err)
		}
	}
	return nil
}

// Upload puts an object into the configured bucket.
func (c *Client) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := c.mc.PutObject(ctx, c.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("gagal upload '%s' ke MinIO: %w", objectName, err)
	}
	return nil
}

// Download retrieves an object from the configured bucket.
// The caller is responsible for closing the returned reader.
func (c *Client) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	obj, err := c.mc.GetObject(ctx, c.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("gagal download '%s' dari MinIO: %w", objectName, err)
	}
	// Verify the object is accessible by reading its stat.
	if _, err := obj.Stat(); err != nil {
		obj.Close()
		return nil, fmt.Errorf("gagal mengakses '%s' di MinIO: %w", objectName, err)
	}
	return obj, nil
}

// ObjectInfo represents metadata for a single object in the bucket.
type ObjectInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
}

// List returns objects in the bucket filtered by the given prefix.
func (c *Client) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	var objects []ObjectInfo

	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	for obj := range c.mc.ListObjects(ctx, c.bucket, opts) {
		if obj.Err != nil {
			return nil, fmt.Errorf("gagal list objek di MinIO: %w", obj.Err)
		}
		objects = append(objects, ObjectInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
		})
	}
	return objects, nil
}

// Delete removes an object from the configured bucket.
func (c *Client) Delete(ctx context.Context, objectName string) error {
	if err := c.mc.RemoveObject(ctx, c.bucket, objectName, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("gagal hapus '%s' dari MinIO: %w", objectName, err)
	}
	return nil
}
