// Package storage provides an S3-compatible object store client backed by MinIO.
// It is used across Kubric for evidence bundles (GRC), SBOM artifacts (supply chain),
// scan reports (SOC), and asset backups (NOC/DR).
//
// MinIO is declared in docker-compose.yml (minio/minio, ports 9050/9051).
package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ObjectStore wraps a MinIO client with Kubric-specific helpers.
type ObjectStore struct {
	client *minio.Client
}

// ObjectMeta describes a stored object.
type ObjectMeta struct {
	Bucket       string    `json:"bucket"`
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"content_type"`
	ETag         string    `json:"etag"`
	LastModified time.Time `json:"last_modified"`
}

// New connects to a MinIO/S3 endpoint.
// endpoint: "minio:9000", accessKey/secretKey from env or Vault.
// useSSL should be false for internal docker-compose networking.
func New(endpoint, accessKey, secretKey string, useSSL bool) (*ObjectStore, error) {
	if endpoint == "" {
		return nil, nil // disabled — caller checks nil
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	return &ObjectStore{client: client}, nil
}

// Close is a no-op for MinIO (HTTP client), but satisfies the integration pattern.
func (s *ObjectStore) Close() {}

// EnsureBucket creates the bucket if it doesn't exist.
func (s *ObjectStore) EnsureBucket(ctx context.Context, bucket, region string) error {
	exists, err := s.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("minio bucket check: %w", err)
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: region})
}

// PutObject uploads data to bucket/key with the specified content type.
func (s *ObjectStore) PutObject(ctx context.Context, bucket, key string, data []byte, contentType string) (*ObjectMeta, error) {
	reader := bytes.NewReader(data)
	info, err := s.client.PutObject(ctx, bucket, key, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, fmt.Errorf("minio put %s/%s: %w", bucket, key, err)
	}
	return &ObjectMeta{
		Bucket:       bucket,
		Key:          info.Key,
		Size:         info.Size,
		ETag:         info.ETag,
		LastModified: time.Now().UTC(),
	}, nil
}

// GetObject downloads the object at bucket/key.
func (s *ObjectStore) GetObject(ctx context.Context, bucket, key string) ([]byte, *ObjectMeta, error) {
	obj, err := s.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("minio get %s/%s: %w", bucket, key, err)
	}
	defer obj.Close()
	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, nil, fmt.Errorf("minio read %s/%s: %w", bucket, key, err)
	}
	stat, err := obj.Stat()
	if err != nil {
		return data, nil, fmt.Errorf("minio stat %s/%s: %w", bucket, key, err)
	}
	return data, &ObjectMeta{
		Bucket:       bucket,
		Key:          stat.Key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		ETag:         stat.ETag,
		LastModified: stat.LastModified,
	}, nil
}

// DeleteObject removes the object at bucket/key.
func (s *ObjectStore) DeleteObject(ctx context.Context, bucket, key string) error {
	return s.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}

// ListObjects returns metadata for all objects in bucket matching the prefix.
func (s *ObjectStore) ListObjects(ctx context.Context, bucket, prefix string) ([]ObjectMeta, error) {
	var out []ObjectMeta
	for obj := range s.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if obj.Err != nil {
			return out, fmt.Errorf("minio list %s/%s: %w", bucket, prefix, obj.Err)
		}
		out = append(out, ObjectMeta{
			Bucket:       bucket,
			Key:          obj.Key,
			Size:         obj.Size,
			ContentType:  obj.ContentType,
			ETag:         obj.ETag,
			LastModified: obj.LastModified,
		})
	}
	return out, nil
}

// PresignedGet generates a pre-signed GET URL valid for the given duration.
func (s *ObjectStore) PresignedGet(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, bucket, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("minio presign %s/%s: %w", bucket, key, err)
	}
	return url.String(), nil
}

// ---- Kubric-specific bucket constants ──────────────────────────────────────

const (
	BucketEvidence = "kubric-evidence"   // GRC evidence bundles
	BucketSBOM     = "kubric-sbom"       // CycloneDX / SPDX SBOM artifacts
	BucketScans    = "kubric-scans"      // SOC scan reports
	BucketBackups  = "kubric-backups"    // DR backup snapshots
)

// EnsureDefaultBuckets creates all standard Kubric buckets if they don't exist.
func (s *ObjectStore) EnsureDefaultBuckets(ctx context.Context) error {
	for _, b := range []string{BucketEvidence, BucketSBOM, BucketScans, BucketBackups} {
		if err := s.EnsureBucket(ctx, b, "us-east-1"); err != nil {
			return fmt.Errorf("ensure bucket %s: %w", b, err)
		}
	}
	return nil
}
