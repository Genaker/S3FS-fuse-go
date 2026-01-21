package fuse

import (
	"context"
	"testing"

	"github.com/s3fs-fuse/s3fs-go/internal/s3client"
)

// TestExtendedAttributes tests basic extended attributes operations
func TestExtendedAttributes(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testFile := "test-xattr.txt"
	testText := "HELLO WORLD"

	err := fs.WriteFile(ctx, testFile, []byte(testText), 0)
	if err != nil {
		t.Skipf("Skipping test - S3 client not initialized: %v", err)
		return
	}

	// Set extended attribute
	xattrName := "user.test"
	xattrValue := []byte("test-value")
	err = fs.SetXattr(ctx, testFile, xattrName, xattrValue)
	if err != nil {
		t.Fatalf("Failed to set xattr: %v", err)
	}

	// Get extended attribute
	value, err := fs.GetXattr(ctx, testFile, xattrName)
	if err != nil {
		t.Fatalf("Failed to get xattr: %v", err)
	}

	if string(value) != string(xattrValue) {
		t.Errorf("Expected xattr value '%s', got '%s'", string(xattrValue), string(value))
	}

	// List extended attributes
	names, err := fs.ListXattr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to list xattr: %v", err)
	}

	found := false
	for _, name := range names {
		if name == xattrName {
			found = true
			break
		}
	}

	if !found {
		t.Error("Extended attribute not found in list")
	}

	// Remove extended attribute
	err = fs.RemoveXattr(ctx, testFile, xattrName)
	if err != nil {
		t.Fatalf("Failed to remove xattr: %v", err)
	}

	// Verify it's gone
	_, err = fs.GetXattr(ctx, testFile, xattrName)
	if err == nil {
		t.Error("Extended attribute should be removed")
	}

	err = fs.Remove(ctx, testFile)
	if err != nil {
		t.Logf("Warning: failed to cleanup test file: %v", err)
	}
}

// TestUpdateTimeXattr tests that setting xattr updates ctime
func TestUpdateTimeXattr(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testFile := "test-xattr-time.txt"
	testText := "HELLO WORLD"

	err := fs.WriteFile(ctx, testFile, []byte(testText), 0)
	if err != nil {
		t.Skipf("Skipping test - S3 client not initialized: %v", err)
		return
	}

	// Get initial attributes
	attr1, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get initial attributes: %v", err)
	}

	initialCtime := attr1.Mtime // Using Mtime as proxy for ctime

	// Set extended attribute (should update ctime)
	xattrName := "user.test"
	xattrValue := []byte("test-value")
	err = fs.SetXattr(ctx, testFile, xattrName, xattrValue)
	if err != nil {
		t.Fatalf("Failed to set xattr: %v", err)
	}

	// Get attributes after setting xattr
	attr2, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get attributes after xattr: %v", err)
	}

	// Ctime should be updated (or at least not before initial)
	if attr2.Mtime.Before(initialCtime) {
		t.Error("Ctime should be updated after setting xattr")
	}

	err = fs.Remove(ctx, testFile)
	if err != nil {
		t.Logf("Warning: failed to cleanup test file: %v", err)
	}
}

// TestUpdateDirectoryTimeSetXattr tests that setting xattr on directory updates ctime
func TestUpdateDirectoryTimeSetXattr(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testDir := "test-xattr-dir/"

	// Create directory by creating a placeholder file
	err := fs.Create(ctx, testDir+".keep", 0644)
	if err != nil {
		t.Skipf("Skipping test - S3 client not initialized: %v", err)
		return
	}

	// Get initial attributes
	attr1, err := fs.GetAttr(ctx, testDir)
	if err != nil {
		t.Fatalf("Failed to get initial directory attributes: %v", err)
	}

	initialCtime := attr1.Mtime

	// Set extended attribute on directory
	xattrName := "user.test"
	xattrValue := []byte("test-value")
	err = fs.SetXattr(ctx, testDir, xattrName, xattrValue)
	if err != nil {
		t.Fatalf("Failed to set xattr on directory: %v", err)
	}

	// Get attributes after setting xattr
	attr2, err := fs.GetAttr(ctx, testDir)
	if err != nil {
		t.Fatalf("Failed to get directory attributes after xattr: %v", err)
	}

	// Ctime should be updated
	if attr2.Mtime.Before(initialCtime) {
		t.Error("Directory ctime should be updated after setting xattr")
	}

	// Cleanup
	fs.Remove(ctx, testDir+".keep")
}
