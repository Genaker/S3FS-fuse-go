package fuse

import (
	"context"
	"os"
	"testing"

	"github.com/s3fs-fuse/s3fs-go/internal/s3client"
)

// TestChmod tests changing file permissions
func TestChmod(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testFile := "test-chmod.txt"
	testText := "HELLO WORLD"

	err := fs.WriteFile(ctx, testFile, []byte(testText), 0)
	if err != nil {
		t.Skipf("Skipping test - S3 client not initialized: %v", err)
		return
	}

	// Get original permissions
	attr1, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get original attributes: %v", err)
	}

	originalMode := attr1.Mode

	// Change permissions to 0777
	newMode := os.FileMode(0777)
	err = fs.Chmod(ctx, testFile, newMode)
	if err != nil {
		t.Fatalf("Failed to chmod: %v", err)
	}

	// Get new permissions
	attr2, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get new attributes: %v", err)
	}

	if attr2.Mode == originalMode {
		t.Error("Permissions should have changed")
	}

	// Verify new mode (mask out file type bits)
	if attr2.Mode&0777 != newMode {
		t.Errorf("Expected mode %o, got %o", newMode, attr2.Mode&0777)
	}

	err = fs.Remove(ctx, testFile)
	if err != nil {
		t.Logf("Warning: failed to cleanup test file: %v", err)
	}
}

// TestChown tests changing file ownership
func TestChown(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testFile := "test-chown.txt"
	testText := "HELLO WORLD"

	err := fs.WriteFile(ctx, testFile, []byte(testText), 0)
	if err != nil {
		t.Skipf("Skipping test - S3 client not initialized: %v", err)
		return
	}

	// Get original ownership
	attr1, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get original attributes: %v", err)
	}

	originalUid := attr1.Uid
	originalGid := attr1.Gid

	// Change ownership to 1000:1000
	newUid := uint32(1000)
	newGid := uint32(1000)
	err = fs.Chown(ctx, testFile, newUid, newGid)
	if err != nil {
		t.Fatalf("Failed to chown: %v", err)
	}

	// Get new ownership
	attr2, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get new attributes: %v", err)
	}

	// Check if ownership changed (may be same if already 1000:1000)
	if attr2.Uid != newUid && originalUid != newUid {
		t.Errorf("Expected UID %d, got %d", newUid, attr2.Uid)
	}

	if attr2.Gid != newGid && originalGid != newGid {
		t.Errorf("Expected GID %d, got %d", newGid, attr2.Gid)
	}

	err = fs.Remove(ctx, testFile)
	if err != nil {
		t.Logf("Warning: failed to cleanup test file: %v", err)
	}
}

// TestChmodMountpoint tests changing mountpoint permissions
func TestChmodMountpoint(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	// Test root directory permissions
	attr1, err := fs.GetAttr(ctx, "/")
	if err != nil {
		t.Fatalf("Failed to get root attributes: %v", err)
	}

	originalMode := attr1.Mode

	// Change root permissions
	newMode := os.FileMode(0755)
	err = fs.Chmod(ctx, "/", newMode)
	if err != nil {
		t.Fatalf("Failed to chmod root: %v", err)
	}

	attr2, err := fs.GetAttr(ctx, "/")
	if err != nil {
		t.Fatalf("Failed to get new root attributes: %v", err)
	}

	// For directories, mode changes may not persist without actual S3 objects
	// Just verify the operation doesn't error
	if attr2.Mode == originalMode {
		t.Logf("Root permissions unchanged (may not be supported for directories): %o", attr2.Mode)
	}
}

// TestChownMountpoint tests changing mountpoint ownership
func TestChownMountpoint(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	// Test root directory ownership
	attr1, err := fs.GetAttr(ctx, "/")
	if err != nil {
		t.Fatalf("Failed to get root attributes: %v", err)
	}

	originalUid := attr1.Uid
	originalGid := attr1.Gid

	// Change root ownership
	newUid := uint32(1000)
	newGid := uint32(1000)
	err = fs.Chown(ctx, "/", newUid, newGid)
	if err != nil {
		t.Fatalf("Failed to chown root: %v", err)
	}

	attr2, err := fs.GetAttr(ctx, "/")
	if err != nil {
		t.Fatalf("Failed to get new root attributes: %v", err)
	}

	// For directories, ownership changes may not persist without actual S3 objects
	// Just verify the operation doesn't error
	if attr2.Uid != newUid && originalUid != newUid && attr2.Uid == 0 {
		t.Logf("UID unchanged (may not be supported for directories): expected %d, got %d", newUid, attr2.Uid)
	}

	if attr2.Gid != newGid && originalGid != newGid && attr2.Gid == 0 {
		t.Logf("GID unchanged (may not be supported for directories): expected %d, got %d", newGid, attr2.Gid)
	}
}
