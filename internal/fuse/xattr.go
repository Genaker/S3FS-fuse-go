package fuse

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SetXattr sets an extended attribute
func (fs *Filesystem) SetXattr(ctx context.Context, path string, name string, value []byte) error {
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

	// Store xattr in metadata with prefix
	// Use base64 encoding for binary values
	xattrKey := fmt.Sprintf("x-amz-meta-xattr-%s", name)
	metadata[xattrKey] = string(value)
	// Update ctime when setting xattr
	metadata["x-amz-meta-ctime"] = fmt.Sprintf("%d", time.Now().Unix())

	// Update metadata using CopyObject
	if isDir {
		// Directory - update .keep marker with metadata
		keepPath := normalizedPath + ".keep"
		err = fs.client.PutObjectWithMetadata(ctx, keepPath, []byte{}, metadata)
		if err != nil {
			return fmt.Errorf("failed to set xattr on directory: %w", err)
		}
	} else {
		// File - use CopyObject with metadata directive
		err = fs.client.CopyObjectWithMetadata(ctx, normalizedPath, normalizedPath, metadata)
		if err != nil {
			return fmt.Errorf("failed to set xattr: %w", err)
		}
	}

	return nil
}

// GetXattr gets an extended attribute value
func (fs *Filesystem) GetXattr(ctx context.Context, path string, name string) ([]byte, error) {
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
			return nil, fmt.Errorf("extended attribute not found: %w", err)
		}
	} else {
		// For files, get metadata
		metadata, err = fs.client.HeadObject(ctx, normalizedPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get object metadata: %w", err)
		}
	}

	// Look for xattr in metadata
	xattrKey := fmt.Sprintf("x-amz-meta-xattr-%s", name)
	value, ok := metadata[xattrKey]
	if !ok {
		return nil, fmt.Errorf("extended attribute '%s' not found", name)
	}

	return []byte(value), nil
}

// ListXattr lists all extended attribute names
func (fs *Filesystem) ListXattr(ctx context.Context, path string) ([]string, error) {
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
			return []string{}, nil // No xattrs
		}
	} else {
		// For files, get metadata
		metadata, err = fs.client.HeadObject(ctx, normalizedPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get object metadata: %w", err)
		}
	}

	// Extract xattr names from metadata keys
	var names []string
	prefix := "x-amz-meta-xattr-"
	for key := range metadata {
		if strings.HasPrefix(key, prefix) {
			name := strings.TrimPrefix(key, prefix)
			names = append(names, name)
		}
	}

	return names, nil
}

// RemoveXattr removes an extended attribute
func (fs *Filesystem) RemoveXattr(ctx context.Context, path string, name string) error {
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
			return fmt.Errorf("extended attribute not found: %w", err)
		}
	} else {
		// For files, get current metadata
		metadata, err = fs.client.HeadObject(ctx, normalizedPath)
		if err != nil {
			return fmt.Errorf("failed to get object metadata: %w", err)
		}
	}

	// Remove xattr from metadata
	xattrKey := fmt.Sprintf("x-amz-meta-xattr-%s", name)
	if _, ok := metadata[xattrKey]; !ok {
		return fmt.Errorf("extended attribute '%s' not found", name)
	}

	delete(metadata, xattrKey)

	// Update metadata
	if isDir {
		// Directory - update .keep marker
		keepPath := normalizedPath + ".keep"
		err = fs.client.PutObjectWithMetadata(ctx, keepPath, []byte{}, metadata)
		if err != nil {
			return fmt.Errorf("failed to remove xattr from directory: %w", err)
		}
	} else {
		// File - use CopyObject with metadata directive
		err = fs.client.CopyObjectWithMetadata(ctx, normalizedPath, normalizedPath, metadata)
		if err != nil {
			return fmt.Errorf("failed to remove xattr: %w", err)
		}
	}

	return nil
}
