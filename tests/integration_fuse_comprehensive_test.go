//go:build integration

package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestChmod tests changing file permissions
func TestChmod(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/test-chmod-%d.txt", timestamp)
	testData := []byte("test data")

	// Create file
	err := fs.WriteFile(ctx, filePath, testData, 0)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Change permissions to 0755
	err = fs.Chmod(ctx, filePath, 0755)
	if err != nil {
		t.Fatalf("Failed to chmod file: %v", err)
	}

	// Verify permissions were changed
	attr, err := fs.GetAttr(ctx, filePath)
	if err != nil {
		t.Fatalf("Failed to get file attributes: %v", err)
	}

	expectedMode := os.FileMode(0755)
	if attr.Mode&0777 != expectedMode {
		t.Errorf("Expected mode %o, got %o", expectedMode, attr.Mode&0777)
	}

	// Change permissions to 0600
	err = fs.Chmod(ctx, filePath, 0600)
	if err != nil {
		t.Fatalf("Failed to chmod file to 0600: %v", err)
	}

	attr, err = fs.GetAttr(ctx, filePath)
	if err != nil {
		t.Fatalf("Failed to get file attributes: %v", err)
	}

	expectedMode = os.FileMode(0600)
	if attr.Mode&0777 != expectedMode {
		t.Errorf("Expected mode %o, got %o", expectedMode, attr.Mode&0777)
	}

	// Cleanup
	fs.Remove(ctx, filePath)
}

// TestChmodDirectory tests changing directory permissions
func TestChmodDirectory(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	dirPath := fmt.Sprintf("/test-chmod-dir-%d", timestamp)

	// Create directory
	err := fs.Mkdir(ctx, dirPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Change permissions to 0700
	err = fs.Chmod(ctx, dirPath, 0700)
	if err != nil {
		t.Fatalf("Failed to chmod directory: %v", err)
	}

	// Verify permissions were changed
	attr, err := fs.GetAttr(ctx, dirPath)
	if err != nil {
		t.Fatalf("Failed to get directory attributes: %v", err)
	}

	expectedMode := os.FileMode(0700)
	if attr.Mode&0777 != expectedMode {
		t.Errorf("Expected mode %o, got %o", expectedMode, attr.Mode&0777)
	}

	// Cleanup
	fs.Rmdir(ctx, dirPath)
}

// TestChown tests changing file ownership
func TestChown(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/test-chown-%d.txt", timestamp)
	testData := []byte("test data")

	// Create file
	err := fs.WriteFile(ctx, filePath, testData, 0)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Change ownership
	newUid := uint32(1000)
	newGid := uint32(1000)
	err = fs.Chown(ctx, filePath, newUid, newGid)
	if err != nil {
		t.Fatalf("Failed to chown file: %v", err)
	}

	// Verify ownership was changed
	attr, err := fs.GetAttr(ctx, filePath)
	if err != nil {
		t.Fatalf("Failed to get file attributes: %v", err)
	}

	if attr.Uid != newUid {
		t.Errorf("Expected UID %d, got %d", newUid, attr.Uid)
	}
	if attr.Gid != newGid {
		t.Errorf("Expected GID %d, got %d", newGid, attr.Gid)
	}

	// Change only UID
	err = fs.Chown(ctx, filePath, 2000, attr.Gid)
	if err != nil {
		t.Fatalf("Failed to chown file (UID only): %v", err)
	}

	attr, err = fs.GetAttr(ctx, filePath)
	if err != nil {
		t.Fatalf("Failed to get file attributes: %v", err)
	}

	if attr.Uid != 2000 {
		t.Errorf("Expected UID %d, got %d", 2000, attr.Uid)
	}

	// Cleanup
	fs.Remove(ctx, filePath)
}

// TestTruncate tests truncating files
func TestTruncate(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/test-truncate-%d.txt", timestamp)
	initialData := []byte("This is a longer test string for truncation")

	// Create file with initial data
	err := fs.WriteFile(ctx, filePath, initialData, 0)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Verify initial size
	attr, err := fs.GetAttr(ctx, filePath)
	if err != nil {
		t.Fatalf("Failed to get file attributes: %v", err)
	}
	if attr.Size != int64(len(initialData)) {
		t.Errorf("Expected initial size %d, got %d", len(initialData), attr.Size)
	}

	// Truncate to 10 bytes using WriteFile with offset 0
	truncatedData := initialData[:10]
	err = fs.WriteFile(ctx, filePath, truncatedData, 0)
	if err != nil {
		t.Fatalf("Failed to truncate file: %v", err)
	}

	// Flush buffered data to ensure it's uploaded
	err = fs.Flush(ctx, filePath)
	if err != nil {
		t.Fatalf("Failed to flush file: %v", err)
	}

	// Verify truncated size
	attr, err = fs.GetAttr(ctx, filePath)
	if err != nil {
		t.Fatalf("Failed to get file attributes: %v", err)
	}
	if attr.Size != 10 {
		t.Errorf("Expected truncated size 10, got %d", attr.Size)
	}

	// Read truncated file
	data, err := fs.ReadFile(ctx, filePath, 0, 0)
	if err != nil {
		t.Fatalf("Failed to read truncated file: %v", err)
	}
	if len(data) != 10 {
		t.Errorf("Expected read size 10, got %d", len(data))
	}

	// Truncate to 0 (empty file) by writing empty data
	err = fs.WriteFile(ctx, filePath, []byte{}, 0)
	if err != nil {
		t.Fatalf("Failed to truncate file to 0: %v", err)
	}

	// Flush buffered data
	err = fs.Flush(ctx, filePath)
	if err != nil {
		t.Fatalf("Failed to flush file: %v", err)
	}

	attr, err = fs.GetAttr(ctx, filePath)
	if err != nil {
		t.Fatalf("Failed to get file attributes: %v", err)
	}
	if attr.Size != 0 {
		t.Errorf("Expected size 0, got %d", attr.Size)
	}

	// Cleanup
	fs.Remove(ctx, filePath)
}

// TestCreate tests creating new files
func TestCreate(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/test-create-%d.txt", timestamp)

	// Create new file
	err := fs.Create(ctx, filePath, 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Verify file exists
	attr, err := fs.GetAttr(ctx, filePath)
	if err != nil {
		t.Fatalf("Failed to get file attributes: %v", err)
	}

	if attr.Size != 0 {
		t.Errorf("Expected new file size 0, got %d", attr.Size)
	}

	expectedMode := os.FileMode(0644)
	if attr.Mode&0777 != expectedMode {
		t.Errorf("Expected mode %o, got %o", expectedMode, attr.Mode&0777)
	}

	// Try to create again (should fail)
	err = fs.Create(ctx, filePath, 0644)
	if err == nil {
		t.Error("Expected error when creating existing file")
	}

	// Cleanup
	fs.Remove(ctx, filePath)
}

// TestReadDirEmpty tests reading empty directory
func TestReadDirEmpty(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	dirPath := fmt.Sprintf("/test-empty-dir-%d", timestamp)

	// Create empty directory
	err := fs.Mkdir(ctx, dirPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Read directory
	entries, err := fs.ReadDir(ctx, dirPath)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	// Should be empty (or only contain .keep)
	realEntries := 0
	for _, entry := range entries {
		if entry.Name != ".keep" {
			realEntries++
		}
	}

	if realEntries != 0 {
		t.Errorf("Expected empty directory, found %d entries", realEntries)
	}

	// Cleanup
	fs.Rmdir(ctx, dirPath)
}

// TestReadDirWithFiles tests reading directory with multiple files
func TestReadDirWithFiles(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	dirPath := fmt.Sprintf("/test-dir-%d", timestamp)

	// Create directory
	err := fs.Mkdir(ctx, dirPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Create multiple files
	fileNames := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, fileName := range fileNames {
		filePath := fmt.Sprintf("%s/%s", dirPath, fileName)
		err = fs.WriteFile(ctx, filePath, []byte("test"), 0)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", fileName, err)
		}
	}

	// Read directory
	entries, err := fs.ReadDir(ctx, dirPath)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	// Verify all files are present
	foundFiles := make(map[string]bool)
	for _, entry := range entries {
		if entry.Name != ".keep" {
			foundFiles[entry.Name] = true
		}
	}

	for _, fileName := range fileNames {
		if !foundFiles[fileName] {
			t.Errorf("File %s not found in directory listing", fileName)
		}
	}

	// Cleanup
	for _, fileName := range fileNames {
		filePath := fmt.Sprintf("%s/%s", dirPath, fileName)
		fs.Remove(ctx, filePath)
	}
	fs.Rmdir(ctx, dirPath)
}

// TestGetAttrNonExistent tests getting attributes of non-existent file
func TestGetAttrNonExistent(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	_, err := fs.GetAttr(ctx, "/nonexistent-file-12345")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// TestReadFileRange tests reading file with range
func TestReadFileRange(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/test-range-%d.txt", timestamp)
	testData := []byte("0123456789ABCDEF")

	// Create file
	err := fs.WriteFile(ctx, filePath, testData, 0)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Read range from offset 5, size 5
	data, err := fs.ReadFile(ctx, filePath, 5, 5)
	if err != nil {
		t.Fatalf("Failed to read file range: %v", err)
	}

	expected := testData[5:10]
	if string(data) != string(expected) {
		t.Errorf("Expected %q, got %q", string(expected), string(data))
	}

	// Read range from offset 10 to end
	data, err = fs.ReadFile(ctx, filePath, 10, 0)
	if err != nil {
		t.Fatalf("Failed to read file range to end: %v", err)
	}

	expected = testData[10:]
	if string(data) != string(expected) {
		t.Errorf("Expected %q, got %q", string(expected), string(data))
	}

	// Cleanup
	fs.Remove(ctx, filePath)
}

// TestWriteFileOffset tests writing file at specific offset
func TestWriteFileOffset(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/test-write-offset-%d.txt", timestamp)
	initialData := []byte("Hello World")

	// Create file with initial data
	err := fs.WriteFile(ctx, filePath, initialData, 0)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Write at offset 6 (should replace "Wo" with "Go", leaving "rld")
	newData := []byte("Go")
	err = fs.WriteFile(ctx, filePath, newData, 6)
	if err != nil {
		t.Fatalf("Failed to write at offset: %v", err)
	}

	// Read entire file
	data, err := fs.ReadFile(ctx, filePath, 0, 0)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// WriteFile at offset replaces bytes starting at offset, not truncating
	// So "Hello World" at offset 6 with "Go" becomes "Hello Gorld"
	expected := []byte("Hello Gorld")
	if string(data) != string(expected) {
		t.Errorf("Expected %q, got %q", string(expected), string(data))
	}
	
	// Test writing at offset 0 (full file replacement) - use a new file to avoid cache issues
	newFilePath := fmt.Sprintf("/test-write-offset-new-%d.txt", timestamp)
	truncData := []byte("X")
	err = fs.WriteFile(ctx, newFilePath, truncData, 0)
	if err != nil {
		t.Fatalf("Failed to write at offset 0: %v", err)
	}
	
	// Wait a bit for S3 to propagate
	time.Sleep(200 * time.Millisecond)
	
	data, err = fs.ReadFile(ctx, newFilePath, 0, 0)
	if err != nil {
		t.Fatalf("Failed to read file after write at offset 0: %v", err)
	}
	
	// Should be "X" (replaced entire file)
	expected = []byte("X")
	if string(data) != string(expected) {
		t.Errorf("After write at offset 0: Expected %q, got %q", string(expected), string(data))
	}
	
	// Cleanup
	fs.Remove(ctx, newFilePath)

	// Cleanup
	fs.Remove(ctx, filePath)
}

// TestSymlinkMultiple tests creating multiple symlinks
func TestSymlinkMultiple(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	targetPath := fmt.Sprintf("/target-%d.txt", timestamp)
	link1Path := fmt.Sprintf("/link1-%d", timestamp)
	link2Path := fmt.Sprintf("/link2-%d", timestamp)

	// Create target file
	err := fs.WriteFile(ctx, targetPath, []byte("target content"), 0)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create first symlink
	err = fs.Symlink(ctx, targetPath, link1Path)
	if err != nil {
		t.Fatalf("Failed to create first symlink: %v", err)
	}

	// Create second symlink
	err = fs.Symlink(ctx, targetPath, link2Path)
	if err != nil {
		t.Fatalf("Failed to create second symlink: %v", err)
	}

	// Read both symlinks
	target1, err := fs.Readlink(ctx, link1Path)
	if err != nil {
		t.Fatalf("Failed to read first symlink: %v", err)
	}

	target2, err := fs.Readlink(ctx, link2Path)
	if err != nil {
		t.Fatalf("Failed to read second symlink: %v", err)
	}

	if target1 != targetPath {
		t.Errorf("First symlink: expected %q, got %q", targetPath, target1)
	}
	if target2 != targetPath {
		t.Errorf("Second symlink: expected %q, got %q", targetPath, target2)
	}

	// Cleanup
	fs.Remove(ctx, link1Path)
	fs.Remove(ctx, link2Path)
	fs.Remove(ctx, targetPath)
}

// TestSymlinkRelativePath tests symlink with relative path
func TestSymlinkRelativePath(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	targetPath := fmt.Sprintf("/target-%d.txt", timestamp)
	linkPath := fmt.Sprintf("/link-%d", timestamp)

	// Create target file
	err := fs.WriteFile(ctx, targetPath, []byte("target"), 0)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create symlink with relative path
	relativeTarget := fmt.Sprintf("target-%d.txt", timestamp)
	err = fs.Symlink(ctx, relativeTarget, linkPath)
	if err != nil {
		t.Fatalf("Failed to create symlink with relative path: %v", err)
	}

	// Read symlink
	target, err := fs.Readlink(ctx, linkPath)
	if err != nil {
		t.Fatalf("Failed to read symlink: %v", err)
	}

	if target != relativeTarget {
		t.Errorf("Expected relative target %q, got %q", relativeTarget, target)
	}

	// Cleanup
	fs.Remove(ctx, linkPath)
	fs.Remove(ctx, targetPath)
}

// TestAccessFile tests access checks on files
func TestAccessFile(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/test-access-%d.txt", timestamp)

	// Create file
	err := fs.WriteFile(ctx, filePath, []byte("test"), 0)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Test F_OK (file exists)
	err = fs.Access(ctx, filePath, 0)
	if err != nil {
		t.Errorf("F_OK check failed: %v", err)
	}

	// Test R_OK (read permission)
	err = fs.Access(ctx, filePath, 4)
	if err != nil {
		t.Errorf("R_OK check failed: %v", err)
	}

	// Test W_OK (write permission)
	err = fs.Access(ctx, filePath, 2)
	if err != nil {
		t.Errorf("W_OK check failed: %v", err)
	}

	// Test X_OK (execute permission)
	err = fs.Access(ctx, filePath, 1)
	if err != nil {
		t.Errorf("X_OK check failed: %v", err)
	}

	// Test combined permissions
	err = fs.Access(ctx, filePath, 6) // R_OK | W_OK
	if err != nil {
		t.Errorf("R_OK|W_OK check failed: %v", err)
	}

	// Cleanup
	fs.Remove(ctx, filePath)
}

// TestAccessDirectory tests access checks on directories
func TestAccessDirectory(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	dirPath := fmt.Sprintf("/test-access-dir-%d", timestamp)

	// Create directory
	err := fs.Mkdir(ctx, dirPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Test F_OK (directory exists)
	err = fs.Access(ctx, dirPath, 0)
	if err != nil {
		t.Errorf("F_OK check failed: %v", err)
	}

	// Test X_OK (execute/search permission for directory)
	err = fs.Access(ctx, dirPath, 1)
	if err != nil {
		t.Errorf("X_OK check failed: %v", err)
	}

	// Cleanup
	fs.Rmdir(ctx, dirPath)
}

// TestStatfsValues tests filesystem statistics values
func TestStatfsValues(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	statfs, err := fs.Statfs(ctx)
	if err != nil {
		t.Fatalf("Failed to get filesystem stats: %v", err)
	}

	// Verify all fields are set
	if statfs.Bsize == 0 {
		t.Error("Block size should not be zero")
	}
	if statfs.Blocks == 0 {
		t.Error("Total blocks should not be zero")
	}
	if statfs.Bfree == 0 {
		t.Error("Free blocks should not be zero")
	}
	if statfs.Bavail == 0 {
		t.Error("Available blocks should not be zero")
	}
	if statfs.Files == 0 {
		t.Error("Total files should not be zero")
	}
	if statfs.Ffree == 0 {
		t.Error("Free files should not be zero")
	}
	if statfs.Namelen == 0 {
		t.Error("Max filename length should not be zero")
	}

	// Verify reasonable values
	if statfs.Bsize < 512 {
		t.Errorf("Block size %d seems too small", statfs.Bsize)
	}
	if statfs.Namelen < 255 {
		t.Errorf("Max filename length %d seems too small", statfs.Namelen)
	}
}

// TestFlushWithData tests flushing file buffers with data
func TestFlushWithData(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/test-flush-%d.txt", timestamp)
	testData := []byte("test data for flushing")

	// Create and write file
	err := fs.WriteFile(ctx, filePath, testData, 0)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Flush should succeed
	err = fs.Flush(ctx, filePath)
	if err != nil {
		t.Errorf("Flush failed: %v", err)
	}

	// Verify data is still readable after flush
	data, err := fs.ReadFile(ctx, filePath, 0, 0)
	if err != nil {
		t.Fatalf("Failed to read file after flush: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("Expected %q, got %q", string(testData), string(data))
	}

	// Cleanup
	fs.Remove(ctx, filePath)
}

// TestFsyncWithData tests syncing file data
func TestFsyncWithData(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/test-fsync-%d.txt", timestamp)
	testData := []byte("test data for fsync")

	// Create and write file
	err := fs.WriteFile(ctx, filePath, testData, 0)
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

	// Verify data is still readable after fsync
	data, err := fs.ReadFile(ctx, filePath, 0, 0)
	if err != nil {
		t.Fatalf("Failed to read file after fsync: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("Expected %q, got %q", string(testData), string(data))
	}

	// Cleanup
	fs.Remove(ctx, filePath)
}

// TestReleaseAfterWrite tests releasing file handle after write
func TestReleaseAfterWrite(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/test-release-%d.txt", timestamp)
	testData := []byte("test data")

	// Create file
	err := fs.Create(ctx, filePath, 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Write data
	err = fs.WriteFile(ctx, filePath, testData, 0)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Release should succeed
	err = fs.Release(ctx, filePath)
	if err != nil {
		t.Errorf("Release failed: %v", err)
	}

	// Verify data is still readable after release
	data, err := fs.ReadFile(ctx, filePath, 0, 0)
	if err != nil {
		t.Fatalf("Failed to read file after release: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("Expected %q, got %q", string(testData), string(data))
	}

	// Cleanup
	fs.Remove(ctx, filePath)
}

// TestRenameDirectory tests renaming directories
func TestRenameDirectory(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	oldDirPath := fmt.Sprintf("/old-dir-%d", timestamp)
	newDirPath := fmt.Sprintf("/new-dir-%d", timestamp)

	// Create directory
	err := fs.Mkdir(ctx, oldDirPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Create file in directory
	filePath := fmt.Sprintf("%s/file.txt", oldDirPath)
	err = fs.WriteFile(ctx, filePath, []byte("test"), 0)
	if err != nil {
		t.Fatalf("Failed to create file in directory: %v", err)
	}

	// Rename directory
	err = fs.Rename(ctx, oldDirPath, newDirPath)
	if err != nil {
		t.Fatalf("Failed to rename directory: %v", err)
	}

	// Verify old directory doesn't exist
	_, err = fs.GetAttr(ctx, oldDirPath)
	if err == nil {
		t.Error("Old directory should not exist after rename")
	}

	// Verify new directory exists
	attr, err := fs.GetAttr(ctx, newDirPath)
	if err != nil {
		t.Fatalf("Failed to get new directory attributes: %v", err)
	}
	if !attr.Mode.IsDir() {
		t.Error("Renamed path should be a directory")
	}

	// Verify file exists in new directory
	newFilePath := fmt.Sprintf("%s/file.txt", newDirPath)
	data, err := fs.ReadFile(ctx, newFilePath, 0, 0)
	if err != nil {
		t.Fatalf("Failed to read file in renamed directory: %v", err)
	}
	if string(data) != "test" {
		t.Errorf("Expected 'test', got %q", string(data))
	}

	// Cleanup
	fs.Remove(ctx, newFilePath)
	fs.Rmdir(ctx, newDirPath)
}

// TestRemoveFile tests removing files
func TestRemoveFile(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/test-remove-%d.txt", timestamp)

	// Create file
	err := fs.WriteFile(ctx, filePath, []byte("test"), 0)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Verify file exists
	_, err = fs.GetAttr(ctx, filePath)
	if err != nil {
		t.Fatalf("File should exist: %v", err)
	}

	// Remove file
	err = fs.Remove(ctx, filePath)
	if err != nil {
		t.Fatalf("Failed to remove file: %v", err)
	}

	// Verify file doesn't exist
	_, err = fs.GetAttr(ctx, filePath)
	if err == nil {
		t.Error("File should not exist after removal")
	}
}

// TestRemoveNonExistent tests removing non-existent file
func TestRemoveNonExistent(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	err := fs.Remove(ctx, "/nonexistent-file-12345")
	if err == nil {
		t.Error("Expected error when removing non-existent file")
	}
}

// TestXattrMultiple tests multiple extended attributes
func TestXattrMultiple(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/test-xattr-multi-%d.txt", timestamp)

	// Create file
	err := fs.WriteFile(ctx, filePath, []byte("test"), 0)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Set multiple xattrs
	xattrs := map[string][]byte{
		"user.test1": []byte("value1"),
		"user.test2": []byte("value2"),
		"user.test3": []byte("value3"),
	}

	for name, value := range xattrs {
		err = fs.SetXattr(ctx, filePath, name, value)
		if err != nil {
			t.Fatalf("Failed to set xattr %s: %v", name, err)
		}
	}

	// List xattrs
	names, err := fs.ListXattr(ctx, filePath)
	if err != nil {
		t.Fatalf("Failed to list xattrs: %v", err)
	}

	// Verify all xattrs are listed
	foundNames := make(map[string]bool)
	for _, name := range names {
		foundNames[name] = true
	}

	for name := range xattrs {
		if !foundNames[name] {
			t.Errorf("Xattr %s not found in list", name)
		}
	}

	// Verify all xattr values
	for name, expectedValue := range xattrs {
		value, err := fs.GetXattr(ctx, filePath, name)
		if err != nil {
			t.Fatalf("Failed to get xattr %s: %v", name, err)
		}
		if string(value) != string(expectedValue) {
			t.Errorf("Xattr %s: expected %q, got %q", name, string(expectedValue), string(value))
		}
	}

	// Cleanup
	fs.Remove(ctx, filePath)
}

// TestUtimensMultiple tests setting times multiple times
func TestUtimensMultiple(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	filePath := fmt.Sprintf("/test-utimens-multi-%d.txt", timestamp)

	// Create file
	err := fs.WriteFile(ctx, filePath, []byte("test"), 0)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Set times first time
	atime1 := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)
	mtime1 := time.Date(2021, 2, 2, 13, 0, 0, 0, time.UTC)
	err = fs.Utimens(ctx, filePath, atime1, mtime1)
	if err != nil {
		t.Fatalf("Failed to set times first time: %v", err)
	}

	// Verify times
	attr, err := fs.GetAttr(ctx, filePath)
	if err != nil {
		t.Fatalf("Failed to get file attributes: %v", err)
	}

	timeDiff := attr.Mtime.Unix() - mtime1.Unix()
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > 1 {
		t.Errorf("First mtime: expected %v, got %v (diff: %d)", mtime1, attr.Mtime, timeDiff)
	}

	// Set times second time
	atime2 := time.Date(2022, 3, 3, 14, 0, 0, 0, time.UTC)
	mtime2 := time.Date(2023, 4, 4, 15, 0, 0, 0, time.UTC)
	err = fs.Utimens(ctx, filePath, atime2, mtime2)
	if err != nil {
		t.Fatalf("Failed to set times second time: %v", err)
	}

	// Verify times updated
	attr, err = fs.GetAttr(ctx, filePath)
	if err != nil {
		t.Fatalf("Failed to get file attributes: %v", err)
	}

	timeDiff = attr.Mtime.Unix() - mtime2.Unix()
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > 1 {
		t.Errorf("Second mtime: expected %v, got %v (diff: %d)", mtime2, attr.Mtime, timeDiff)
	}

	// Cleanup
	fs.Remove(ctx, filePath)
}

// TestReadDirNested tests reading nested directory structure
func TestReadDirNested(t *testing.T) {
	RequireLocalStack(t)
	
	fs := SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
	ctx := context.Background()

	timestamp := time.Now().UnixNano()
	rootDir := fmt.Sprintf("/nested-%d", timestamp)
	subDir := fmt.Sprintf("%s/subdir", rootDir)

	// Create nested directory structure
	err := fs.Mkdir(ctx, rootDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create root directory: %v", err)
	}

	err = fs.Mkdir(ctx, subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create files in both directories
	rootFile := fmt.Sprintf("%s/root.txt", rootDir)
	subFile := fmt.Sprintf("%s/sub.txt", subDir)

	err = fs.WriteFile(ctx, rootFile, []byte("root"), 0)
	if err != nil {
		t.Fatalf("Failed to create root file: %v", err)
	}

	err = fs.WriteFile(ctx, subFile, []byte("sub"), 0)
	if err != nil {
		t.Fatalf("Failed to create sub file: %v", err)
	}

	// Read root directory
	rootEntries, err := fs.ReadDir(ctx, rootDir)
	if err != nil {
		t.Fatalf("Failed to read root directory: %v", err)
	}

	// Verify subdirectory is listed
	foundSubdir := false
	for _, entry := range rootEntries {
		if entry.Name == "subdir" && entry.IsDir {
			foundSubdir = true
			break
		}
	}
	if !foundSubdir {
		t.Error("Subdirectory not found in root directory listing")
	}

	// Read subdirectory
	subEntries, err := fs.ReadDir(ctx, subDir)
	if err != nil {
		t.Fatalf("Failed to read subdirectory: %v", err)
	}

	// Verify file is listed
	foundSubFile := false
	for _, entry := range subEntries {
		if entry.Name == "sub.txt" && !entry.IsDir {
			foundSubFile = true
			break
		}
	}
	if !foundSubFile {
		t.Error("File not found in subdirectory listing")
	}

	// Cleanup
	fs.Remove(ctx, subFile)
	fs.Rmdir(ctx, subDir)
	fs.Remove(ctx, rootFile)
	fs.Rmdir(ctx, rootDir)
}
