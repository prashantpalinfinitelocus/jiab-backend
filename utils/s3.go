package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"

	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
)

type S3Service interface {
	UploadFile(ctx context.Context, file *multipart.FileHeader, folder string) (string, error)
	DeleteFile(ctx context.Context, fileURL string) error
	UploadFileFromReader(ctx context.Context, file io.ReadCloser, folder string) (string, string, error)
}

type s3Service struct {
	bucketName string
	region     string
	client     *s3.Client
}

func NewS3Service(bucketName, region, accessKey, secretKey string) (S3Service, error) {
	// Create custom endpoint resolver with modern AWS SDK v2 patterns
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				URL:               fmt.Sprintf("https://s3.%s.amazonaws.com", region),
				HostnameImmutable: true,
				PartitionID:       "aws",
				SigningRegion:     region,
				Source:            aws.EndpointSourceCustom,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	// Load AWS configuration with modern options
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRetryMode(aws.RetryModeStandard),
		config.WithRetryMaxAttempts(3),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with modern options
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = false  // Use virtual-hosted style URLs
		o.UseAccelerate = false // Disable S3 Transfer Acceleration
		o.UseARNRegion = true   // Enable ARN region support
	})

	return &s3Service{
		bucketName: bucketName,
		region:     region,
		client:     client,
	}, nil
}

func (s *s3Service) UploadFile(ctx context.Context, file *multipart.FileHeader, folder string) (string, error) {
	// Open the file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Generate unique filename with folder prefix
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s/%s%s", folder, uuid.New().String(), ext)

	// Get content type with fallback
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Upload to S3 with modern options
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(filename),
		Body:        src,
		ContentType: aws.String(contentType),
		Metadata: map[string]string{
			"original-filename": file.Filename,
		},
	}

	_, err = s.client.PutObject(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			return "", fmt.Errorf("failed to upload file to S3 (code: %s): %w", apiErr.ErrorCode(), err)
		}
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	// Generate the public URL using the modern AWS SDK v2 URL format
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucketName, s.region, filename)
	return url, nil
}

func (s *s3Service) DeleteFile(ctx context.Context, fileURL string) error {
	// Extract key from URL with validation
	prefix := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/", s.bucketName, s.region)
	if len(fileURL) <= len(prefix) {
		return fmt.Errorf("invalid S3 URL format")
	}
	key := fileURL[len(prefix):]

	// Delete from S3 with modern error handling
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "NoSuchKey":
				return fmt.Errorf("file not found in S3: %w", err)
			case "AccessDenied":
				return fmt.Errorf("access denied to S3 bucket: %w", err)
			default:
				return fmt.Errorf("failed to delete file from S3 (code: %s): %w", apiErr.ErrorCode(), err)
			}
		}
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	return nil
}

type sizedReader struct {
	io.Reader
	size int64
}

func (sr *sizedReader) Size() int64 {
	return sr.size
}

func (s *s3Service) UploadFileFromReader(ctx context.Context, file io.ReadCloser, folder string) (string, string, error) {
	defer file.Close()

	// Read all data to get content length
	data, err := io.ReadAll(file)
	if err != nil {
		return "", "", fmt.Errorf("failed to read file data: %w", err)
	}

	// Generate unique filename with folder prefix
	filename := fmt.Sprintf("%s/%s", folder, uuid.New().String())

	// Create sized reader
	sizedData := &sizedReader{
		Reader: bytes.NewReader(data),
		size:   int64(len(data)),
	}

	// Upload to S3 with content length
	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.bucketName),
		Key:           aws.String(filename),
		Body:          sizedData,
		ContentType:   aws.String("application/octet-stream"),
		ContentLength: aws.Int64(sizedData.Size()),
	}

	_, err = s.client.PutObject(ctx, input)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			return "", "", errs.NewInternalServerError("failed to upload file to S3", err)
		}
		return "", "", errs.NewInternalServerError("failed to upload file to S3", err)
	}

	// Generate the public URL using the modern AWS SDK v2 URL format
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucketName, s.region, filename)
	return filename, url, nil
}
