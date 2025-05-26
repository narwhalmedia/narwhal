package transcode

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"

	"github.com/narwhalmedia/narwhal/internal/domain/transcode"
)

type LocalStorage struct {
	basePath string
	logger   *zap.Logger
}

func NewLocalStorage(basePath string, logger *zap.Logger) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	return &LocalStorage{
		basePath: basePath,
		logger:   logger,
	}, nil
}

func (s *LocalStorage) Store(ctx context.Context, key string, reader io.Reader) error {
	path := filepath.Join(s.basePath, key)
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (s *LocalStorage) Retrieve(ctx context.Context, key string) (io.ReadCloser, error) {
	path := filepath.Join(s.basePath, key)
	
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, transcode.ErrStorageKeyNotFound
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	path := filepath.Join(s.basePath, key)
	
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return transcode.ErrStorageKeyNotFound
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (s *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	path := filepath.Join(s.basePath, key)
	
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *LocalStorage) GetURL(ctx context.Context, key string) (string, error) {
	if exists, err := s.Exists(ctx, key); err != nil {
		return "", err
	} else if !exists {
		return "", transcode.ErrStorageKeyNotFound
	}

	return fmt.Sprintf("file://%s", filepath.Join(s.basePath, key)), nil
}

type S3Storage struct {
	client   *s3.Client
	bucket   string
	prefix   string
	region   string
	logger   *zap.Logger
}

func NewS3Storage(bucket, prefix, region string, logger *zap.Logger) (*S3Storage, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	return &S3Storage{
		client: client,
		bucket: bucket,
		prefix: prefix,
		region: region,
		logger: logger,
	}, nil
}

func (s *S3Storage) Store(ctx context.Context, key string, reader io.Reader) error {
	fullKey := s.getFullKey(key)

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullKey),
		Body:   reader,
	})

	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

func (s *S3Storage) Retrieve(ctx context.Context, key string) (io.ReadCloser, error) {
	fullKey := s.getFullKey(key)

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullKey),
	})

	if err != nil {
		// TODO: Check if key not found error
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}

	return result.Body, nil
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	fullKey := s.getFullKey(key)

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullKey),
	})

	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}

func (s *S3Storage) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := s.getFullKey(key)

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fullKey),
	})

	if err != nil {
		// TODO: Check if not found error
		return false, nil
	}

	return true, nil
}

func (s *S3Storage) GetURL(ctx context.Context, key string) (string, error) {
	fullKey := s.getFullKey(key)
	
	if exists, err := s.Exists(ctx, key); err != nil {
		return "", err
	} else if !exists {
		return "", transcode.ErrStorageKeyNotFound
	}

	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", 
		s.bucket, s.region, fullKey), nil
}

func (s *S3Storage) getFullKey(key string) string {
	if s.prefix != "" {
		return filepath.Join(s.prefix, key)
	}
	return key
}