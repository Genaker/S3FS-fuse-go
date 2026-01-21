package fuse

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Utimens sets file access and modification times
func (fs *Filesystem) Utimens(ctx context.Context, path string, atime, mtime time.Time) error {
	normalizedPath := fs.normalizePath(path)

	// Check if it's a directory by checking attributes
	attr, err := fs.GetAttr(ctx, path)
	isDir := false
	if err == nil && attr.Mode.IsDir() {
		isDir = true
		// Normalize directory path
		if !strings.HasSuffix(normalizedPath, "/") {
			normalizedPath += "/"
		}
	}

	var metadata map[string]string
	if isDir {
		// For directories, check for .keep marker
		keepPath := normalizedPath + ".keep"
		metadata, err = fs.client.HeadObject(ctx, keepPath)
		if err != nil {
			// No marker, create new metadata
			metadata = make(map[string]string)
		}
	} else {
		// For files, get current metadata
		metadata, err = fs.client.HeadObject(ctx, normalizedPath)
		if err != nil {
			return fmt.Errorf("failed to get object metadata: %w", err)
		}
	}

	// Update time metadata
	metadata["x-amz-meta-atime"] = fmt.Sprintf("%d", atime.Unix())
	metadata["x-amz-meta-mtime"] = fmt.Sprintf("%d", mtime.Unix())
	metadata["x-amz-meta-ctime"] = fmt.Sprintf("%d", time.Now().Unix())

	// Update metadata using CopyObject
	if isDir {
		// Directory - update .keep marker with metadata
		keepPath := normalizedPath + ".keep"
		err = fs.client.PutObjectWithMetadata(ctx, keepPath, []byte{}, metadata)
		if err != nil {
			return fmt.Errorf("failed to set times on directory: %w", err)
		}
	} else {
		// File - use CopyObject with metadata directive
		err = fs.client.CopyObjectWithMetadata(ctx, normalizedPath, normalizedPath, metadata)
		if err != nil {
			return fmt.Errorf("failed to set times: %w", err)
		}
	}

	return nil
}
