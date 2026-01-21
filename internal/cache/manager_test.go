package cache

import (
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	m := NewManager(100, 5*time.Minute, 50, 5, 4096)
	defer m.Close()

	if m == nil {
		t.Fatal("NewManager returned nil")
	}

	if m.statCache == nil {
		t.Error("Stat cache is nil")
	}

	if m.fdCache == nil {
		t.Error("FD cache is nil")
	}

	if m.tree == nil {
		t.Error("Tree is nil")
	}
}

func TestDefaultManager(t *testing.T) {
	m := DefaultManager()
	defer m.Close()

	if m == nil {
		t.Fatal("DefaultManager returned nil")
	}

	if m.statCache == nil {
		t.Error("Stat cache is nil")
	}

	if m.fdCache == nil {
		t.Error("FD cache is nil")
	}
}

func TestManager_GetStatCache(t *testing.T) {
	m := NewManager(100, 5*time.Minute, 50, 5, 4096)
	defer m.Close()

	statCache := m.GetStatCache()
	if statCache == nil {
		t.Fatal("GetStatCache returned nil")
	}

	// Test that it works
	attr := &CachedAttr{
		Mode:  0644,
		Size:  1024,
		Mtime: time.Now(),
		Uid:   1000,
		Gid:   1000,
	}

	statCache.Set("/test/file.txt", attr, nil)
	entry, found := statCache.Get("/test/file.txt")
	if !found {
		t.Error("Entry not found in stat cache")
	}

	if entry.Attr.Size != attr.Size {
		t.Errorf("Expected size %d, got %d", attr.Size, entry.Attr.Size)
	}
}

func TestManager_GetFdCache(t *testing.T) {
	m := NewManager(100, 5*time.Minute, 50, 5, 4096)
	defer m.Close()

	fdCache := m.GetFdCache()
	if fdCache == nil {
		t.Fatal("GetFdCache returned nil")
	}

	// Test that it works
	entity, err := fdCache.Open("/test/file.txt", 1024, time.Now())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if entity == nil {
		t.Fatal("Entity is nil")
	}
}

func TestManager_GetTree(t *testing.T) {
	m := NewManager(100, 5*time.Minute, 50, 5, 4096)
	defer m.Close()

	tree := m.GetTree()
	if tree == nil {
		t.Fatal("GetTree returned nil")
	}

	// Test that it works
	entry := &StatCacheEntry{
		Path: "/test/file.txt",
		Attr: &CachedAttr{
			Mode:  0644,
			Size:  1024,
			Mtime: time.Now(),
			Uid:   1000,
			Gid:   1000,
		},
		ExpiresAt: time.Now().Add(5 * time.Minute),
		LastAccess: time.Now(),
	}

	tree.Set("/test/file.txt", entry)
	node, found := tree.Get("/test/file.txt")
	if !found {
		t.Error("Node not found in tree")
	}

	if node.entry.Path != entry.Path {
		t.Errorf("Expected path '%s', got '%s'", entry.Path, node.entry.Path)
	}
}

func TestManager_Close(t *testing.T) {
	m := NewManager(100, 5*time.Minute, 50, 5, 4096)

	// Add some data
	attr := &CachedAttr{
		Mode:  0644,
		Size:  1024,
		Mtime: time.Now(),
		Uid:   1000,
		Gid:   1000,
	}
	m.statCache.Set("/test/file.txt", attr, nil)

	entity, _ := m.fdCache.Open("/test/file.txt", 1024, time.Now())

	// Close should clean up
	m.Close()

	// Verify cleanup (caches should be closed)
	// Note: We can't directly verify cleanup, but Close should not panic
	if m.statCache == nil || m.fdCache == nil {
		t.Error("Caches should not be nil after Close")
	}

	// Entity should be closed
	if entity.GetFile() != nil {
		t.Error("Entity file should be closed")
	}
}
