package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/RodolfoBonis/spooliq/core/logger"
)

// CDNService handles file uploads to the CDN
type CDNService struct {
	baseURL    string
	apiKey     string
	bucket     string
	httpClient *http.Client
	logger     logger.Logger
}

// CDNUploadResponse represents the response from CDN upload
type CDNUploadResponse struct {
	Message string `json:"message"`
	URL     string `json:"url"`
}

// NewCDNService creates a new CDN service instance
func NewCDNService(baseURL, apiKey, bucket string, logger logger.Logger) *CDNService {
	return &CDNService{
		baseURL: baseURL,
		apiKey:  apiKey,
		bucket:  bucket,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}
}

// UploadFile uploads a file to the CDN
func (s *CDNService) UploadFile(ctx context.Context, file io.Reader, filename string, folder string) (string, error) {
	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file field
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		s.logger.Error(ctx, "Failed to create form file", map[string]interface{}{
			"error":    err.Error(),
			"filename": filename,
		})
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		s.logger.Error(ctx, "Failed to copy file content", map[string]interface{}{
			"error":    err.Error(),
			"filename": filename,
		})
		return "", fmt.Errorf("failed to copy file content: %w", err)
	}

	// Add folder field if provided
	if folder != "" {
		err = writer.WriteField("folder", folder)
		if err != nil {
			s.logger.Error(ctx, "Failed to add folder field", map[string]interface{}{
				"error":  err.Error(),
				"folder": folder,
			})
			return "", fmt.Errorf("failed to add folder field: %w", err)
		}
	}

	err = writer.Close()
	if err != nil {
		s.logger.Error(ctx, "Failed to close multipart writer", map[string]interface{}{
			"error": err.Error(),
		})
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	uploadURL := fmt.Sprintf("%s/v1/upload", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, body)
	if err != nil {
		s.logger.Error(ctx, "Failed to create upload request", map[string]interface{}{
			"error": err.Error(),
			"url":   uploadURL,
		})
		return "", fmt.Errorf("failed to create upload request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-KEY", s.apiKey)

	s.logger.Info(ctx, "Uploading file to CDN", map[string]interface{}{
		"filename": filename,
		"folder":   folder,
		"url":      uploadURL,
	})

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error(ctx, "Failed to execute upload request", map[string]interface{}{
			"error": err.Error(),
		})
		return "", fmt.Errorf("failed to execute upload request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error(ctx, "Failed to read response body", map[string]interface{}{
			"error": err.Error(),
		})
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		s.logger.Error(ctx, "CDN upload failed", map[string]interface{}{
			"status_code": resp.StatusCode,
			"response":    string(respBody),
		})
		return "", fmt.Errorf("CDN upload failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var uploadResp CDNUploadResponse
	err = json.Unmarshal(respBody, &uploadResp)
	if err != nil {
		s.logger.Error(ctx, "Failed to parse upload response", map[string]interface{}{
			"error":    err.Error(),
			"response": string(respBody),
		})
		return "", fmt.Errorf("failed to parse upload response: %w", err)
	}

	s.logger.Info(ctx, "File uploaded successfully to CDN", map[string]interface{}{
		"filename": filename,
		"url":      uploadResp.URL,
	})

	return uploadResp.URL, nil
}

// GetFileURL constructs the full URL for a file path
func (s *CDNService) GetFileURL(path string) string {
	return fmt.Sprintf("%s/v1/cdn/%s/%s", s.baseURL, s.bucket, path)
}

// DownloadFile downloads a file from the CDN with authentication
func (s *CDNService) DownloadFile(ctx context.Context, path string) ([]byte, error) {
	// Construct CDN URL
	fileURL := s.GetFileURL(path)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", fileURL, nil)
	if err != nil {
		s.logger.Error(ctx, "Failed to create download request", map[string]interface{}{
			"error": err.Error(),
			"url":   fileURL,
		})
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	// Set authentication header
	req.Header.Set("X-API-KEY", s.apiKey)

	s.logger.Info(ctx, "Downloading file from CDN", map[string]interface{}{
		"path": path,
		"url":  fileURL,
	})

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error(ctx, "Failed to execute download request", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to execute download request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error(ctx, "Failed to read download response", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to read download response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		s.logger.Error(ctx, "CDN download failed", map[string]interface{}{
			"status_code": resp.StatusCode,
			"response":    string(data),
		})
		return nil, fmt.Errorf("CDN download failed with status %d: %s", resp.StatusCode, string(data))
	}

	s.logger.Info(ctx, "File downloaded successfully from CDN", map[string]interface{}{
		"path": path,
		"size": len(data),
	})

	return data, nil
}
