// Package noc provides NOC operations tooling.
// K-NOC-BR-008 — MinIO Object Store Client: S3-compatible object operations for backup artifacts.
// Uses github.com/minio/minio-go/v7 which is present in go.mod.
package noc

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOObject represents a single object in MinIO.
type MinIOObject struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
}

// MinIOClient wraps the minio-go client for backup artifact management.
type MinIOClient struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	client    *minio.Client
}

// NewMinIOClient reads MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY, and MINIO_BUCKET
// from the environment and creates a connected client.
func NewMinIOClient() (*MinIOClient, error) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9000"
	}
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	bucket := os.Getenv("MINIO_BUCKET")
	if bucket == "" {
		bucket = "kubric-backups"
	}

	useSSL := os.Getenv("MINIO_USE_SSL") == "true"
	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &MinIOClient{
		Endpoint:  endpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    bucket,
		client:    mc,
	}, nil
}

// UploadFile uploads a local file to MinIO at the given object key.
func (m *MinIOClient) UploadFile(ctx context.Context, localPath, objectKey string) error {
	_, err := m.client.FPutObject(ctx, m.Bucket, objectKey, localPath, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("minio upload %q -> %q: %w", localPath, objectKey, err)
	}
	return nil
}

// DownloadFile downloads an object from MinIO to a local file path.
func (m *MinIOClient) DownloadFile(ctx context.Context, objectKey, localPath string) error {
	if err := m.client.FGetObject(ctx, m.Bucket, objectKey, localPath, minio.GetObjectOptions{}); err != nil {
		return fmt.Errorf("minio download %q -> %q: %w", objectKey, localPath, err)
	}
	return nil
}

// ListObjects returns all objects with the given prefix.
func (m *MinIOClient) ListObjects(ctx context.Context, prefix string) ([]MinIOObject, error) {
	var objects []MinIOObject
	for obj := range m.client.ListObjects(ctx, m.Bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if obj.Err != nil {
			return nil, fmt.Errorf("list objects: %w", obj.Err)
		}
		objects = append(objects, MinIOObject{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
			ETag:         obj.ETag,
		})
	}
	return objects, nil
}

// DeleteObject removes the given object key from the bucket.
func (m *MinIOClient) DeleteObject(ctx context.Context, objectKey string) error {
	if err := m.client.RemoveObject(ctx, m.Bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("minio delete %q: %w", objectKey, err)
	}
	return nil
}

// BucketExists returns true if the bucket exists and is accessible.
func (m *MinIOClient) BucketExists(ctx context.Context, bucket string) (bool, error) {
	exists, err := m.client.BucketExists(ctx, bucket)
	if err != nil {
		return false, fmt.Errorf("check bucket exists %q: %w", bucket, err)
	}
	return exists, nil
}

// CreateBucket creates the bucket if it does not already exist.
func (m *MinIOClient) CreateBucket(ctx context.Context, bucket string) error {
	exists, err := m.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if err := m.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("minio create bucket %q: %w", bucket, err)
	}
	return nil
}

// GetPresignedURL generates a time-limited presigned download URL for an object.
func (m *MinIOClient) GetPresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	u, err := m.client.PresignedGetObject(ctx, m.Bucket, objectKey, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("presign %q: %w", objectKey, err)
	}
	return u.String(), nil
}

// StreamObject reads an object directly into the provided writer (useful for large files).
func (m *MinIOClient) StreamObject(ctx context.Context, objectKey string, w io.Writer) error {
	obj, err := m.client.GetObject(ctx, m.Bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("get object %q: %w", objectKey, err)
	}
	defer obj.Close()
	if _, err := io.Copy(w, obj); err != nil {
		return fmt.Errorf("stream object %q: %w", objectKey, err)
	}
	return nil
}
