//go:build integration

package fuse

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/s3fs-fuse/s3fs-go/internal/integration"
)

func TestSymlinkReadlink(t *testing.T) {
	integration.RequireLocalStack(t)
	
	fs := integration.SetupTestFilesystem(t, integration.LocalStackBucket, integration.LocalStackRegion)
	ctx := context.Background()

	// Use unique names to avoid conflicts
	timestamp := time.Now().UnixNano()
	targetPath := fmt.Sprintf("/target-%d.txt", timestamp)
	linkPath := fmt.Sprintf("/symlink-%d", timestamp)

	// Create a target file first
	targetData := []byte("target content")
	err := fs.WriteFile(ctx, targetPath, targetData, 0)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create a symlink
	err = fs.Symlink(ctx, targetPath, linkPath)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Verify symlink exists
	attr, err := fs.GetAttr(ctx, linkPath)
	if err != nil {
		t.Fatalf("Failed to get symlink attributes: %v", err)
	}
	
	if attr.Mode&os.ModeSymlink == 0 {
		t.Errorf("Expected symlink mode, got %v", attr.Mode)
	}

	// Read the symlink target
	readTarget, err := fs.Readlink(ctx, linkPath)
	if err != nil {
		t.Fatalf("Failed to read symlink: %v", err)
	}
	
	if readTarget != targetPath {
		t.Errorf("Expected target %q, got %q", targetPath, readTarget)
	}
}

func TestSymlinkAlreadyExists(t *testing.T) {
	integration.RequireLocalStack(t)
	
	fs := integration.SetupTestFilesystem(t, integration.LocalStackBucket, integration.LocalStackRegion)
	ctx := context.Background()

	// Use unique names
	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/existing-%d.txt", timestamp)

	// Create a file first
	err := fs.Create(ctx, filePath, 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Try to create symlink with same name
	err = fs.Symlink(ctx, "/target", filePath)
	if err != syscall.EEXIST {
		t.Errorf("Expected EEXIST, got %v", err)
	}
	
	// Cleanup
	fs.Remove(ctx, filePath)
}

func TestReadlinkNotFound(t *testing.T) {
	integration.RequireLocalStack(t)
	
	fs := integration.SetupTestFilesystem(t, integration.LocalStackBucket, integration.LocalStackRegion)
	ctx := context.Background()

	_, err := fs.Readlink(ctx, "/nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent symlink")
	}
}

func TestLink(t *testing.T) {
	integration.RequireLocalStack(t)
	
	fs := integration.SetupTestFilesystem(t, integration.LocalStackBucket, integration.LocalStackRegion)
	ctx := context.Background()

	// Link should return ENOTSUP
	err := fs.Link(ctx, "/source", "/dest")
	if err != syscall.ENOTSUP {
		t.Errorf("Expected ENOTSUP, got %v", err)
	}
}

func TestMknod(t *testing.T) {
	integration.RequireLocalStack(t)
	
	fs := integration.SetupTestFilesystem(t, integration.LocalStackBucket, integration.LocalStackRegion)
	ctx := context.Background()

	// Mknod should return ENOTSUP
	err := fs.Mknod(ctx, "/dev/null", 0644, 0)
	if err != syscall.ENOTSUP {
		t.Errorf("Expected ENOTSUP, got %v", err)
	}
}

func TestAccess(t *testing.T) {
	integration.RequireLocalStack(t)
	
	fs := integration.SetupTestFilesystem(t, integration.LocalStackBucket, integration.LocalStackRegion)
	ctx := context.Background()

	// Use unique name
	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/test-%d.txt", timestamp)

	// Create a file
	err := fs.Create(ctx, filePath, 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Test F_OK (file exists)
	err = fs.Access(ctx, filePath, 0)
	if err != nil {
		t.Errorf("Expected file to exist, got error: %v", err)
	}

	// Test R_OK (read permission)
	err = fs.Access(ctx, filePath, 4)
	if err != nil {
		t.Errorf("Expected read permission, got error: %v", err)
	}

	// Test W_OK (write permission)
	err = fs.Access(ctx, filePath, 2)
	if err != nil {
		t.Errorf("Expected write permission, got error: %v", err)
	}

	// Test X_OK (execute permission)
	err = fs.Access(ctx, filePath, 1)
	if err != nil {
		t.Errorf("Expected execute permission, got error: %v", err)
	}

	// Test nonexistent file
	err = fs.Access(ctx, "/nonexistent", 0)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
	
	// Cleanup
	fs.Remove(ctx, filePath)
}

func TestStatfs(t *testing.T) {
	integration.RequireLocalStack(t)
	
	fs := integration.SetupTestFilesystem(t, integration.LocalStackBucket, integration.LocalStackRegion)
	ctx := context.Background()

	statfs, err := fs.Statfs(ctx)
	if err != nil {
		t.Fatalf("Failed to get filesystem stats: %v", err)
	}

	if statfs.Bsize == 0 {
		t.Error("Expected nonzero block size")
	}
	if statfs.Blocks == 0 {
		t.Error("Expected nonzero total blocks")
	}
	if statfs.Bfree == 0 {
		t.Error("Expected nonzero free blocks")
	}
	if statfs.Namelen == 0 {
		t.Error("Expected nonzero max filename length")
	}
}

func TestFlush(t *testing.T) {
	integration.RequireLocalStack(t)
	
	fs := integration.SetupTestFilesystem(t, integration.LocalStackBucket, integration.LocalStackRegion)
	ctx := context.Background()

	// Use unique name
	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/test-%d.txt", timestamp)

	// Create and write to a file
	err := fs.Create(ctx, filePath, 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	data := []byte("test data")
	err = fs.WriteFile(ctx, filePath, data, 0)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Flush should succeed even if file is not cached
	err = fs.Flush(ctx, filePath)
	if err != nil {
		t.Errorf("Flush failed: %v", err)
	}
	
	// Cleanup
	fs.Remove(ctx, filePath)
}

func TestFsync(t *testing.T) {
	integration.RequireLocalStack(t)
	
	fs := integration.SetupTestFilesystem(t, integration.LocalStackBucket, integration.LocalStackRegion)
	ctx := context.Background()

	// Use unique name
	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/test-%d.txt", timestamp)

	// Create and write to a file
	err := fs.Create(ctx, filePath, 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	data := []byte("test data")
	err = fs.WriteFile(ctx, filePath, data, 0)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Test fsync (sync data and metadata)
	err = fs.Fsync(ctx, filePath, false)
	if err != nil {
		t.Errorf("Fsync failed: %v", err)
	}

	// Test fdatasync (sync data only)
	err = fs.Fsync(ctx, filePath, true)
	if err != nil {
		t.Errorf("Fdatasync failed: %v", err)
	}
	
	// Cleanup
	fs.Remove(ctx, filePath)
}

func TestRelease(t *testing.T) {
	integration.RequireLocalStack(t)
	
	fs := integration.SetupTestFilesystem(t, integration.LocalStackBucket, integration.LocalStackRegion)
	ctx := context.Background()

	// Use unique name
	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/test-%d.txt", timestamp)

	// Create a file
	err := fs.Create(ctx, filePath, 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Release should succeed even if file is not cached
	err = fs.Release(ctx, filePath)
	if err != nil {
		t.Errorf("Release failed: %v", err)
	}
	
	// Cleanup
	fs.Remove(ctx, filePath)
}

func TestOpendir(t *testing.T) {
	integration.RequireLocalStack(t)
	
	fs := integration.SetupTestFilesystem(t, integration.LocalStackBucket, integration.LocalStackRegion)
	ctx := context.Background()

	// Use unique directory name to avoid conflicts
	timestamp := time.Now().UnixNano()
	dirPath := fmt.Sprintf("/testdir-%d", timestamp)
	err := fs.Mkdir(ctx, dirPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Opendir should succeed
	err = fs.Opendir(ctx, dirPath)
	if err != nil {
		t.Errorf("Opendir failed: %v", err)
	}

	// Opendir on file should fail
	filePath := fmt.Sprintf("/test-%d.txt", timestamp)
	err = fs.Create(ctx, filePath, 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	err = fs.Opendir(ctx, filePath)
	if err != syscall.ENOTDIR {
		t.Errorf("Expected ENOTDIR for file, got %v", err)
	}

	// Opendir on nonexistent path should fail
	err = fs.Opendir(ctx, "/nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent directory")
	}
}
