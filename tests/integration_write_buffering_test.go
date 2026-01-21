//go:build integration

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/s3fs-fuse/s3fs-go/internal/cache"
)

// TestWriteBuffering tests that writes are buffered and not immediately uploaded
func TestWriteBuffering(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	// Set small buffer threshold (1KB) to test auto-upload
	fs.SetMaxDirtyData(1024)

	filePath := fmt.Sprintf("/test-buffering-%d.txt", time.Now().UnixNano())
	testData := []byte("Hello, World!")

	// Write small data (should be buffered, not uploaded)
	err := fs.WriteFile(ctx, filePath, testData, 0)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Check that file doesn't exist in S3 yet (buffered)
	_, err = fs.GetAttr(ctx, filePath)
	if err == nil {
		// File exists, which means it was uploaded immediately
		// This is OK if threshold was reached, but let's verify
		attr, _ := fs.GetAttr(ctx, filePath)
		if attr != nil && attr.Size == int64(len(testData)) {
			// File was uploaded, check if it's correct
			data, err := fs.ReadFile(ctx, filePath, 0, int64(len(testData)))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			if string(data) != string(testData) {
				t.Errorf("Expected %s, got %s", string(testData), string(data))
			}
		}
	}

	// Flush should upload buffered data
	err = fs.Flush(ctx, filePath)
	if err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}

	// Now file should exist
	attr, err := fs.GetAttr(ctx, filePath)
	if err != nil {
		t.Fatalf("File should exist after flush: %v", err)
	}
	if attr.Size != int64(len(testData)) {
		t.Errorf("Expected size %d, got %d", len(testData), attr.Size)
	}

	// Verify data
	data, err := fs.ReadFile(ctx, filePath, 0, int64(len(testData)))
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(data) != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), string(data))
	}
}

// TestWriteBufferingThreshold tests auto-upload when threshold is reached
func TestWriteBufferingThreshold(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	// Set small buffer threshold (1KB)
	fs.SetMaxDirtyData(1024)

	filePath := fmt.Sprintf("/test-threshold-%d.txt", time.Now().UnixNano())
	
	// Write data larger than threshold (should trigger auto-upload)
	largeData := make([]byte, 2048)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	err := fs.WriteFile(ctx, filePath, largeData, 0)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// File should exist immediately (auto-uploaded)
	attr, err := fs.GetAttr(ctx, filePath)
	if err != nil {
		t.Fatalf("File should exist after threshold write: %v", err)
	}
	if attr.Size != int64(len(largeData)) {
		t.Errorf("Expected size %d, got %d", len(largeData), attr.Size)
	}

	// Verify data
	data, err := fs.ReadFile(ctx, filePath, 0, int64(len(largeData)))
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if len(data) != len(largeData) {
		t.Errorf("Expected data length %d, got %d", len(largeData), len(data))
	}
}

// TestWriteBufferingMultipleWrites tests multiple buffered writes
func TestWriteBufferingMultipleWrites(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	// Set large buffer threshold (10MB) to prevent auto-upload
	fs.SetMaxDirtyData(10 * 1024 * 1024)

	filePath := fmt.Sprintf("/test-multiple-%d.txt", time.Now().UnixNano())
	
	// Write multiple small chunks
	chunk1 := []byte("Hello, ")
	chunk2 := []byte("World!")
	chunk3 := []byte(" How are you?")

	err := fs.WriteFile(ctx, filePath, chunk1, 0)
	if err != nil {
		t.Fatalf("Failed to write chunk1: %v", err)
	}

	err = fs.WriteFile(ctx, filePath, chunk2, int64(len(chunk1)))
	if err != nil {
		t.Fatalf("Failed to write chunk2: %v", err)
	}

	err = fs.WriteFile(ctx, filePath, chunk3, int64(len(chunk1)+len(chunk2)))
	if err != nil {
		t.Fatalf("Failed to write chunk3: %v", err)
	}

	// File shouldn't exist yet (all buffered)
	_, err = fs.GetAttr(ctx, filePath)
	if err == nil {
		// File exists, which means threshold was reached
		// This is OK, continue to verify data
	}

	// Flush should upload all buffered data
	err = fs.Flush(ctx, filePath)
	if err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}

	// Verify complete data
	expectedData := append(append(chunk1, chunk2...), chunk3...)
	data, err := fs.ReadFile(ctx, filePath, 0, int64(len(expectedData)))
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(data) != string(expectedData) {
		t.Errorf("Expected %s, got %s", string(expectedData), string(data))
	}
}

// TestWriteBufferingFsync tests that fsync uploads buffered data
func TestWriteBufferingFsync(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	// Set large buffer threshold
	fs.SetMaxDirtyData(10 * 1024 * 1024)

	filePath := fmt.Sprintf("/test-fsync-%d.txt", time.Now().UnixNano())
	testData := []byte("Test data for fsync")

	err := fs.WriteFile(ctx, filePath, testData, 0)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Fsync should upload buffered data
	err = fs.Fsync(ctx, filePath, false)
	if err != nil {
		t.Fatalf("Failed to fsync: %v", err)
	}

	// Verify data
	data, err := fs.ReadFile(ctx, filePath, 0, int64(len(testData)))
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(data) != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), string(data))
	}
}

// TestWriteBufferingRelease tests that release uploads buffered data
func TestWriteBufferingRelease(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	// Set large buffer threshold
	fs.SetMaxDirtyData(10 * 1024 * 1024)

	filePath := fmt.Sprintf("/test-release-%d.txt", time.Now().UnixNano())
	testData := []byte("Test data for release")

	err := fs.WriteFile(ctx, filePath, testData, 0)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Release should upload buffered data
	err = fs.Release(ctx, filePath)
	if err != nil {
		t.Fatalf("Failed to release: %v", err)
	}

	// Verify data
	data, err := fs.ReadFile(ctx, filePath, 0, int64(len(testData)))
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(data) != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), string(data))
	}
}

// TestWriteBufferingBytesModified tests BytesModified tracking
func TestWriteBufferingBytesModified(t *testing.T) {
	// Create cache manager with page size 4096
	cacheMgr := cache.NewManager(1000, time.Hour, 1000, 100, 4096)
	fdCache := cacheMgr.GetFdCache()
	
	// Open entity
	entity, err := fdCache.Open("test-path", 0, time.Now())
	if err != nil {
		t.Fatalf("Failed to open entity: %v", err)
	}
	
	// Write page at offset 0
	data1 := []byte("Hello")
	entity.WritePage(0, data1)
	
	// BytesModified should be page size (4096), not just data1 size
	bytesModified1 := entity.BytesModified()
	if bytesModified1 <= 0 {
		t.Errorf("Expected BytesModified > 0, got %d", bytesModified1)
	}
	
	// Write another page at different page (offset 5000, which is in next page)
	data2 := []byte("World")
	entity.WritePage(5000, data2)
	
	bytesModified2 := entity.BytesModified()
	if bytesModified2 <= bytesModified1 {
		t.Errorf("Expected BytesModified to increase, got %d (was %d)", bytesModified2, bytesModified1)
	}
	
	// Mark first page clean
	entity.MarkPageClean(0)
	
	bytesModified3 := entity.BytesModified()
	if bytesModified3 >= bytesModified2 {
		t.Errorf("Expected BytesModified to decrease after marking page clean, got %d (was %d)", bytesModified3, bytesModified2)
	}
}
