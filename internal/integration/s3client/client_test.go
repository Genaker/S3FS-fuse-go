//go:build integration

package s3client

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/s3fs-fuse/s3fs-go/internal/integration"
)

// TestPutGet tests putting and getting objects
func TestPutGet(t *testing.T) {
	client := integration.SetupTestClient(t, integration.LocalStackBucket, integration.LocalStackRegion)
	ctx := context.Background()

	testKey := fmt.Sprintf("test-put-get-%d", time.Now().UnixNano())
	testData := []byte("Hello from integration test!")

	// Put object
	err := client.PutObject(ctx, testKey, testData)
	if err != nil {
		t.Fatalf("PutObject failed: %v", err)
	}

	// Get object
	data, err := client.GetObject(ctx, testKey)
	if err != nil {
		t.Fatalf("GetObject failed: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("Expected '%s', got '%s'", string(testData), string(data))
	}

	// Cleanup
	client.DeleteObject(ctx, testKey)
}

// TestListObjects tests listing objects
func TestListObjects(t *testing.T) {
	client := integration.SetupTestClient(t, integration.LocalStackBucket, integration.LocalStackRegion)
	ctx := context.Background()

	prefix := fmt.Sprintf("test-list-%d", time.Now().UnixNano())

	// Create multiple objects
	testKeys := []string{
		prefix + "/file1.txt",
		prefix + "/file2.txt",
		prefix + "/file3.txt",
	}

	for _, key := range testKeys {
		err := client.PutObject(ctx, key, []byte("test data"))
		if err != nil {
			t.Fatalf("Failed to put object %s: %v", key, err)
		}
	}

	// List objects with prefix
	objects, err := client.ListObjects(ctx, prefix+"/")
	if err != nil {
		t.Fatalf("ListObjects failed: %v", err)
	}

	if len(objects) < len(testKeys) {
		t.Errorf("Expected at least %d objects, got %d", len(testKeys), len(objects))
	}

	// Cleanup
	for _, key := range testKeys {
		client.DeleteObject(ctx, key)
	}
}

// TestDeleteObject tests deleting objects
func TestDeleteObject(t *testing.T) {
	client := integration.SetupTestClient(t, integration.LocalStackBucket, integration.LocalStackRegion)
	ctx := context.Background()

	testKey := fmt.Sprintf("test-delete-%d", time.Now().UnixNano())
	testData := []byte("Test data")

	// Put object
	err := client.PutObject(ctx, testKey, testData)
	if err != nil {
		t.Fatalf("PutObject failed: %v", err)
	}

	// Delete object
	err = client.DeleteObject(ctx, testKey)
	if err != nil {
		t.Fatalf("DeleteObject failed: %v", err)
	}

	// Verify object doesn't exist
	_, err = client.GetObject(ctx, testKey)
	if err == nil {
		t.Error("Object should not exist after deletion")
	}
}

// TestHeadObject tests getting object metadata
func TestHeadObject(t *testing.T) {
	client := integration.SetupTestClient(t, integration.LocalStackBucket, integration.LocalStackRegion)
	ctx := context.Background()

	testKey := fmt.Sprintf("test-head-%d", time.Now().UnixNano())
	testData := []byte("Test data")

	// Put object
	err := client.PutObject(ctx, testKey, testData)
	if err != nil {
		t.Fatalf("PutObject failed: %v", err)
	}

	// Get metadata
	metadata, err := client.HeadObject(ctx, testKey)
	if err != nil {
		t.Fatalf("HeadObject failed: %v", err)
	}

	if metadata == nil {
		t.Error("Metadata should not be nil")
	}

	// Cleanup
	client.DeleteObject(ctx, testKey)
}

// TestGetObjectRange tests range reads
func TestGetObjectRange(t *testing.T) {
	client := integration.SetupTestClient(t, integration.LocalStackBucket, integration.LocalStackRegion)
	ctx := context.Background()

	testKey := fmt.Sprintf("test-range-%d", time.Now().UnixNano())
	testData := []byte("Hello, World!")

	// Put object
	err := client.PutObject(ctx, testKey, testData)
	if err != nil {
		t.Fatalf("PutObject failed: %v", err)
	}

	// Read range
	data, err := client.GetObjectRange(ctx, testKey, 0, 5)
	if err != nil {
		t.Fatalf("GetObjectRange failed: %v", err)
	}

	expected := testData[0:6] // 0-5 inclusive
	if string(data) != string(expected) {
		t.Errorf("Expected '%s', got '%s'", string(expected), string(data))
	}

	// Cleanup
	client.DeleteObject(ctx, testKey)
}

// TestPutObjectWithMetadata tests putting objects with metadata
func TestPutObjectWithMetadata(t *testing.T) {
	client := integration.SetupTestClient(t, integration.LocalStackBucket, integration.LocalStackRegion)
	ctx := context.Background()

	testKey := fmt.Sprintf("test-metadata-%d", time.Now().UnixNano())
	testData := []byte("Test data")
	metadata := map[string]string{
		"x-amz-meta-test": "test-value",
		"x-amz-meta-mode":  "0644",
	}

	// Put object with metadata
	err := client.PutObjectWithMetadata(ctx, testKey, testData, metadata)
	if err != nil {
		t.Fatalf("PutObjectWithMetadata failed: %v", err)
	}

	// Get metadata
	retrievedMetadata, err := client.HeadObject(ctx, testKey)
	if err != nil {
		t.Fatalf("HeadObject failed: %v", err)
	}

	if retrievedMetadata["x-amz-meta-test"] != "test-value" {
		t.Errorf("Expected metadata 'test-value', got '%s'", retrievedMetadata["x-amz-meta-test"])
	}

	// Cleanup
	client.DeleteObject(ctx, testKey)
}
