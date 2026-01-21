package fuse

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/s3fs-fuse/s3fs-go/internal/credentials"
	"github.com/s3fs-fuse/s3fs-go/internal/s3client"
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

// setupLocalStackFilesystemTest sets up LocalStack test environment for filesystem
func setupLocalStackFilesystemTest(t *testing.T) *Filesystem {
	if !isLocalStackAvailable() {
		t.Skip("LocalStack is not available. Start it with: docker-compose -f docker-compose.localstack.yml up -d")
	}

	// Create credentials for LocalStack (dummy credentials)
	creds := credentials.NewCredentials()
	creds.AccessKeyID = "test"
	creds.SecretAccessKey = "test"

	// Create client with LocalStack endpoint
	client := s3client.NewClientWithEndpoint(localstackBucket, localstackRegion, localstackEndpoint, creds)

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

	return NewFilesystem(client)
}

// TestLocalStackRename tests renaming a file with LocalStack
func TestLocalStackRename(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	oldPath := fmt.Sprintf("test-rename-%d.txt", time.Now().UnixNano())
	newPath := fmt.Sprintf("test-renamed-%d.txt", time.Now().UnixNano())
	testData := []byte("Hello, World!")

	// Create file
	err := fs.WriteFile(ctx, oldPath, testData, 0)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Rename file
	err = fs.Rename(ctx, oldPath, newPath)
	if err != nil {
		t.Fatalf("Failed to rename file: %v", err)
	}

	// Verify old file doesn't exist
	_, err = fs.GetAttr(ctx, oldPath)
	if err == nil {
		t.Error("Old file should not exist after rename")
	}

	// Verify new file exists and has correct content
	data, err := fs.ReadFile(ctx, newPath, 0, 0)
	if err != nil {
		t.Fatalf("Failed to read renamed file: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("Expected content '%s', got '%s'", string(testData), string(data))
	}

	// Cleanup
	fs.Remove(ctx, newPath)
}

// TestLocalStackRenameLargeFile tests renaming a large file (triggers multipart copy)
func TestLocalStackRenameLargeFile(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	oldPath := fmt.Sprintf("test-rename-large-%d.txt", time.Now().UnixNano())
	newPath := fmt.Sprintf("test-renamed-large-%d.txt", time.Now().UnixNano())

	// Create a file larger than 5MB to trigger multipart copy
	largeData := make([]byte, 6*1024*1024) // 6MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	err := fs.WriteFile(ctx, oldPath, largeData, 0)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// Rename file
	err = fs.Rename(ctx, oldPath, newPath)
	if err != nil {
		t.Fatalf("Failed to rename large file: %v", err)
	}

	// Verify new file exists and has correct size
	attr, err := fs.GetAttr(ctx, newPath)
	if err != nil {
		t.Fatalf("Failed to get attributes of renamed file: %v", err)
	}

	if attr.Size != int64(len(largeData)) {
		t.Errorf("Expected size %d, got %d", len(largeData), attr.Size)
	}

	// Cleanup
	fs.Remove(ctx, newPath)
}

// TestLocalStackRenameNonExistent tests renaming a non-existent file
func TestLocalStackRenameNonExistent(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	oldPath := fmt.Sprintf("test-nonexistent-%d.txt", time.Now().UnixNano())
	newPath := fmt.Sprintf("test-renamed-%d.txt", time.Now().UnixNano())

	// Try to rename non-existent file
	err := fs.Rename(ctx, oldPath, newPath)
	if err == nil {
		t.Error("Expected error when renaming non-existent file")
	}
}

// TestLocalStackUtimens tests setting file times with LocalStack
func TestLocalStackUtimens(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	testFile := fmt.Sprintf("test-utimens-%d.txt", time.Now().UnixNano())
	testData := []byte("Test data")

	// Create file
	err := fs.WriteFile(ctx, testFile, testData, 0)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Set custom times
	atime := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)
	mtime := time.Date(2021, 2, 2, 13, 0, 0, 0, time.UTC)

	err = fs.Utimens(ctx, testFile, atime, mtime)
	if err != nil {
		t.Fatalf("Failed to set file times: %v", err)
	}

	// Verify times were set
	attr, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get file attributes: %v", err)
	}

	// Check mtime (atime might not be preserved in S3)
	// Allow small time difference due to S3 metadata precision (S3 stores times as seconds)
	timeDiff := attr.Mtime.Unix() - mtime.Unix()
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > 1 { // Allow 1 second difference
		t.Errorf("Expected mtime %v, got %v (diff: %d seconds)", mtime, attr.Mtime, timeDiff)
	}

	// Cleanup
	fs.Remove(ctx, testFile)
}

// TestLocalStackUtimensDirectory tests setting times on a directory
func TestLocalStackUtimensDirectory(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	testDir := fmt.Sprintf("test-utimens-dir-%d", time.Now().UnixNano())

	// Create directory
	err := fs.Mkdir(ctx, testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Set custom times
	atime := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)
	mtime := time.Date(2021, 2, 2, 13, 0, 0, 0, time.UTC)

	err = fs.Utimens(ctx, testDir, atime, mtime)
	if err != nil {
		t.Fatalf("Failed to set directory times: %v", err)
	}

	// Verify times were set
	attr, err := fs.GetAttr(ctx, testDir)
	if err != nil {
		t.Fatalf("Failed to get directory attributes: %v", err)
	}

	if !attr.Mode.IsDir() {
		t.Error("Path is not a directory")
	}

	// Cleanup
	fs.Rmdir(ctx, testDir)
}

// TestLocalStackWriteFileAppend tests appending to a file
func TestLocalStackWriteFileAppend(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	testFile := fmt.Sprintf("test-write-append-%d.txt", time.Now().UnixNano())
	initialData := []byte("Hello")
	appendData := []byte(" World")

	// Write initial data
	err := fs.WriteFile(ctx, testFile, initialData, 0)
	if err != nil {
		t.Fatalf("Failed to write initial data: %v", err)
	}

	// Append data
	err = fs.WriteFile(ctx, testFile, appendData, int64(len(initialData)))
	if err != nil {
		t.Fatalf("Failed to append data: %v", err)
	}

	// Read back and verify
	data, err := fs.ReadFile(ctx, testFile, 0, 0)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expected := append(initialData, appendData...)
	if string(data) != string(expected) {
		t.Errorf("Expected '%s', got '%s'", string(expected), string(data))
	}

	// Cleanup
	fs.Remove(ctx, testFile)
}

// TestLocalStackWriteFileOverwrite tests overwriting a file
func TestLocalStackWriteFileOverwrite(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	testFile := fmt.Sprintf("test-write-overwrite-%d.txt", time.Now().UnixNano())
	initialData := []byte("Initial content")
	newData := []byte("New content")

	// Write initial data
	err := fs.WriteFile(ctx, testFile, initialData, 0)
	if err != nil {
		t.Fatalf("Failed to write initial data: %v", err)
	}

	// Overwrite with new data
	err = fs.WriteFile(ctx, testFile, newData, 0)
	if err != nil {
		t.Fatalf("Failed to overwrite file: %v", err)
	}

	// Read back and verify
	data, err := fs.ReadFile(ctx, testFile, 0, 0)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != string(newData) {
		t.Errorf("Expected '%s', got '%s'", string(newData), string(data))
	}

	// Cleanup
	fs.Remove(ctx, testFile)
}

// TestLocalStackWriteFileTruncate tests truncating a file
func TestLocalStackWriteFileTruncate(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	testFile := fmt.Sprintf("test-write-truncate-%d.txt", time.Now().UnixNano())
	initialData := []byte("This is a long string that will be truncated")

	// Write initial data
	err := fs.WriteFile(ctx, testFile, initialData, 0)
	if err != nil {
		t.Fatalf("Failed to write initial data: %v", err)
	}

	// Truncate to empty
	err = fs.WriteFile(ctx, testFile, []byte{}, 0)
	if err != nil {
		t.Fatalf("Failed to truncate file: %v", err)
	}

	// Read back and verify
	data, err := fs.ReadFile(ctx, testFile, 0, 0)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("Expected empty file, got %d bytes", len(data))
	}

	// Cleanup
	fs.Remove(ctx, testFile)
}

// TestLocalStackWriteFileLarge tests writing a large file
func TestLocalStackWriteFileLarge(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	testFile := fmt.Sprintf("test-write-large-%d.txt", time.Now().UnixNano())

	// Create a large file (1MB)
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	err := fs.WriteFile(ctx, testFile, largeData, 0)
	if err != nil {
		t.Fatalf("Failed to write large file: %v", err)
	}

	// Verify size
	attr, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get file attributes: %v", err)
	}

	if attr.Size != int64(len(largeData)) {
		t.Errorf("Expected size %d, got %d", len(largeData), attr.Size)
	}

	// Cleanup
	fs.Remove(ctx, testFile)
}

// TestLocalStackSetXattr tests setting extended attributes on a file
func TestLocalStackSetXattr(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	testFile := fmt.Sprintf("test-xattr-%d.txt", time.Now().UnixNano())
	testData := []byte("Test data")

	// Create file
	err := fs.WriteFile(ctx, testFile, testData, 0)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Set extended attribute
	xattrName := "user.test"
	xattrValue := []byte("test value")

	err = fs.SetXattr(ctx, testFile, xattrName, xattrValue)
	if err != nil {
		t.Fatalf("Failed to set extended attribute: %v", err)
	}

	// Get extended attribute
	value, err := fs.GetXattr(ctx, testFile, xattrName)
	if err != nil {
		t.Fatalf("Failed to get extended attribute: %v", err)
	}

	if string(value) != string(xattrValue) {
		t.Errorf("Expected xattr value '%s', got '%s'", string(xattrValue), string(value))
	}

	// Cleanup
	fs.Remove(ctx, testFile)
}

// TestLocalStackGetXattr tests getting extended attributes
func TestLocalStackGetXattr(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	testFile := fmt.Sprintf("test-xattr-get-%d.txt", time.Now().UnixNano())
	testData := []byte("Test data")

	// Create file
	err := fs.WriteFile(ctx, testFile, testData, 0)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Set multiple extended attributes
	xattr1Name := "user.test1"
	xattr1Value := []byte("value1")
	xattr2Name := "user.test2"
	xattr2Value := []byte("value2")

	err = fs.SetXattr(ctx, testFile, xattr1Name, xattr1Value)
	if err != nil {
		t.Fatalf("Failed to set xattr1: %v", err)
	}

	err = fs.SetXattr(ctx, testFile, xattr2Name, xattr2Value)
	if err != nil {
		t.Fatalf("Failed to set xattr2: %v", err)
	}

	// Get first xattr
	value1, err := fs.GetXattr(ctx, testFile, xattr1Name)
	if err != nil {
		t.Fatalf("Failed to get xattr1: %v", err)
	}
	if string(value1) != string(xattr1Value) {
		t.Errorf("Expected xattr1 value '%s', got '%s'", string(xattr1Value), string(value1))
	}

	// Get second xattr
	value2, err := fs.GetXattr(ctx, testFile, xattr2Name)
	if err != nil {
		t.Fatalf("Failed to get xattr2: %v", err)
	}
	if string(value2) != string(xattr2Value) {
		t.Errorf("Expected xattr2 value '%s', got '%s'", string(xattr2Value), string(value2))
	}

	// Cleanup
	fs.Remove(ctx, testFile)
}

// TestLocalStackGetXattrNonExistent tests getting non-existent extended attribute
func TestLocalStackGetXattrNonExistent(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	testFile := fmt.Sprintf("test-xattr-nonexistent-%d.txt", time.Now().UnixNano())
	testData := []byte("Test data")

	// Create file
	err := fs.WriteFile(ctx, testFile, testData, 0)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Try to get non-existent xattr
	_, err = fs.GetXattr(ctx, testFile, "user.nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent xattr")
	}

	// Cleanup
	fs.Remove(ctx, testFile)
}

// TestLocalStackListXattr tests listing extended attributes
func TestLocalStackListXattr(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	testFile := fmt.Sprintf("test-xattr-list-%d.txt", time.Now().UnixNano())
	testData := []byte("Test data")

	// Create file
	err := fs.WriteFile(ctx, testFile, testData, 0)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Set multiple extended attributes
	xattrs := map[string][]byte{
		"user.test1": []byte("value1"),
		"user.test2": []byte("value2"),
		"user.test3": []byte("value3"),
	}

	for name, value := range xattrs {
		err = fs.SetXattr(ctx, testFile, name, value)
		if err != nil {
			t.Fatalf("Failed to set xattr %s: %v", name, err)
		}
	}

	// List extended attributes
	names, err := fs.ListXattr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to list xattrs: %v", err)
	}

	if len(names) != len(xattrs) {
		t.Errorf("Expected %d xattrs, got %d", len(xattrs), len(names))
	}

	// Verify all xattrs are listed
	for name := range xattrs {
		found := false
		for _, listedName := range names {
			if listedName == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Xattr %s not found in list", name)
		}
	}

	// Cleanup
	fs.Remove(ctx, testFile)
}

// TestLocalStackRemoveXattr tests removing extended attributes
func TestLocalStackRemoveXattr(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	testFile := fmt.Sprintf("test-xattr-remove-%d.txt", time.Now().UnixNano())
	testData := []byte("Test data")

	// Create file
	err := fs.WriteFile(ctx, testFile, testData, 0)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Set extended attribute
	xattrName := "user.test"
	xattrValue := []byte("test value")

	err = fs.SetXattr(ctx, testFile, xattrName, xattrValue)
	if err != nil {
		t.Fatalf("Failed to set xattr: %v", err)
	}

	// Verify it exists
	_, err = fs.GetXattr(ctx, testFile, xattrName)
	if err != nil {
		t.Fatalf("Xattr should exist: %v", err)
	}

	// Remove extended attribute
	err = fs.RemoveXattr(ctx, testFile, xattrName)
	if err != nil {
		t.Fatalf("Failed to remove xattr: %v", err)
	}

	// Verify it's gone
	_, err = fs.GetXattr(ctx, testFile, xattrName)
	if err == nil {
		t.Error("Xattr should not exist after removal")
	}

	// Cleanup
	fs.Remove(ctx, testFile)
}

// TestLocalStackXattrDirectory tests extended attributes on directories
func TestLocalStackXattrDirectory(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	testDir := fmt.Sprintf("test-xattr-dir-%d", time.Now().UnixNano())

	// Create directory
	err := fs.Mkdir(ctx, testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Set extended attribute on directory
	xattrName := "user.dirtest"
	xattrValue := []byte("dir value")

	err = fs.SetXattr(ctx, testDir, xattrName, xattrValue)
	if err != nil {
		t.Fatalf("Failed to set xattr on directory: %v", err)
	}

	// Get extended attribute from directory
	value, err := fs.GetXattr(ctx, testDir, xattrName)
	if err != nil {
		t.Fatalf("Failed to get xattr from directory: %v", err)
	}

	if string(value) != string(xattrValue) {
		t.Errorf("Expected xattr value '%s', got '%s'", string(xattrValue), string(value))
	}

	// List xattrs on directory
	names, err := fs.ListXattr(ctx, testDir)
	if err != nil {
		t.Fatalf("Failed to list xattrs on directory: %v", err)
	}

	found := false
	for _, name := range names {
		if name == xattrName {
			found = true
			break
		}
	}
	if !found {
		t.Error("Xattr not found in directory list")
	}

	// Remove xattr from directory
	err = fs.RemoveXattr(ctx, testDir, xattrName)
	if err != nil {
		t.Fatalf("Failed to remove xattr from directory: %v", err)
	}

	// Cleanup
	fs.Rmdir(ctx, testDir)
}

// TestLocalStackWriteFileMiddle tests writing to middle of file
func TestLocalStackWriteFileMiddle(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	testFile := fmt.Sprintf("test-write-middle-%d.txt", time.Now().UnixNano())
	initialData := []byte("Hello World")
	middleData := []byte("XXX")

	// Write initial data
	err := fs.WriteFile(ctx, testFile, initialData, 0)
	if err != nil {
		t.Fatalf("Failed to write initial data: %v", err)
	}

	// Write to middle of file
	err = fs.WriteFile(ctx, testFile, middleData, 6)
	if err != nil {
		t.Fatalf("Failed to write to middle: %v", err)
	}

	// Read back and verify
	data, err := fs.ReadFile(ctx, testFile, 0, 0)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expected := []byte("Hello XXXld")
	if string(data) != string(expected) {
		t.Errorf("Expected '%s', got '%s'", string(expected), string(data))
	}

	// Cleanup
	fs.Remove(ctx, testFile)
}

// TestLocalStackMkdirNested tests creating nested directories
func TestLocalStackMkdirNested(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	parentDir := fmt.Sprintf("test-nested-parent-%d", time.Now().UnixNano())
	childDir := parentDir + "/child"

	// Create parent directory
	err := fs.Mkdir(ctx, parentDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create parent directory: %v", err)
	}

	// Create child directory
	err = fs.Mkdir(ctx, childDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create child directory: %v", err)
	}

	// Verify both directories exist
	parentAttr, err := fs.GetAttr(ctx, parentDir)
	if err != nil {
		t.Fatalf("Failed to get parent attributes: %v", err)
	}
	if !parentAttr.Mode.IsDir() {
		t.Error("Parent is not a directory")
	}

	childAttr, err := fs.GetAttr(ctx, childDir)
	if err != nil {
		t.Fatalf("Failed to get child attributes: %v", err)
	}
	if !childAttr.Mode.IsDir() {
		t.Error("Child is not a directory")
	}

	// Cleanup
	fs.Rmdir(ctx, childDir)
	fs.Rmdir(ctx, parentDir)
}

// TestLocalStackRmdirWithMarker tests removing directory with .keep marker
func TestLocalStackRmdirWithMarker(t *testing.T) {
	fs := setupLocalStackFilesystemTest(t)
	ctx := context.Background()

	testDir := fmt.Sprintf("test-rmdir-marker-%d", time.Now().UnixNano())

	// Create directory (creates .keep marker)
	err := fs.Mkdir(ctx, testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Remove directory
	err = fs.Rmdir(ctx, testDir)
	if err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}

	// Verify directory doesn't exist
	_, err = fs.GetAttr(ctx, testDir)
	if err == nil {
		t.Error("Directory should not exist after removal")
	}
}
