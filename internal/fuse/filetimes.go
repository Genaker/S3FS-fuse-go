package fuse

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Utimens sets file access and modification times
func (fs *Filesystem) Utimens(ctx context.Context, path string, atime, mtime time.Time) error {
	// Flush buffered data before updating metadata
	if err := fs.flushBufferedData(ctx, path); err != nil {
		return fmt.Errorf("failed to flush buffered data before utimens: %w", err)
	}
	
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

	backend := fs.getBackend()
	if backend == nil {
		return fmt.Errorf("no storage backend available")
	}

	var metadata map[string]string
	if isDir {
		// For directories, check for .keep marker
		keepPath := normalizedPath + ".keep"
		keepAttr, err := backend.GetAttr(ctx, keepPath)
		if err != nil {
			// No marker, create new metadata
			metadata = make(map[string]string)
		} else {
			// Convert attributes to metadata map
			metadata = make(map[string]string)
			metadata["mode"] = fmt.Sprintf("%o", keepAttr.Mode)
			metadata["uid"] = fmt.Sprintf("%d", keepAttr.Uid)
			metadata["gid"] = fmt.Sprintf("%d", keepAttr.Gid)
			metadata["mtime"] = fmt.Sprintf("%d", keepAttr.Mtime.Unix())
		}
	} else {
		// For files, get current attributes
		fileAttr, err := backend.GetAttr(ctx, normalizedPath)
		if err != nil {
			return fmt.Errorf("failed to get object metadata: %w", err)
		}
		// Convert attributes to metadata map
		metadata = make(map[string]string)
		metadata["mode"] = fmt.Sprintf("%o", fileAttr.Mode)
		metadata["uid"] = fmt.Sprintf("%d", fileAttr.Uid)
		metadata["gid"] = fmt.Sprintf("%d", fileAttr.Gid)
		metadata["mtime"] = fmt.Sprintf("%d", fileAttr.Mtime.Unix())
	}

	// HeadObject returns metadata keys WITHOUT "x-amz-meta-" prefix (AWS SDK strips it)
	// CopyObjectWithMetadata/PutObjectWithMetadata expect keys WITH prefix and will strip it
	// So we set both with and without prefix to ensure compatibility
	// Ensure mtime is actually updated (not before or equal to current mtime)
	// Always ensure mtime is at least 1 second after current time to guarantee update
	now := time.Now()
	currentMtime := mtime
	// Check mtime in metadata (HeadObject returns keys without prefix)
	currentMtimeStr := metadata["mtime"]
	if currentMtimeStr == "" {
		currentMtimeStr = metadata["x-amz-meta-mtime"]
	}
	if currentMtimeStr != "" {
		var currentMtimeUnix int64
		if _, err := fmt.Sscanf(currentMtimeStr, "%d", &currentMtimeUnix); err == nil {
			currentMtimeParsed := time.Unix(currentMtimeUnix, 0)
			// Only add 1 second if the new mtime equals the current mtime (same second)
			// This ensures update for "touch" operations while respecting explicit time settings
			if mtime.Unix() == currentMtimeParsed.Unix() {
				// Same second - add 1 second to guarantee update
				currentMtime = currentMtimeParsed.Add(time.Second)
			} else {
				// Different time - use the explicitly set time
				currentMtime = mtime
			}
		} else {
			// If parsing failed, use the passed mtime
			currentMtime = mtime
		}
	} else {
		// If no mtime in metadata, use the passed mtime
		currentMtime = mtime
	}
	metadata["x-amz-meta-atime"] = fmt.Sprintf("%d", atime.Unix())
	metadata["x-amz-meta-mtime"] = fmt.Sprintf("%d", currentMtime.Unix())
	metadata["x-amz-meta-ctime"] = fmt.Sprintf("%d", now.Unix())
	// Also set without prefix for consistency
	metadata["atime"] = fmt.Sprintf("%d", atime.Unix())
	metadata["mtime"] = fmt.Sprintf("%d", currentMtime.Unix())
	metadata["ctime"] = fmt.Sprintf("%d", now.Unix())

	// Update metadata using WriteWithMetadata
	if isDir {
		// Directory - update .keep marker with metadata
		keepPath := normalizedPath + ".keep"
		err = backend.WriteWithMetadata(ctx, keepPath, []byte{}, metadata)
		if err != nil {
			return fmt.Errorf("failed to set times on directory: %w", err)
		}
	} else {
		// File - read existing data (or use empty if file doesn't exist), then write back with new metadata
		existingData, err := backend.Read(ctx, normalizedPath)
		if err != nil {
			// File doesn't exist - create empty file with metadata
			existingData = []byte{}
		}
		err = backend.WriteWithMetadata(ctx, normalizedPath, existingData, metadata)
		if err != nil {
			return fmt.Errorf("failed to set times: %w", err)
		}
	}

	// Invalidate cache so GetAttr will read fresh metadata
	if fs.cache != nil {
		fs.cache.GetStatCache().Delete(path)
		// Update entity mtime if entity exists in FD cache
		fdCache := fs.cache.GetFdCache()
		if entity, found := fdCache.Get(normalizedPath); found {
			entity.SetMtime(mtime)
		}
	}

	return nil
}
