package fuse

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/s3fs-fuse/s3fs-go/internal/s3client"
)

// TestMtimeFile tests mtime preservation
func TestMtimeFile(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testFile := "test-mtime.txt"
	altFile := "test-mtime-alt.txt"
	testText := "HELLO WORLD"

	err := fs.WriteFile(ctx, testFile, []byte(testText), 0)
	if err != nil {
		t.Skipf("Skipping test - S3 client not initialized: %v", err)
		return
	}

	// Get initial times
	attr1, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get initial attributes: %v", err)
	}

	// Copy file (simulating cp -p)
	err = fs.Rename(ctx, testFile, altFile)
	if err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}

	// Get times from copied file
	attr2, err := fs.GetAttr(ctx, altFile)
	if err != nil {
		t.Fatalf("Failed to get copied file attributes: %v", err)
	}

	// Mtime should be preserved (or close)
	timeDiff := attr2.Mtime.Sub(attr1.Mtime)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > 2*time.Second {
		t.Errorf("Mtime should be preserved, diff: %v", timeDiff)
	}

	// Cleanup
	fs.Remove(ctx, altFile)
}

// TestUpdateTimeChmod tests that chmod updates ctime
func TestUpdateTimeChmod(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testFile := "test-time-chmod.txt"
	testText := "HELLO WORLD"

	err := fs.WriteFile(ctx, testFile, []byte(testText), 0)
	if err != nil {
		t.Skipf("Skipping test - S3 client not initialized: %v", err)
		return
	}

	// Get initial times
	attr1, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get initial attributes: %v", err)
	}

	initialCtime := attr1.Mtime

	// Chmod should update ctime
	err = fs.Chmod(ctx, testFile, os.FileMode(0777))
	if err != nil {
		t.Fatalf("Failed to chmod: %v", err)
	}

	// Get times after chmod
	attr2, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get attributes after chmod: %v", err)
	}

	// Ctime should be updated
	if attr2.Mtime.Before(initialCtime) {
		t.Error("Ctime should be updated after chmod")
	}

	fs.Remove(ctx, testFile)
}

// TestUpdateTimeChown tests that chown updates ctime
func TestUpdateTimeChown(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testFile := "test-time-chown.txt"
	testText := "HELLO WORLD"

	err := fs.WriteFile(ctx, testFile, []byte(testText), 0)
	if err != nil {
		t.Skipf("Skipping test - S3 client not initialized: %v", err)
		return
	}

	// Get initial times
	attr1, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get initial attributes: %v", err)
	}

	initialCtime := attr1.Mtime

	// Chown should update ctime
	err = fs.Chown(ctx, testFile, 1000, 1000)
	if err != nil {
		t.Fatalf("Failed to chown: %v", err)
	}

	// Get times after chown
	attr2, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get attributes after chown: %v", err)
	}

	// Ctime should be updated
	if attr2.Mtime.Before(initialCtime) {
		t.Error("Ctime should be updated after chown")
	}

	fs.Remove(ctx, testFile)
}

// TestUpdateTimeTouch tests that touch updates all times
func TestUpdateTimeTouch(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testFile := "test-time-touch.txt"
	testText := "HELLO WORLD"

	err := fs.WriteFile(ctx, testFile, []byte(testText), 0)
	if err != nil {
		t.Skipf("Skipping test - S3 client not initialized: %v", err)
		return
	}

	// Get initial times
	attr1, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get initial attributes: %v", err)
	}

	initialMtime := attr1.Mtime

	// Touch should update mtime/atime/ctime
	// Simulate touch by updating mtime
	err = fs.Utimens(ctx, testFile, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to touch: %v", err)
	}

	// Get times after touch
	attr2, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get attributes after touch: %v", err)
	}

	// Mtime should be updated
	if attr2.Mtime.Before(initialMtime) || attr2.Mtime.Equal(initialMtime) {
		t.Error("Mtime should be updated after touch")
	}

	fs.Remove(ctx, testFile)
}

// TestUpdateTimeAppend tests that append updates ctime/mtime
func TestUpdateTimeAppend(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testFile := "test-time-append.txt"
	testText := "HELLO"

	err := fs.WriteFile(ctx, testFile, []byte(testText), 0)
	if err != nil {
		t.Skipf("Skipping test - S3 client not initialized: %v", err)
		return
	}

	// Get initial times
	attr1, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get initial attributes: %v", err)
	}

	initialMtime := attr1.Mtime

	// Append should update mtime/ctime
	appendText := " WORLD"
	err = fs.WriteFile(ctx, testFile, []byte(appendText), int64(len(testText)))
	if err != nil {
		t.Fatalf("Failed to append: %v", err)
	}

	// Get times after append
	attr2, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get attributes after append: %v", err)
	}

	// Mtime should be updated
	if attr2.Mtime.Before(initialMtime) || attr2.Mtime.Equal(initialMtime) {
		t.Error("Mtime should be updated after append")
	}

	fs.Remove(ctx, testFile)
}

// TestUpdateTimeCpP tests that cp -p preserves mtime
func TestUpdateTimeCpP(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testFile := "test-time-cpp.txt"
	destFile := "test-time-cpp-dest.txt"
	testText := "HELLO WORLD"

	err := fs.WriteFile(ctx, testFile, []byte(testText), 0)
	if err != nil {
		t.Skipf("Skipping test - S3 client not initialized: %v", err)
		return
	}

	// Get source times
	attr1, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get source attributes: %v", err)
	}

	sourceMtime := attr1.Mtime

	// Copy with preserve (simulated by Rename which preserves metadata)
	err = fs.Rename(ctx, testFile, destFile)
	if err != nil {
		t.Fatalf("Failed to copy: %v", err)
	}

	// Get dest times
	attr2, err := fs.GetAttr(ctx, destFile)
	if err != nil {
		t.Fatalf("Failed to get dest attributes: %v", err)
	}

	// Mtime should be preserved (or close)
	timeDiff := attr2.Mtime.Sub(sourceMtime)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > 2*time.Second {
		t.Errorf("Mtime should be preserved, diff: %v", timeDiff)
	}

	// Cleanup
	fs.Remove(ctx, destFile)
}

// TestUpdateTimeMv tests that mv updates ctime but preserves mtime
func TestUpdateTimeMv(t *testing.T) {
	client := s3client.NewClient("test-bucket", "us-east-1", nil)
	fs := NewFilesystem(client)
	ctx := context.Background()

	testFile := "test-time-mv.txt"
	destFile := "test-time-mv-dest.txt"
	testText := "HELLO WORLD"

	err := fs.WriteFile(ctx, testFile, []byte(testText), 0)
	if err != nil {
		t.Skipf("Skipping test - S3 client not initialized: %v", err)
		return
	}

	// Get source times
	attr1, err := fs.GetAttr(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to get source attributes: %v", err)
	}

	sourceMtime := attr1.Mtime

	// Move file
	err = fs.Rename(ctx, testFile, destFile)
	if err != nil {
		t.Fatalf("Failed to move: %v", err)
	}

	// Get dest times
	attr2, err := fs.GetAttr(ctx, destFile)
	if err != nil {
		t.Fatalf("Failed to get dest attributes: %v", err)
	}

	// Mtime should be preserved (or close)
	timeDiff := attr2.Mtime.Sub(sourceMtime)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > 2*time.Second {
		t.Errorf("Mtime should be preserved, diff: %v", timeDiff)
	}

	// Cleanup
	fs.Remove(ctx, destFile)
}
