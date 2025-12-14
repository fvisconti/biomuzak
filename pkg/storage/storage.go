package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ReadSeekCloser combines io.ReadSeeker and io.Closer
type ReadSeekCloser interface {
	io.ReadSeeker
	io.Closer
}

// StorageService defines the interface for object storage operations
type StorageService interface {
	UploadFile(ctx context.Context, objectName string, reader io.Reader, objectSize int64, contentType string) error
	GetFileStream(ctx context.Context, objectName string) (ReadSeekCloser, error)
	GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error)
}

// MinIOStorage implements StorageService using MinIO
type MinIOStorage struct {
	Client *minio.Client
	Bucket string
}

// NewMinIOStorage creates a new MinIOStorage instance
func NewMinIOStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinIOStorage, error) {
	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	// Check if bucket exists, create if not (although createbuckets service should handle this)
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, bucket)
	if err != nil {
		log.Printf("Warning: Failed to check if bucket exists: %v", err)
	} else if !exists {
		log.Printf("Bucket %s does not exist, attempting to create...", bucket)
		err = minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket %s: %w", bucket, err)
		}
		// Set policy to public read for simplicity in development (optional)
		policy := `{"Version": "2012-10-17","Statement": [{"Action": ["s3:GetObject"],"Effect": "Allow","Principal": {"AWS": ["*"]},"Resource": ["arn:aws:s3:::` + bucket + `/*"],"Sid": ""}]}`
		err = minioClient.SetBucketPolicy(ctx, bucket, policy)
		if err != nil {
			log.Printf("Warning: Failed to set bucket policy: %v", err)
		}
	}

	return &MinIOStorage{
		Client: minioClient,
		Bucket: bucket,
	}, nil
}

// UploadFile uploads a file to the configured bucket
func (s *MinIOStorage) UploadFile(ctx context.Context, objectName string, reader io.Reader, objectSize int64, contentType string) error {
	_, err := s.Client.PutObject(ctx, s.Bucket, objectName, reader, objectSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to minio: %w", err)
	}
	return nil
}

// GetFileStream retrieves a file as a stream
func (s *MinIOStorage) GetFileStream(ctx context.Context, objectName string) (ReadSeekCloser, error) {
	obj, err := s.Client.GetObject(ctx, s.Bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from minio: %w", err)
	}
	return obj, nil
}

// GetPresignedURL generates a presigned URL for temporary access
func (s *MinIOStorage) GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error) {
	reqParams := make(url.Values)
	presignedURL, err := s.Client.PresignedGetObject(ctx, s.Bucket, objectName, expires, reqParams)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned url: %w", err)
	}
	return presignedURL, nil
}
