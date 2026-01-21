package s3client

import (
	"context"
	"crypto/rand"
	"testing"
)

const (
	bigFileBlockSize = 25 * 1024 * 1024 // 25MB
	bigFileCount     = 1
	bigFileLength    = bigFileBlockSize * bigFileCount
)

// generateTestData generates random test data of specified size
func generateTestData(size int64) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}

// TestMultipartUpload tests multi-part upload of large file
func TestMultipartUpload(t *testing.T) {
	client := NewClient("test-bucket", "us-east-1", nil)
	ctx := context.Background()

	if client.s3Client == nil {
		t.Skip("S3 client not initialized - skipping multipart upload test")
		return
	}

	// Generate large test data (>25MB to trigger multipart)
	testData := generateTestData(bigFileLength)
	testKey := "test-multipart-file.bin"

	// Upload using multipart
	err := client.PutObjectMultipart(ctx, testKey, testData)
	if err != nil {
		t.Fatalf("Failed to upload multipart: %v", err)
	}

	// Verify by downloading
	downloaded, err := client.GetObject(ctx, testKey)
	if err != nil {
		t.Fatalf("Failed to download file: %v", err)
	}

	if len(downloaded) != len(testData) {
		t.Errorf("Size mismatch: expected %d, got %d", len(testData), len(downloaded))
	}

	// Compare data
	for i := range testData {
		if testData[i] != downloaded[i] {
			t.Errorf("Data mismatch at offset %d", i)
			break
		}
	}

	// Cleanup
	client.DeleteObject(ctx, testKey)
}

// TestMultipartCopy tests multi-part copy operation
func TestMultipartCopy(t *testing.T) {
	client := NewClient("test-bucket", "us-east-1", nil)
	ctx := context.Background()

	if client.s3Client == nil {
		t.Skip("S3 client not initialized - skipping multipart copy test")
		return
	}

	sourceKey := "test-multipart-source.bin"
	destKey := "test-multipart-dest.bin"

	// Create source file
	testData := generateTestData(bigFileLength)
	err := client.PutObjectMultipart(ctx, sourceKey, testData)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy using multipart copy
	err = client.CopyObjectMultipart(ctx, sourceKey, destKey)
	if err != nil {
		t.Fatalf("Failed to copy multipart: %v", err)
	}

	// Verify destination
	downloaded, err := client.GetObject(ctx, destKey)
	if err != nil {
		t.Fatalf("Failed to download copied file: %v", err)
	}

	if len(downloaded) != len(testData) {
		t.Errorf("Size mismatch: expected %d, got %d", len(testData), len(downloaded))
	}

	// Cleanup
	client.DeleteObject(ctx, sourceKey)
	client.DeleteObject(ctx, destKey)
}

// TestMultipartMix tests mixed multipart operations (partial writes)
func TestMultipartMix(t *testing.T) {
	client := NewClient("test-bucket", "us-east-1", nil)
	ctx := context.Background()

	if client.s3Client == nil {
		t.Skip("S3 client not initialized - skipping multipart mix test")
		return
	}

	testKey := "test-multipart-mix.bin"
	testData := generateTestData(bigFileLength)

	// (1) Create initial file
	err := client.PutObjectMultipart(ctx, testKey, testData)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// (2) Modify middle of file (at 7.5MB offset)
	modifyOffset := int64(15 * 1024 * 1024 / 2)
	modifyData := []byte("0123456789ABCDEF")
	
	// Read existing, modify, write back
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

	// Verify modification
	downloaded, err := client.GetObject(ctx, testKey)
	if err != nil {
		t.Fatalf("Failed to verify: %v", err)
	}

	// Check modified section
	for i := 0; i < len(modifyData); i++ {
		if downloaded[modifyOffset+int64(i)] != modifyData[i] {
			t.Errorf("Modification not preserved at offset %d", modifyOffset+int64(i))
		}
	}

	// Cleanup
	client.DeleteObject(ctx, testKey)
}

// TestMultipartAbort tests aborting incomplete multipart upload
func TestMultipartAbort(t *testing.T) {
	client := NewClient("test-bucket", "us-east-1", nil)
	ctx := context.Background()

	if client.s3Client == nil {
		t.Skip("S3 client not initialized - skipping multipart abort test")
		return
	}

	testKey := "test-multipart-abort.bin"

	// Start multipart upload
	uploadID, err := client.CreateMultipartUpload(ctx, testKey)
	if err != nil {
		t.Fatalf("Failed to create multipart upload: %v", err)
	}

	// Abort it
	err = client.AbortMultipartUpload(ctx, testKey, uploadID)
	if err != nil {
		t.Fatalf("Failed to abort multipart upload: %v", err)
	}

	// Verify object doesn't exist
	_, err = client.GetObject(ctx, testKey)
	if err == nil {
		t.Error("Object should not exist after abort")
	}
}
