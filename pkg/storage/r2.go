package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2Client wraps the Cloudflare R2 S3-compatible client
type R2Client struct {
	client     *s3.Client
	bucketName string
	publicURL  string
}

// R2Config holds R2 configuration
type R2Config struct {
	AccountID     string
	AccessKeyID   string
	SecretKey     string
	BucketName    string
	PublicURL     string
}

// NewR2Client creates a new Cloudflare R2 storage client
func NewR2Client(config R2Config) (*R2Client, error) {
	if config.AccountID == "" || config.AccessKeyID == "" || config.SecretKey == "" {
		return nil, fmt.Errorf("R2 account ID, access key ID, and secret key are required")
	}

	if config.BucketName == "" {
		return nil, fmt.Errorf("R2 bucket name is required")
	}

	// Configure AWS SDK for R2
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", config.AccountID),
		}, nil
	})

	cfg := aws.Config{
		Region: "auto",
		Credentials: credentials.NewStaticCredentialsProvider(
			config.AccessKeyID,
			config.SecretKey,
			"",
		),
		EndpointResolverWithOptions: r2Resolver,
	}

	client := &R2Client{
		client:     s3.NewFromConfig(cfg),
		bucketName: config.BucketName,
		publicURL:  config.PublicURL,
	}

	log.Println("✓ R2 client initialized")
	return client, nil
}

// TestConnection verifies R2 bucket is accessible
func (r *R2Client) TestConnection(ctx context.Context) error {
	// Try to list objects (HeadBucket is not supported by R2)
	_, err := r.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(r.bucketName),
		MaxKeys: aws.Int32(1),
	})

	if err != nil {
		return fmt.Errorf("failed to access R2 bucket: %w", err)
	}

	return nil
}

// Upload uploads a file to R2
func (r *R2Client) Upload(ctx context.Context, key string, body io.Reader, contentType string) (string, error) {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload to R2: %w", err)
	}

	// Return public URL
	url := fmt.Sprintf("%s/%s", r.publicURL, key)
	return url, nil
}

// GetPresignedURL generates a presigned URL for temporary access
func (r *R2Client) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(r.client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return request.URL, nil
}

// Delete deletes a file from R2
func (r *R2Client) Delete(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete from R2: %w", err)
	}

	return nil
}

// List lists objects in the bucket with a prefix
func (r *R2Client) List(ctx context.Context, prefix string) ([]string, error) {
	output, err := r.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(r.bucketName),
		Prefix: aws.String(prefix),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	keys := make([]string, 0, len(output.Contents))
	for _, obj := range output.Contents {
		if obj.Key != nil {
			keys = append(keys, *obj.Key)
		}
	}

	return keys, nil
}
