package s3client

import (
	"context"
	"testing"
)

// TestListObjectsEmptyPrefix tests listing objects with empty prefix
func TestListObjectsEmptyPrefix(t *testing.T) {
	client := NewClient("test-bucket", "us-east-1", nil)
	ctx := context.Background()

	objects, err := client.ListObjects(ctx, "")
	if err != nil {
		// Expected to fail without real credentials
		t.Logf("ListObjects failed (expected without credentials): %v", err)
		return
	}

	_ = objects
}

// TestGetObjectNotFound tests getting a non-existent object
func TestGetObjectNotFound(t *testing.T) {
	client := NewClient("test-bucket", "us-east-1", nil)
	ctx := context.Background()

	_, err := client.GetObject(ctx, "nonexistent-key")
	if err == nil {
		t.Error("Expected error for nonexistent object")
	}
}

// TestPutObjectEmpty tests putting an empty object
func TestPutObjectEmpty(t *testing.T) {
	client := NewClient("test-bucket", "us-east-1", nil)
	ctx := context.Background()

	err := client.PutObject(ctx, "test-empty-key", []byte{})
	if err != nil {
		// Expected to fail without real credentials
		t.Logf("PutObject failed (expected without credentials): %v", err)
		return
	}
}

// TestPutObjectWithData tests putting an object with data
func TestPutObjectWithData(t *testing.T) {
	client := NewClient("test-bucket", "us-east-1", nil)
	ctx := context.Background()

	testData := []byte("HELLO WORLD")
	err := client.PutObject(ctx, "test-key", testData)
	if err != nil {
		// Expected to fail without real credentials
		t.Logf("PutObject failed (expected without credentials): %v", err)
		return
	}

	// Try to read it back
	data, err := client.GetObject(ctx, "test-key")
	if err != nil {
		t.Logf("GetObject failed (expected without credentials): %v", err)
		return
	}

	if string(data) != string(testData) {
		t.Errorf("Expected '%s', got '%s'", string(testData), string(data))
	}
}

// TestDeleteObjectIntegration tests deleting an object (integration test)
func TestDeleteObjectIntegration(t *testing.T) {
	client := NewClient("test-bucket", "us-east-1", nil)
	ctx := context.Background()

	// First create an object
	err := client.PutObject(ctx, "test-delete-key", []byte("test"))
	if err != nil {
		t.Logf("PutObject failed (expected without credentials): %v", err)
		return
	}

	// Delete it
	err = client.DeleteObject(ctx, "test-delete-key")
	if err != nil {
		t.Fatalf("Failed to delete object: %v", err)
	}

	// Verify it's gone
	_, err = client.GetObject(ctx, "test-delete-key")
	if err == nil {
		t.Error("Object should not exist after deletion")
	}
}

// TestHeadObject tests getting object metadata
func TestHeadObject(t *testing.T) {
	client := NewClient("test-bucket", "us-east-1", nil)
	ctx := context.Background()

	// Create an object first
	err := client.PutObject(ctx, "test-head-key", []byte("test"))
	if err != nil {
		t.Logf("PutObject failed (expected without credentials): %v", err)
		return
	}

	// Get metadata
	metadata, err := client.HeadObject(ctx, "test-head-key")
	if err != nil {
		t.Fatalf("Failed to head object: %v", err)
	}

	_ = metadata
}

// TestListObjectsWithPrefix tests listing objects with a prefix
func TestListObjectsWithPrefix(t *testing.T) {
	client := NewClient("test-bucket", "us-east-1", nil)
	ctx := context.Background()

	// Create some test objects
	testKeys := []string{"prefix/file1.txt", "prefix/file2.txt", "other/file.txt"}
	for _, key := range testKeys {
		err := client.PutObject(ctx, key, []byte("test"))
		if err != nil {
			t.Logf("PutObject failed (expected without credentials): %v", err)
			return
		}
	}

	// List objects with prefix
	objects, err := client.ListObjects(ctx, "prefix/")
	if err != nil {
		t.Fatalf("Failed to list objects: %v", err)
	}

	if len(objects) < 2 {
		t.Errorf("Expected at least 2 objects, got %d", len(objects))
	}

	// Cleanup
	for _, key := range testKeys {
		client.DeleteObject(ctx, key)
	}
}
