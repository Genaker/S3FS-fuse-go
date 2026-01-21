package s3client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/s3fs-fuse/s3fs-go/internal/credentials"
)

// Client represents an S3 client
type Client struct {
	bucket   string
	region   string
	endpoint string
	creds    *credentials.Credentials
	s3Client *s3.Client
}

// NewClient creates a new S3 client
func NewClient(bucket, region string, creds *credentials.Credentials) *Client {
	return NewClientWithEndpoint(bucket, region, "", creds)
}

// NewClientWithEndpoint creates a new S3 client with custom endpoint
func NewClientWithEndpoint(bucket, region, endpoint string, creds *credentials.Credentials) *Client {
	client := &Client{
		bucket:   bucket,
		region:   region,
		endpoint: endpoint,
		creds:    creds,
	}

	// Initialize AWS SDK client
	if creds != nil && creds.IsValid() {
		cfgOptions := []func(*config.LoadOptions) error{
			config.WithRegion(region),
			config.WithCredentialsProvider(awscreds.NewStaticCredentialsProvider(
				creds.AccessKeyID,
				creds.SecretAccessKey,
				creds.SessionToken,
			)),
		}

		cfg, err := config.LoadDefaultConfig(context.Background(), cfgOptions...)
		if err == nil {
			s3Options := []func(*s3.Options){}
			if endpoint != "" {
				s3Options = append(s3Options, func(o *s3.Options) {
					o.BaseEndpoint = aws.String(endpoint)
					o.UsePathStyle = true // Required for LocalStack
				})
			}
			client.s3Client = s3.NewFromConfig(cfg, s3Options...)
		}
	}

	return client
}

// ListObjects lists objects with the given prefix
func (c *Client) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	if c.s3Client == nil {
		return nil, fmt.Errorf("S3 client not initialized")
	}

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	}

	result, err := c.s3Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	keys := make([]string, 0, len(result.Contents))
	for _, obj := range result.Contents {
		if obj.Key != nil {
			keys = append(keys, *obj.Key)
		}
	}

	return keys, nil
}

// GetObject retrieves an object from S3
func (c *Client) GetObject(ctx context.Context, key string) ([]byte, error) {
	return c.GetObjectRange(ctx, key, 0, 0)
}

// GetObjectRange retrieves an object from S3 with optional range
// If start and end are both 0, retrieves the entire object
// If end is 0, retrieves from start to end of object
func (c *Client) GetObjectRange(ctx context.Context, key string, start, end int64) ([]byte, error) {
	if c.s3Client == nil {
		return nil, fmt.Errorf("S3 client not initialized")
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	// Add range header if specified
	if start > 0 || end > 0 {
		var rangeHeader string
		if end > 0 {
			rangeHeader = fmt.Sprintf("bytes=%d-%d", start, end)
		} else {
			rangeHeader = fmt.Sprintf("bytes=%d-", start)
		}
		input.Range = aws.String(rangeHeader)
	}

	result, err := c.s3Client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object body: %w", err)
	}

	return data, nil
}

// PutObject uploads an object to S3
func (c *Client) PutObject(ctx context.Context, key string, data []byte) error {
	return c.PutObjectWithMetadata(ctx, key, data, nil)
}

// PutObjectWithMetadata uploads an object to S3 with metadata
func (c *Client) PutObjectWithMetadata(ctx context.Context, key string, data []byte, metadata map[string]string) error {
	if c.s3Client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	// AWS SDK expects metadata keys WITHOUT "x-amz-meta-" prefix
	// It adds the prefix automatically
	cleanMetadata := make(map[string]string)
	const metaPrefix = "x-amz-meta-"
	for k, v := range metadata {
		// Remove "x-amz-meta-" prefix if present
		key := k
		if strings.HasPrefix(k, metaPrefix) {
			key = k[len(metaPrefix):] // Remove prefix
		}
		cleanMetadata[key] = v
	}

	input := &s3.PutObjectInput{
		Bucket:   aws.String(c.bucket),
		Key:      aws.String(key),
		Body:     bytes.NewReader(data),
		Metadata: cleanMetadata,
	}

	_, err := c.s3Client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}

	return nil
}

// CopyObjectWithMetadata copies an object with updated metadata
func (c *Client) CopyObjectWithMetadata(ctx context.Context, sourceKey, destKey string, metadata map[string]string) error {
	if c.s3Client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	// AWS SDK expects metadata keys WITHOUT "x-amz-meta-" prefix
	// It adds the prefix automatically
	cleanMetadata := make(map[string]string)
	const metaPrefix = "x-amz-meta-"
	for k, v := range metadata {
		// Remove "x-amz-meta-" prefix if present
		key := k
		if strings.HasPrefix(k, metaPrefix) {
			key = k[len(metaPrefix):] // Remove prefix
		}
		cleanMetadata[key] = v
	}

	copySource := fmt.Sprintf("%s/%s", c.bucket, sourceKey)
	input := &s3.CopyObjectInput{
		Bucket:            aws.String(c.bucket),
		Key:               aws.String(destKey),
		CopySource:        aws.String(copySource),
		Metadata:          cleanMetadata,
		MetadataDirective: types.MetadataDirectiveReplace,
	}

	_, err := c.s3Client.CopyObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to copy object with metadata: %w", err)
	}

	return nil
}

// DeleteObject deletes an object from S3
func (c *Client) DeleteObject(ctx context.Context, key string) error {
	if c.s3Client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	_, err := c.s3Client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// HeadObject retrieves object metadata
func (c *Client) HeadObject(ctx context.Context, key string) (map[string]string, error) {
	if c.s3Client == nil {
		return nil, fmt.Errorf("S3 client not initialized")
	}

	input := &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	result, err := c.s3Client.HeadObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to head object: %w", err)
	}

	metadata := make(map[string]string)
	if result.Metadata != nil {
		for k, v := range result.Metadata {
			metadata[k] = v
		}
	}

	return metadata, nil
}

// HeadObjectSize retrieves object size from metadata without downloading
func (c *Client) HeadObjectSize(ctx context.Context, key string) (int64, error) {
	if c.s3Client == nil {
		return 0, fmt.Errorf("S3 client not initialized")
	}

	input := &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	result, err := c.s3Client.HeadObject(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("failed to head object: %w", err)
	}

	if result.ContentLength != nil {
		return *result.ContentLength, nil
	}

	return 0, nil
}

// CreateBucket creates an S3 bucket
func (c *Client) CreateBucket(ctx context.Context) error {
	if c.s3Client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	input := &s3.CreateBucketInput{
		Bucket: aws.String(c.bucket),
	}

	_, err := c.s3Client.CreateBucket(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}
