package media

import (
	"context"
	"fmt"
	"mime"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/princekumarofficial/stories-service/internal/config"
)

type Service struct {
	client     *minio.Client
	bucketName string
	config     *config.Media
	useSSL     bool
}

type UploadInfo struct {
	ObjectKey   string `json:"object_key"`
	UploadURL   string `json:"upload_url"`
	ExpiresAt   int64  `json:"expires_at"`
	MaxFileSize int64  `json:"max_file_size"`
	ContentType string `json:"content_type"`
}

// NewService creates a new media service instance
func NewService(cfg *config.Config) (*Service, error) {
	// Initialize MinIO client
	client, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKeyID, cfg.MinIO.SecretAccessKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	service := &Service{
		client:     client,
		bucketName: cfg.MinIO.BucketName,
		config:     &cfg.Media,
		useSSL:     cfg.MinIO.UseSSL,
	}

	// Ensure bucket exists
	if err := service.ensureBucket(); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	return service, nil
}

// ensureBucket creates the bucket if it doesn't exist
func (s *Service) ensureBucket() error {
	ctx := context.Background()

	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		err = s.client.MakeBucket(ctx, s.bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return nil
}

// ValidateContentType checks if the content type is allowed
func (s *Service) ValidateContentType(contentType string) bool {
	for _, allowed := range s.config.AllowedMimeTypes {
		if contentType == allowed {
			return true
		}
	}
	return false
}

// GenerateObjectKey creates a unique object key for the file
func (s *Service) GenerateObjectKey(userID string, contentType string) string {
	// Extract file extension from content type
	extensions, err := mime.ExtensionsByType(contentType)
	var ext string
	if err == nil && len(extensions) > 0 {
		ext = extensions[0]
	} else {
		// Fallback based on content type
		switch contentType {
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/gif":
			ext = ".gif"
		case "video/mp4":
			ext = ".mp4"
		case "video/mpeg":
			ext = ".mpeg"
		default:
			ext = ""
		}
	}

	// Generate unique filename
	filename := uuid.New().String() + ext

	// Create object key with user-based folder structure
	return fmt.Sprintf("users/%s/media/%s", userID, filename)
}

// GeneratePresignedUploadURL creates a presigned URL for uploading
func (s *Service) GeneratePresignedUploadURL(userID string, contentType string) (*UploadInfo, error) {
	// Validate content type
	if !s.ValidateContentType(contentType) {
		return nil, fmt.Errorf("content type %s is not allowed", contentType)
	}

	// Generate object key
	objectKey := s.GenerateObjectKey(userID, contentType)

	// Create presigned URL for upload
	expiry := time.Duration(s.config.PresignedURLTTL) * time.Second

	presignedURL, err := s.client.PresignedPutObject(
		context.Background(),
		s.bucketName,
		objectKey,
		expiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return &UploadInfo{
		ObjectKey:   objectKey,
		UploadURL:   presignedURL.String(),
		ExpiresAt:   time.Now().Add(expiry).Unix(),
		MaxFileSize: s.config.MaxFileSize,
		ContentType: contentType,
	}, nil
}

// GeneratePresignedDownloadURL creates a presigned URL for downloading
func (s *Service) GeneratePresignedDownloadURL(objectKey string, expiry time.Duration) (*url.URL, error) {
	return s.client.PresignedGetObject(
		context.Background(),
		s.bucketName,
		objectKey,
		expiry,
		nil,
	)
}

// GetMediaURL returns the public URL for accessing media (if bucket is public)
func (s *Service) GetMediaURL(objectKey string) string {
	// For development with MinIO, construct the direct URL
	// In production, you might want to use CDN URLs
	scheme := "http"
	if s.useSSL {
		scheme = "https"
	}

	endpoint := strings.TrimPrefix(s.client.EndpointURL().String(), scheme+"://")
	return fmt.Sprintf("%s://%s/%s/%s", scheme, endpoint, s.bucketName, objectKey)
}

// DeleteObject removes an object from storage
func (s *Service) DeleteObject(objectKey string) error {
	return s.client.RemoveObject(
		context.Background(),
		s.bucketName,
		objectKey,
		minio.RemoveObjectOptions{},
	)
}

// GetObjectInfo returns information about an object
func (s *Service) GetObjectInfo(objectKey string) (minio.ObjectInfo, error) {
	return s.client.StatObject(
		context.Background(),
		s.bucketName,
		objectKey,
		minio.StatObjectOptions{},
	)
}

// ListUserMedia lists all media files for a specific user
func (s *Service) ListUserMedia(userID string) ([]minio.ObjectInfo, error) {
	prefix := fmt.Sprintf("users/%s/media/", userID)

	var objects []minio.ObjectInfo
	objectsCh := s.client.ListObjects(
		context.Background(),
		s.bucketName,
		minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: true,
		},
	)

	for object := range objectsCh {
		if object.Err != nil {
			return nil, object.Err
		}
		objects = append(objects, object)
	}

	return objects, nil
}
