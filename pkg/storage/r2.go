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

// R2Client wraps the Cloudflare R2 S3-compatible client.
type R2Client struct {
	client     *s3.Client
	bucketName string
	publicURL  string
}

// R2Config holds all configuration needed to connect to a Cloudflare R2 bucket.
type R2Config struct {
	AccountID   string
	AccessKeyID string
	SecretKey   string
	BucketName  string
	PublicURL   string
}

// NewR2Client creates a new Cloudflare R2 storage client using the S3-compatible API.
// R2's endpoint is derived from the Cloudflare account ID.
func NewR2Client(config R2Config) (*R2Client, error) {
	if config.AccountID == "" || config.AccessKeyID == "" || config.SecretKey == "" {
		return nil, fmt.Errorf("R2 account ID, access key ID, and secret key are required")
	}
	if config.BucketName == "" {
		return nil, fmt.Errorf("R2 bucket name is required")
	}

	// R2 endpoint: https://<accountID>.r2.cloudflarestorage.com
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", config.AccountID)

	cfg := aws.Config{
		Region: "auto",
		Credentials: credentials.NewStaticCredentialsProvider(
			config.AccessKeyID,
			config.SecretKey,
			"",
		),
	}

	// Use BaseEndpoint (per-operation option) instead of the deprecated
	// EndpointResolverWithOptions on the aws.Config level.
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	log.Println("✓ R2 client initialized")
	return &R2Client{
		client:     s3Client,
		bucketName: config.BucketName,
		publicURL:  config.PublicURL,
	}, nil
}

// TestConnection verifies the R2 bucket is accessible by listing a single object.
// R2 does not support HeadBucket, so ListObjectsV2 is used instead.
func (r *R2Client) TestConnection(ctx context.Context) error {
	_, err := r.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(r.bucketName),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return fmt.Errorf("failed to access R2 bucket: %w", err)
	}
	return nil
}

// Upload stores a file in R2 at the given key and returns its public URL.
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
	return fmt.Sprintf("%s/%s", r.publicURL, key), nil
}

// GetPresignedURL generates a time-limited presigned URL for direct download.
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

// Delete removes a file from R2.
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

// List returns all object keys in the bucket that share the given prefix.
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
