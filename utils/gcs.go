package utils

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"

	"github.com/google/uuid"
)

type GCSService interface {
	UploadFile(ctx context.Context, file *multipart.FileHeader, path string) (string, string, error)
	DeleteFile(ctx context.Context, fileURL string) error
	UploadFileFromReader(ctx context.Context, file io.ReadCloser, folder string) (string, string, error)
	// UploadFileFromReaderWithOptions is UploadFileFromReader but with an
	// explicit object filename and content type, so the uploaded file opens
	// correctly as that type (e.g. a .csv export) instead of the generic
	// randomly-named, application/octet-stream object UploadFileFromReader
	// produces.
	UploadFileFromReaderWithOptions(ctx context.Context, file io.ReadCloser, folder, filename, contentType string) (string, string, error)
	UploadFileFromBytes(ctx context.Context, data []byte, path string, contentType string) (string, string, error)
	GetFileSignedURL(ctx context.Context, objectPath string, expiry time.Duration) (string, error)
	ReadFileContent(ctx context.Context, objectPath string) (string, error)
}

type gcsService struct {
	bucketName string
	client     *storage.Client
	projectID  string
}

func NewGCSService(bucketName, projectID string) (GCSService, error) {
	ctx := context.Background()

	client, err := storage.NewClient(ctx) // No manual credentials
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	return &gcsService{
		bucketName: bucketName,
		client:     client,
		projectID:  projectID,
	}, nil
}

func (s *gcsService) UploadFile(ctx context.Context, file *multipart.FileHeader, path string) (string, string, error) {
	src, err := file.Open()
	if err != nil {
		return "", "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s/%s", path, ext)

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	writer := s.client.Bucket(s.bucketName).Object(filename).NewWriter(ctx)
	writer.ContentType = contentType
	writer.Metadata = map[string]string{
		"original-filename": file.Filename,
	}
	writer.CacheControl = "public, max-age=86400"

	if _, err := io.Copy(writer, src); err != nil {
		return "", "", fmt.Errorf("failed to write to GCS: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", "", fmt.Errorf("failed to close GCS writer: %w", err)
	}

	url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", s.bucketName, filename)
	return url, filename, nil
}

func (s *gcsService) UploadFileFromReader(ctx context.Context, file io.ReadCloser, path string) (string, string, error) {
	defer file.Close()

	filename := fmt.Sprintf("%s/%s", path, uuid.New().String())

	writer := s.client.Bucket(s.bucketName).Object(filename).NewWriter(ctx)
	writer.ContentType = "application/octet-stream"
	writer.CacheControl = "public, max-age=86400"

	if _, err := io.Copy(writer, file); err != nil {
		return "", "", fmt.Errorf("failed to write to GCS: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", "", fmt.Errorf("failed to close GCS writer: %w", err)
	}

	url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", s.bucketName, filename)
	return url, filename, nil
}

func (s *gcsService) UploadFileFromReaderWithOptions(ctx context.Context, file io.ReadCloser, folder, filename, contentType string) (string, string, error) {
	defer file.Close()

	objectPath := fmt.Sprintf("%s/%s", folder, filename)

	writer := s.client.Bucket(s.bucketName).Object(objectPath).NewWriter(ctx)
	writer.ContentType = contentType
	writer.ContentDisposition = fmt.Sprintf("attachment; filename=%q", filename)
	writer.CacheControl = "public, max-age=86400"

	if _, err := io.Copy(writer, file); err != nil {
		return "", "", fmt.Errorf("failed to write to GCS: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", "", fmt.Errorf("failed to close GCS writer: %w", err)
	}

	url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", s.bucketName, objectPath)
	return url, objectPath, nil
}

func (s *gcsService) UploadFileFromBytes(ctx context.Context, data []byte, path string, contentType string) (string, string, error) {
	writer := s.client.Bucket(s.bucketName).Object(path).NewWriter(ctx)
	writer.ContentType = contentType
	writer.CacheControl = "public, max-age=86400"

	if _, err := writer.Write(data); err != nil {
		return "", "", fmt.Errorf("failed to write to GCS: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", "", fmt.Errorf("failed to close GCS writer: %w", err)
	}

	url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", s.bucketName, path)
	return url, path, nil
}

func (s *gcsService) DeleteFile(ctx context.Context, fileURL string) error {
	// GCS URL pattern: https://storage.googleapis.com/<bucket>/<key>
	prefix := fmt.Sprintf("https://storage.googleapis.com/%s/", s.bucketName)
	if !strings.HasPrefix(fileURL, prefix) {
		return fmt.Errorf("invalid GCS URL format")
	}
	key := strings.TrimPrefix(fileURL, prefix)

	object := s.client.Bucket(s.bucketName).Object(key)
	if err := object.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete GCS object: %w", err)
	}

	return nil
}

func (s *gcsService) GetFileSignedURL(ctx context.Context, objectPath string, expiry time.Duration) (string, error) {
	// Generate a V4 signed URL for the object (private, time-limited)
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(expiry),
	}
	url, err := s.client.Bucket(s.bucketName).SignedURL(objectPath, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}
	return url, nil
}

func (s *gcsService) ReadFileContent(ctx context.Context, objectPath string) (string, error) {
	// Extract object key from the full URL if it's a URL
	if strings.HasPrefix(objectPath, "https://storage.googleapis.com/") {
		prefix := fmt.Sprintf("https://storage.googleapis.com/%s/", s.bucketName)
		objectPath = strings.TrimPrefix(objectPath, prefix)
	}

	// Get the object from GCS
	object := s.client.Bucket(s.bucketName).Object(objectPath)
	reader, err := object.NewReader(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create reader for GCS object: %w", err)
	}
	defer reader.Close()

	// Read the content
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read GCS object content: %w", err)
	}

	return string(content), nil
}
