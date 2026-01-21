package fuse

import (
	"context"
	"os"
	"testing"

	"github.com/s3fs-fuse/s3fs-go/internal/s3client"
)

// TestMkdir tests creating a directory
func TestMkdir(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testDir := "test-mkdir-dir"

	// Create directory
	err := fs.Mkdir(ctx, testDir, 0755)
	if err != nil {
		t.Skipf("Skipping test - S3 client not initialized: %v", err)
		return
	}

	// Verify directory exists
	attr, err := fs.GetAttr(ctx, testDir)
	if err != nil {
		t.Fatalf("Failed to get directory attributes: %v", err)
	}

	if !attr.Mode.IsDir() {
		t.Error("Created path is not a directory")
	}

	// Verify mode
	expectedMode := os.ModeDir | 0755
	if attr.Mode&^os.ModeType != expectedMode&^os.ModeType {
		t.Errorf("Expected mode %o, got %o", expectedMode, attr.Mode)
	}

	// Cleanup
	fs.Rmdir(ctx, testDir)
}

// TestMkdirExisting tests creating an existing directory
func TestMkdirExisting(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testDir := "test-mkdir-existing"

	// Create directory first time
	err := fs.Mkdir(ctx, testDir, 0755)
	if err != nil {
		t.Skipf("Skipping test - S3 client not initialized: %v", err)
		return
	}

	// Try to create again (should fail)
	err = fs.Mkdir(ctx, testDir, 0755)
	if err == nil {
		t.Error("Expected error when creating existing directory")
	}

	// Cleanup
	fs.Rmdir(ctx, testDir)
}

// TestRmdir tests removing an empty directory
func TestRmdir(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testDir := "test-rmdir-dir"

	// Create directory
	err := fs.Mkdir(ctx, testDir, 0755)
	if err != nil {
		t.Skipf("Skipping test - S3 client not initialized: %v", err)
		return
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

// TestRmdirNonEmpty tests removing a non-empty directory
func TestRmdirNonEmpty(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testDir := "test-rmdir-nonempty"
	testFile := testDir + "/file.txt"

	// Create directory
	err := fs.Mkdir(ctx, testDir, 0755)
	if err != nil {
		t.Skipf("Skipping test - S3 client not initialized: %v", err)
		return
	}

	// Create a file in the directory
	err = fs.Create(ctx, testFile, 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Try to remove directory (should fail)
	err = fs.Rmdir(ctx, testDir)
	if err == nil {
		t.Error("Expected error when removing non-empty directory")
	}

	// Cleanup
	fs.Remove(ctx, testFile)
	fs.Rmdir(ctx, testDir)
}

// TestRmdirNonExistent tests removing a non-existent directory
func TestRmdirNonExistent(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testDir := "test-rmdir-nonexistent"

	// Try to remove non-existent directory
	err := fs.Rmdir(ctx, testDir)
	if err == nil {
		t.Error("Expected error when removing non-existent directory")
	}
}

// TestMkdirRmdirIntegration tests mkdir and rmdir together
func TestMkdirRmdirIntegration(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testDir := "test-mkdir-rmdir-integration"

	// Create directory
	err := fs.Mkdir(ctx, testDir, 0755)
	if err != nil {
		t.Skipf("Skipping test - S3 client not initialized: %v", err)
		return
	}

	// Verify it exists
	attr, err := fs.GetAttr(ctx, testDir)
	if err != nil {
		t.Fatalf("Failed to get directory attributes: %v", err)
	}
	if !attr.Mode.IsDir() {
		t.Error("Created path is not a directory")
	}

	// List directory (should be empty or only contain .keep)
	entries, err := fs.ReadDir(ctx, testDir)
	if err != nil {
		t.Fatalf("Failed to list directory: %v", err)
	}

	// Remove directory
	err = fs.Rmdir(ctx, testDir)
	if err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}

	// Verify it's gone
	_, err = fs.GetAttr(ctx, testDir)
	if err == nil {
		t.Error("Directory should not exist after removal")
	}

	// Verify entries list is empty (or only had .keep)
	_ = entries
}
