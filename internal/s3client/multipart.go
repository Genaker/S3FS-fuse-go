package s3client

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const (
	// MinMultipartSize is the minimum size for multipart upload (5MB)
	MinMultipartSize = 5 * 1024 * 1024
	// DefaultPartSize is the default part size for multipart upload (5MB)
	DefaultPartSize = 5 * 1024 * 1024
)

// CreateMultipartUpload initiates a multipart upload
func (c *Client) CreateMultipartUpload(ctx context.Context, key string) (string, error) {
	if c.s3Client == nil {
		return "", fmt.Errorf("S3 client not initialized")
	}

	input := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	result, err := c.s3Client.CreateMultipartUpload(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to create multipart upload: %w", err)
	}

	if result.UploadId == nil {
		return "", fmt.Errorf("upload ID is nil")
	}

	return *result.UploadId, nil
}

// UploadPart uploads a single part of a multipart upload
func (c *Client) UploadPart(ctx context.Context, key, uploadID string, partNumber int32, data []byte) (string, error) {
	if c.s3Client == nil {
		return "", fmt.Errorf("S3 client not initialized")
	}

	input := &s3.UploadPartInput{
		Bucket:     aws.String(c.bucket),
		Key:        aws.String(key),
		PartNumber: aws.Int32(partNumber),
		UploadId:   aws.String(uploadID),
		Body:       bytes.NewReader(data),
	}

	result, err := c.s3Client.UploadPart(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload part %d: %w", partNumber, err)
	}

	if result.ETag == nil {
		return "", fmt.Errorf("ETag is nil for part %d", partNumber)
	}

	return *result.ETag, nil
}

// CompleteMultipartUpload completes a multipart upload
func (c *Client) CompleteMultipartUpload(ctx context.Context, key, uploadID string, parts []types.CompletedPart) error {
	if c.s3Client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	input := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(c.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: parts,
		},
	}

	_, err := c.s3Client.CompleteMultipartUpload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	return nil
}

// AbortMultipartUpload aborts a multipart upload
func (c *Client) AbortMultipartUpload(ctx context.Context, key, uploadID string) error {
	if c.s3Client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	input := &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(c.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	}

	_, err := c.s3Client.AbortMultipartUpload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}

	return nil
}

// PutObjectMultipart uploads an object using multipart upload for large files
func (c *Client) PutObjectMultipart(ctx context.Context, key string, data []byte) error {
	if c.s3Client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	// Use simple PutObject for small files
	if int64(len(data)) < MinMultipartSize {
		return c.PutObject(ctx, key, data)
	}

	// Initiate multipart upload
	uploadID, err := c.CreateMultipartUpload(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to create multipart upload: %w", err)
	}

	// Upload parts
	var parts []types.CompletedPart
	partSize := int64(DefaultPartSize)
	totalParts := (int64(len(data)) + partSize - 1) / partSize

	for i := int64(0); i < totalParts; i++ {
		start := i * partSize
		end := start + partSize
		if end > int64(len(data)) {
			end = int64(len(data))
		}

		partData := data[start:end]
		etag, err := c.UploadPart(ctx, key, uploadID, int32(i+1), partData)
		if err != nil {
			// Try to abort on error
			c.AbortMultipartUpload(ctx, key, uploadID)
			return fmt.Errorf("failed to upload part %d: %w", i+1, err)
		}

		parts = append(parts, types.CompletedPart{
			ETag:       aws.String(etag),
			PartNumber: aws.Int32(int32(i + 1)),
		})
	}

	// Complete multipart upload
	err = c.CompleteMultipartUpload(ctx, key, uploadID, parts)
	if err != nil {
		// Try to abort on error
		c.AbortMultipartUpload(ctx, key, uploadID)
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	return nil
}

// CopyPart copies a part from source object for multipart copy
func (c *Client) CopyPart(ctx context.Context, destKey, uploadID string, partNumber int32, sourceKey string, start, end int64) (string, error) {
	if c.s3Client == nil {
		return "", fmt.Errorf("S3 client not initialized")
	}

	copySource := fmt.Sprintf("%s/%s", c.bucket, sourceKey)
	input := &s3.UploadPartCopyInput{
		Bucket:          aws.String(c.bucket),
		Key:             aws.String(destKey),
		PartNumber:      aws.Int32(partNumber),
		UploadId:        aws.String(uploadID),
		CopySource:      aws.String(copySource),
		CopySourceRange: aws.String(fmt.Sprintf("bytes=%d-%d", start, end-1)),
	}

	result, err := c.s3Client.UploadPartCopy(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to copy part %d: %w", partNumber, err)
	}

	if result.CopyPartResult == nil || result.CopyPartResult.ETag == nil {
		return "", fmt.Errorf("ETag is nil for copied part %d", partNumber)
	}

	return *result.CopyPartResult.ETag, nil
}

// CopyObjectMultipart copies an object using multipart copy for large files
func (c *Client) CopyObjectMultipart(ctx context.Context, sourceKey, destKey string) error {
	if c.s3Client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	// Get source object size
	sourceSize, err := c.HeadObjectSize(ctx, sourceKey)
	if err != nil {
		return fmt.Errorf("failed to get source object size: %w", err)
	}

	// Use simple copy for small files
	if sourceSize < MinMultipartSize {
		data, err := c.GetObject(ctx, sourceKey)
		if err != nil {
			return fmt.Errorf("failed to read source object: %w", err)
		}
		return c.PutObject(ctx, destKey, data)
	}

	// Initiate multipart upload
	uploadID, err := c.CreateMultipartUpload(ctx, destKey)
	if err != nil {
		return fmt.Errorf("failed to create multipart upload: %w", err)
	}

	// Copy parts
	var parts []types.CompletedPart
	partSize := int64(DefaultPartSize)
	totalParts := (sourceSize + partSize - 1) / partSize

	for i := int64(0); i < totalParts; i++ {
		start := i * partSize
		end := start + partSize
		if end > sourceSize {
			end = sourceSize
		}

		etag, err := c.CopyPart(ctx, destKey, uploadID, int32(i+1), sourceKey, start, end)
		if err != nil {
			// Try to abort on error
			c.AbortMultipartUpload(ctx, destKey, uploadID)
			return fmt.Errorf("failed to copy part %d: %w", i+1, err)
		}

		parts = append(parts, types.CompletedPart{
			ETag:       aws.String(etag),
			PartNumber: aws.Int32(int32(i + 1)),
		})
	}

	// Complete multipart upload
	err = c.CompleteMultipartUpload(ctx, destKey, uploadID, parts)
	if err != nil {
		// Try to abort on error
		c.AbortMultipartUpload(ctx, destKey, uploadID)
		return fmt.Errorf("failed to complete multipart copy: %w", err)
	}

	return nil
}
