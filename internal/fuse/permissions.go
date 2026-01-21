package fuse

import (
	"context"
	"fmt"
	"os"
	"time"
)

// Chmod changes file permissions
func (fs *Filesystem) Chmod(ctx context.Context, path string, mode os.FileMode) error {
	normalizedPath := fs.normalizePath(path)

	// Get current attributes
	attr, err := fs.GetAttr(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to get file attributes: %w", err)
	}

	// For directories, mode is conceptual (no actual object to update)
	if attr.Mode.IsDir() {
		return nil
	}

	// Use CopyObject with metadata directive to update metadata without re-uploading
	modeStr := fmt.Sprintf("%04o", mode&0777)
	now := time.Now()
	metadata := map[string]string{
		"x-amz-meta-mode": modeStr,
		"x-amz-meta-ctime": fmt.Sprintf("%d", now.Unix()),
	}

	err = fs.client.CopyObjectWithMetadata(ctx, normalizedPath, normalizedPath, metadata)
	if err != nil {
		return fmt.Errorf("failed to update file mode: %w", err)
	}

	return nil
}

// Chown changes file ownership
func (fs *Filesystem) Chown(ctx context.Context, path string, uid, gid uint32) error {
	normalizedPath := fs.normalizePath(path)

	// Get current attributes
	attr, err := fs.GetAttr(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to get file attributes: %w", err)
	}

	// For directories, ownership is conceptual (no actual object to update)
	if attr.Mode.IsDir() {
		return nil
	}

	// Use CopyObject with metadata directive to update metadata without re-uploading
	now := time.Now()
	metadata := map[string]string{
		"x-amz-meta-uid": fmt.Sprintf("%d", uid),
		"x-amz-meta-gid": fmt.Sprintf("%d", gid),
		"x-amz-meta-ctime": fmt.Sprintf("%d", now.Unix()),
	}

	err = fs.client.CopyObjectWithMetadata(ctx, normalizedPath, normalizedPath, metadata)
	if err != nil {
		return fmt.Errorf("failed to update file ownership: %w", err)
	}

	return nil
}
