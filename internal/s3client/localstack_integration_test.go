package s3client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/s3fs-fuse/s3fs-go/internal/credentials"
)

const (
	localstackEndpoint = "http://localhost:4566"
	localstackBucket   = "test-bucket-localstack"
	localstackRegion   = "us-east-1"
)

// isLocalStackAvailable checks if LocalStack is running
func isLocalStackAvailable() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(localstackEndpoint + "/_localstack/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// setupLocalStackTest sets up LocalStack test environment
func setupLocalStackTest(t *testing.T) *Client {
	if !isLocalStackAvailable() {
		t.Skip("LocalStack is not available. Start it with: docker-compose -f docker-compose.localstack.yml up -d")
	}

	// Create credentials for LocalStack (dummy credentials)
	creds := credentials.NewCredentials()
	creds.AccessKeyID = "test"
	creds.SecretAccessKey = "test"

	// Create client with LocalStack endpoint
	client := NewClientWithEndpoint(localstackBucket, localstackRegion, localstackEndpoint, creds)

	// Create bucket if it doesn't exist
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to list objects - if bucket doesn't exist, create it
	_, err := client.ListObjects(ctx, "")
	if err != nil {
		// Bucket doesn't exist, create it
		err = client.CreateBucket(ctx)
		if err != nil {
			// Ignore bucket already exists errors
			if !strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") &&
				!strings.Contains(err.Error(), "BucketAlreadyExists") {
				t.Fatalf("Failed to create bucket: %v", err)
			}
		}
		// Wait a bit for bucket to be ready
		time.Sleep(500 * time.Millisecond)
	}

	return client
}

// TestLocalStackPutGet tests putting and getting objects with LocalStack
func TestLocalStackPutGet(t *testing.T) {
	client := setupLocalStackTest(t)
	ctx := context.Background()

	testKey := fmt.Sprintf("test-put-get-%d", time.Now().UnixNano())
	testData := []byte("Hello from LocalStack integration test!")

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

// TestLocalStackListObjects tests listing objects with LocalStack
func TestLocalStackListObjects(t *testing.T) {
	client := setupLocalStackTest(t)
	ctx := context.Background()

	prefix := fmt.Sprintf("test-list-%d/", time.Now().UnixNano())
	testKeys := []string{
		prefix + "file1.txt",
		prefix + "file2.txt",
		prefix + "file3.txt",
	}

	// Create test files
	for _, key := range testKeys {
		err := client.PutObject(ctx, key, []byte("test content"))
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", key, err)
		}
	}

	// List objects with prefix
	objects, err := client.ListObjects(ctx, prefix)
	if err != nil {
		t.Fatalf("ListObjects failed: %v", err)
	}

	if len(objects) != len(testKeys) {
		t.Errorf("Expected %d objects, got %d", len(testKeys), len(objects))
	}

	// Verify all keys are present
	keyMap := make(map[string]bool)
	for _, obj := range objects {
		keyMap[obj] = true
	}

	for _, key := range testKeys {
		if !keyMap[key] {
			t.Errorf("Expected key '%s' not found in list", key)
		}
	}

	// Cleanup
	for _, key := range testKeys {
		client.DeleteObject(ctx, key)
	}
}

// TestLocalStackDeleteObject tests deleting objects with LocalStack
func TestLocalStackDeleteObject(t *testing.T) {
	client := setupLocalStackTest(t)
	ctx := context.Background()

	testKey := fmt.Sprintf("test-delete-%d", time.Now().UnixNano())
	testData := []byte("Test data for deletion")

	// Create object
	err := client.PutObject(ctx, testKey, testData)
	if err != nil {
		t.Fatalf("PutObject failed: %v", err)
	}

	// Verify object exists
	data, err := client.GetObject(ctx, testKey)
	if err != nil {
		t.Fatalf("GetObject failed: %v", err)
	}
	if string(data) != string(testData) {
		t.Errorf("Data mismatch before deletion")
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

// TestLocalStackGetObjectRange tests getting object ranges with LocalStack
func TestLocalStackGetObjectRange(t *testing.T) {
	client := setupLocalStackTest(t)
	ctx := context.Background()

	testKey := fmt.Sprintf("test-range-%d", time.Now().UnixNano())
	testData := []byte("Hello World from LocalStack!")

	// Create object
	err := client.PutObject(ctx, testKey, testData)
	if err != nil {
		t.Fatalf("PutObject failed: %v", err)
	}

	// Get range (bytes 0-4: "Hello")
	data, err := client.GetObjectRange(ctx, testKey, 0, 4)
	if err != nil {
		t.Fatalf("GetObjectRange failed: %v", err)
	}

	expected := "Hello"
	if string(data) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(data))
	}

	// Get range from middle (bytes 6-10: "World")
	data, err = client.GetObjectRange(ctx, testKey, 6, 10)
	if err != nil {
		t.Fatalf("GetObjectRange failed: %v", err)
	}

	expected = "World"
	if string(data) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(data))
	}

	// Cleanup
	client.DeleteObject(ctx, testKey)
}

// TestLocalStackHeadObject tests getting object metadata with LocalStack
func TestLocalStackHeadObject(t *testing.T) {
	client := setupLocalStackTest(t)
	ctx := context.Background()

	testKey := fmt.Sprintf("test-head-%d", time.Now().UnixNano())
	testData := []byte("Test data for HeadObject")

	// Create object with metadata
	metadata := map[string]string{
		"test-key":  "test-value",
		"test-key2": "test-value2",
	}
	err := client.PutObjectWithMetadata(ctx, testKey, testData, metadata)
	if err != nil {
		t.Fatalf("PutObjectWithMetadata failed: %v", err)
	}

	// Get metadata
	headMetadata, err := client.HeadObject(ctx, testKey)
	if err != nil {
		t.Fatalf("HeadObject failed: %v", err)
	}

	// Verify metadata
	if headMetadata["test-key"] != "test-value" {
		t.Errorf("Expected metadata 'test-key'='test-value', got '%s'", headMetadata["test-key"])
	}

	if headMetadata["test-key2"] != "test-value2" {
		t.Errorf("Expected metadata 'test-key2'='test-value2', got '%s'", headMetadata["test-key2"])
	}

	// Cleanup
	client.DeleteObject(ctx, testKey)
}

// TestLocalStackHeadObjectSize tests getting object size with LocalStack
func TestLocalStackHeadObjectSize(t *testing.T) {
	client := setupLocalStackTest(t)
	ctx := context.Background()

	testKey := fmt.Sprintf("test-size-%d", time.Now().UnixNano())
	testData := []byte("Test data for size check")

	// Create object
	err := client.PutObject(ctx, testKey, testData)
	if err != nil {
		t.Fatalf("PutObject failed: %v", err)
	}

	// Get size
	size, err := client.HeadObjectSize(ctx, testKey)
	if err != nil {
		t.Fatalf("HeadObjectSize failed: %v", err)
	}

	expectedSize := int64(len(testData))
	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}

	// Cleanup
	client.DeleteObject(ctx, testKey)
}

// TestLocalStackIntegration runs a comprehensive integration test
func TestLocalStackIntegration(t *testing.T) {
	client := setupLocalStackTest(t)
	ctx := context.Background()

	// Test sequence: create, read, update, delete
	testKey := fmt.Sprintf("test-integration-%d", time.Now().UnixNano())

	// 1. Create file
	initialData := []byte("Initial content")
	err := client.PutObject(ctx, testKey, initialData)
	if err != nil {
		t.Fatalf("Step 1 - PutObject failed: %v", err)
	}

	// 2. Read file
	data, err := client.GetObject(ctx, testKey)
	if err != nil {
		t.Fatalf("Step 2 - GetObject failed: %v", err)
	}
	if string(data) != string(initialData) {
		t.Errorf("Step 2 - Data mismatch: expected '%s', got '%s'", string(initialData), string(data))
	}

	// 3. Update file (overwrite)
	updatedData := []byte("Updated content")
	err = client.PutObject(ctx, testKey, updatedData)
	if err != nil {
		t.Fatalf("Step 3 - PutObject (update) failed: %v", err)
	}

	// 4. Verify update
	data, err = client.GetObject(ctx, testKey)
	if err != nil {
		t.Fatalf("Step 4 - GetObject failed: %v", err)
	}
	if string(data) != string(updatedData) {
		t.Errorf("Step 4 - Data mismatch: expected '%s', got '%s'", string(updatedData), string(data))
	}

	// 5. Delete file
	err = client.DeleteObject(ctx, testKey)
	if err != nil {
		t.Fatalf("Step 5 - DeleteObject failed: %v", err)
	}

	// 6. Verify deletion
	_, err = client.GetObject(ctx, testKey)
	if err == nil {
		t.Error("Step 6 - Object should not exist after deletion")
	}
}
