package s3client

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const (
	multipartTestBlockSize = 6 * 1024 * 1024 // 6MB (above 5MB threshold)
	multipartTestCount     = 1
	multipartTestLength    = multipartTestBlockSize * multipartTestCount
)

// generateMultipartTestData generates random test data for multipart tests
func generateMultipartTestData(size int64) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}

// TestLocalStackMultipartUpload tests multi-part upload with LocalStack
func TestLocalStackMultipartUpload(t *testing.T) {
	client := setupLocalStackTest(t)
	ctx := context.Background()

	// Generate large test data (>5MB to trigger multipart)
	testData := generateMultipartTestData(multipartTestLength)
	testKey := fmt.Sprintf("test-multipart-%d.bin", time.Now().UnixNano())

	t.Logf("Uploading %d bytes using multipart upload", len(testData))

	// Upload using multipart
	err := client.PutObjectMultipart(ctx, testKey, testData)
	if err != nil {
		t.Fatalf("Failed to upload multipart: %v", err)
	}

	t.Logf("Successfully uploaded multipart file")

	// Verify by downloading
	downloaded, err := client.GetObject(ctx, testKey)
	if err != nil {
		t.Fatalf("Failed to download file: %v", err)
	}

	if len(downloaded) != len(testData) {
		t.Errorf("Size mismatch: expected %d, got %d", len(testData), len(downloaded))
	}

	// Compare data (sample check for performance)
	sampleSize := 1024
	if len(testData) > sampleSize {
		for i := 0; i < sampleSize; i++ {
			if testData[i] != downloaded[i] {
				t.Errorf("Data mismatch at offset %d", i)
				break
			}
		}
		// Check end
		for i := 0; i < sampleSize && i < len(testData); i++ {
			idx := len(testData) - sampleSize + i
			if idx >= 0 && idx < len(downloaded) {
				if testData[idx] != downloaded[idx] {
					t.Errorf("Data mismatch at end offset %d", idx)
					break
				}
			}
		}
	} else {
		// Full comparison for small files
		for i := range testData {
			if testData[i] != downloaded[i] {
				t.Errorf("Data mismatch at offset %d", i)
				break
			}
		}
	}

	t.Logf("Verified downloaded file matches uploaded file")

	// Cleanup
	err = client.DeleteObject(ctx, testKey)
	if err != nil {
		t.Logf("Warning: failed to cleanup test file: %v", err)
	}
}

// TestLocalStackMultipartCopy tests multi-part copy operation with LocalStack
func TestLocalStackMultipartCopy(t *testing.T) {
	client := setupLocalStackTest(t)
	ctx := context.Background()

	sourceKey := fmt.Sprintf("test-multipart-source-%d.bin", time.Now().UnixNano())
	destKey := fmt.Sprintf("test-multipart-dest-%d.bin", time.Now().UnixNano())

	// Create source file using multipart
	testData := generateMultipartTestData(multipartTestLength)
	err := client.PutObjectMultipart(ctx, sourceKey, testData)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	t.Logf("Created source file, copying using multipart copy")

	// Copy using multipart copy
	err = client.CopyObjectMultipart(ctx, sourceKey, destKey)
	if err != nil {
		t.Fatalf("Failed to copy multipart: %v", err)
	}

	t.Logf("Successfully copied file using multipart copy")

	// Verify destination
	downloaded, err := client.GetObject(ctx, destKey)
	if err != nil {
		t.Fatalf("Failed to download copied file: %v", err)
	}

	if len(downloaded) != len(testData) {
		t.Errorf("Size mismatch: expected %d, got %d", len(testData), len(downloaded))
	}

	// Sample verification
	sampleSize := 1024
	for i := 0; i < sampleSize && i < len(testData); i++ {
		if testData[i] != downloaded[i] {
			t.Errorf("Data mismatch at offset %d", i)
			break
		}
	}

	t.Logf("Verified copied file matches source")

	// Cleanup
	client.DeleteObject(ctx, sourceKey)
	client.DeleteObject(ctx, destKey)
}

// TestLocalStackMultipartMix tests mixed multipart operations with LocalStack
func TestLocalStackMultipartMix(t *testing.T) {
	client := setupLocalStackTest(t)
	ctx := context.Background()

	testKey := fmt.Sprintf("test-multipart-mix-%d.bin", time.Now().UnixNano())
	testData := generateMultipartTestData(multipartTestLength)

	// (1) Create initial file using multipart
	err := client.PutObjectMultipart(ctx, testKey, testData)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	t.Logf("Created initial file, modifying middle section")

	// (2) Modify middle of file
	modifyOffset := int64(len(testData) / 2)
	modifyData := []byte("0123456789ABCDEF")

	// Read existing
	existing, err := client.GetObject(ctx, testKey)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Modify in memory
	if modifyOffset+int64(len(modifyData)) <= int64(len(existing)) {
		copy(existing[modifyOffset:], modifyData)
	}

	// Write back using multipart
	err = client.PutObjectMultipart(ctx, testKey, existing)
	if err != nil {
		t.Fatalf("Failed to write modified file: %v", err)
	}

	t.Logf("Successfully modified file using multipart upload")

	// Verify modification
	downloaded, err := client.GetObject(ctx, testKey)
	if err != nil {
		t.Fatalf("Failed to verify: %v", err)
	}

	// Check modified section
	for i := 0; i < len(modifyData); i++ {
		idx := modifyOffset + int64(i)
		if idx < int64(len(downloaded)) {
			if downloaded[idx] != modifyData[i] {
				t.Errorf("Modification not preserved at offset %d: expected %c, got %c",
					idx, modifyData[i], downloaded[idx])
			}
		}
	}

	t.Logf("Verified modification was preserved")

	// Cleanup
	client.DeleteObject(ctx, testKey)
}

// TestLocalStackMultipartAbort tests aborting incomplete multipart upload with LocalStack
func TestLocalStackMultipartAbort(t *testing.T) {
	client := setupLocalStackTest(t)
	ctx := context.Background()

	testKey := fmt.Sprintf("test-multipart-abort-%d.bin", time.Now().UnixNano())

	t.Logf("Creating multipart upload to abort")

	// Start multipart upload
	uploadID, err := client.CreateMultipartUpload(ctx, testKey)
	if err != nil {
		t.Fatalf("Failed to create multipart upload: %v", err)
	}

	t.Logf("Created multipart upload with ID: %s", uploadID)

	// Abort it
	err = client.AbortMultipartUpload(ctx, testKey, uploadID)
	if err != nil {
		t.Fatalf("Failed to abort multipart upload: %v", err)
	}

	t.Logf("Successfully aborted multipart upload")

	// Verify object doesn't exist
	_, err = client.GetObject(ctx, testKey)
	if err == nil {
		t.Error("Object should not exist after abort")
	} else {
		t.Logf("Verified object does not exist after abort (expected error: %v)", err)
	}
}

// TestLocalStackMultipartManual tests manual multipart upload process with LocalStack
func TestLocalStackMultipartManual(t *testing.T) {
	client := setupLocalStackTest(t)
	ctx := context.Background()

	testKey := fmt.Sprintf("test-multipart-manual-%d.bin", time.Now().UnixNano())
	testData := generateMultipartTestData(multipartTestLength)

	// Split data into parts
	partSize := int64(DefaultPartSize)
	var parts []types.CompletedPart

	t.Logf("Starting manual multipart upload with %d bytes", len(testData))

	// Create multipart upload
	uploadID, err := client.CreateMultipartUpload(ctx, testKey)
	if err != nil {
		t.Fatalf("Failed to create multipart upload: %v", err)
	}

	t.Logf("Created multipart upload: %s", uploadID)

	// Upload parts manually
	totalParts := (int64(len(testData)) + partSize - 1) / partSize
	for i := int64(0); i < totalParts; i++ {
		start := i * partSize
		end := start + partSize
		if end > int64(len(testData)) {
			end = int64(len(testData))
		}

		partData := testData[start:end]
		etag, err := client.UploadPart(ctx, testKey, uploadID, int32(i+1), partData)
		if err != nil {
			client.AbortMultipartUpload(ctx, testKey, uploadID)
			t.Fatalf("Failed to upload part %d: %v", i+1, err)
		}

		parts = append(parts, types.CompletedPart{
			ETag:       &etag,
			PartNumber: aws.Int32(int32(i + 1)),
		})

		t.Logf("Uploaded part %d/%d (size: %d)", i+1, totalParts, len(partData))
	}

	// Complete multipart upload
	err = client.CompleteMultipartUpload(ctx, testKey, uploadID, parts)
	if err != nil {
		t.Fatalf("Failed to complete multipart upload: %v", err)
	}

	t.Logf("Completed multipart upload")

	// Verify file
	downloaded, err := client.GetObject(ctx, testKey)
	if err != nil {
		t.Fatalf("Failed to download file: %v", err)
	}

	if len(downloaded) != len(testData) {
		t.Errorf("Size mismatch: expected %d, got %d", len(testData), len(downloaded))
	}

	// Sample verification
	sampleSize := 1024
	for i := 0; i < sampleSize && i < len(testData); i++ {
		if testData[i] != downloaded[i] {
			t.Errorf("Data mismatch at offset %d", i)
			break
		}
	}

	t.Logf("Verified manually uploaded multipart file")

	// Cleanup
	client.DeleteObject(ctx, testKey)
}
