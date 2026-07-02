package services

import (
	"context"
	"fmt"
	"io"
	"mime"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// CDNService uploads/serves files directly via MinIO + the new cdn edge (replaces the rb-cdn proxy).
type CDNService struct {
	minio         *minio.Client
	bucket        string
	publicBaseURL string
	logger        logger.Logger
}

// NewCDNService creates a new CDN service instance. publicBaseURL is the cdn edge that serves the
// bucket (e.g. https://assets.spooliq.com). MinIO creds come from keys (Vault k3s/spooliq/minio).
func NewCDNService(publicBaseURL string, keys entities.CdnKeysEntity, log logger.Logger) *CDNService {
	s := &CDNService{
		bucket:        keys.Bucket,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
		logger:        log,
	}
	if keys.Endpoint == "" || keys.AccessKey == "" || keys.SecretKey == "" {
		return s // minio nil: methods error gracefully (keeps boot working without creds)
	}
	mc, err := minio.New(keys.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(keys.AccessKey, keys.SecretKey, ""),
		Secure: keys.UseSSL,
	})
	if err == nil {
		s.minio = mc
	}
	return s
}

// UploadFile puts file into the bucket under {folder}/{filename} and returns the public cdn URL.
func (s *CDNService) UploadFile(ctx context.Context, file io.Reader, filename string, folder string) (string, error) {
	if s.minio == nil {
		return "", fmt.Errorf("cdn: MinIO client not configured")
	}
	key := path.Join(folder, filename)

	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	uctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if _, err := s.minio.PutObject(uctx, s.bucket, key, file, -1, minio.PutObjectOptions{
		ContentType: contentType,
	}); err != nil {
		s.logger.Error(ctx, "Failed to upload file to MinIO", map[string]interface{}{
			"error": err.Error(), "key": key,
		})
		return "", fmt.Errorf("cdn: put object: %w", err)
	}

	url := s.publicBaseURL + "/" + key
	s.logger.Info(ctx, "File uploaded to MinIO", map[string]interface{}{"key": key, "url": url})
	return url, nil
}

// GetFileURL constructs the public served URL for an object key.
func (s *CDNService) GetFileURL(objectKey string) string {
	return s.publicBaseURL + "/" + strings.TrimPrefix(objectKey, "/")
}

// DownloadFile fetches an object's bytes. pathOrURL may be a bare object key or a full URL (old
// rb-cdn proxy `…/v1/cdn/<bucket>/<key>` or the new edge `https://<host>/<key>`); the key is derived.
func (s *CDNService) DownloadFile(ctx context.Context, pathOrURL string) ([]byte, error) {
	if s.minio == nil {
		return nil, fmt.Errorf("cdn: MinIO client not configured")
	}
	key := s.objectKey(pathOrURL)

	uctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	obj, err := s.minio.GetObject(uctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("cdn: get object: %w", err)
	}
	defer func() { _ = obj.Close() }()

	data, err := io.ReadAll(obj)
	if err != nil {
		s.logger.Error(ctx, "Failed to download file from MinIO", map[string]interface{}{
			"error": err.Error(), "key": key,
		})
		return nil, fmt.Errorf("cdn: read object: %w", err)
	}
	return data, nil
}

// DeleteFile removes an object from the bucket. pathOrURL may be a bare object key or a full
// URL (old rb-cdn proxy or new edge form); the key is derived like DownloadFile.
func (s *CDNService) DeleteFile(ctx context.Context, pathOrURL string) error {
	if s.minio == nil {
		return fmt.Errorf("cdn: MinIO client not configured")
	}
	key := s.objectKey(pathOrURL)

	uctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if err := s.minio.RemoveObject(uctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		s.logger.Error(ctx, "Failed to delete file from MinIO", map[string]interface{}{
			"error": err.Error(), "key": key,
		})
		return fmt.Errorf("cdn: remove object: %w", err)
	}
	s.logger.Info(ctx, "File deleted from MinIO", map[string]interface{}{"key": key})
	return nil
}

// objectKey derives the bucket-relative key from a bare key or any stored URL form.
func (s *CDNService) objectKey(pathOrURL string) string {
	// Old rb-cdn proxy form: <host>/v1/cdn/<bucket>/<key>
	if i := strings.Index(pathOrURL, "/v1/cdn/"); i >= 0 {
		rest := pathOrURL[i+len("/v1/cdn/"):]
		if j := strings.Index(rest, "/"); j >= 0 {
			return rest[j+1:] // drop the bucket segment
		}
		return rest
	}
	// New edge form: https://<host>/<key>
	if i := strings.Index(pathOrURL, "://"); i >= 0 {
		rest := pathOrURL[i+3:]
		if j := strings.Index(rest, "/"); j >= 0 {
			return rest[j+1:]
		}
	}
	return strings.TrimPrefix(pathOrURL, "/")
}
