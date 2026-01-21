package cache

import (
	"os"
	"testing"
	"time"
)

func TestNewCacheTree(t *testing.T) {
	tree := NewCacheTree(100)

	if tree == nil {
		t.Fatal("NewCacheTree returned nil")
	}

	if tree.root == nil {
		t.Fatal("Tree root is nil")
	}
}

func TestCacheTree_SetAndGet(t *testing.T) {
	tree := NewCacheTree(100)

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
		t.Fatal("Node not found in tree")
	}

	if node.entry.Path != entry.Path {
		t.Errorf("Expected path '%s', got '%s'", entry.Path, node.entry.Path)
	}
}

func TestCacheTree_Delete(t *testing.T) {
	tree := NewCacheTree(100)

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
	tree.Delete("/test/file.txt")

	_, found := tree.Get("/test/file.txt")
	if found {
		t.Error("Node should be deleted but was found")
	}
}

func TestCacheTree_GetChildren(t *testing.T) {
	tree := NewCacheTree(100)

	// Create parent directory entry
	parentEntry := &StatCacheEntry{
		Path: "/test",
		Attr: &CachedAttr{
			Mode:  uint32(os.ModeDir | 0755),
			Size:  4096,
			Mtime: time.Now(),
			Uid:   1000,
			Gid:   1000,
		},
		ExpiresAt: time.Now().Add(5 * time.Minute),
		LastAccess: time.Now(),
	}

	// Create child entries
	child1Entry := &StatCacheEntry{
		Path: "/test/file1.txt",
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

	child2Entry := &StatCacheEntry{
		Path: "/test/file2.txt",
		Attr: &CachedAttr{
			Mode:  0644,
			Size:  2048,
			Mtime: time.Now(),
			Uid:   1000,
			Gid:   1000,
		},
		ExpiresAt: time.Now().Add(5 * time.Minute),
		LastAccess: time.Now(),
	}

	tree.Set("/test", parentEntry)
	tree.Set("/test/file1.txt", child1Entry)
	tree.Set("/test/file2.txt", child2Entry)

	children := tree.GetChildren("/test")
	if len(children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(children))
	}
}

func TestCacheTree_Clear(t *testing.T) {
	tree := NewCacheTree(100)

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
	tree.Clear()

	_, found := tree.Get("/test/file.txt")
	if found {
		t.Error("Node should be cleared but was found")
	}
}

func TestCacheTree_NestedPaths(t *testing.T) {
	tree := NewCacheTree(100)

	entry1 := &StatCacheEntry{
		Path: "/a/b/c/file.txt",
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

	entry2 := &StatCacheEntry{
		Path: "/a/b/d/file.txt",
		Attr: &CachedAttr{
			Mode:  0644,
			Size:  2048,
			Mtime: time.Now(),
			Uid:   1000,
			Gid:   1000,
		},
		ExpiresAt: time.Now().Add(5 * time.Minute),
		LastAccess: time.Now(),
	}

	tree.Set("/a/b/c/file.txt", entry1)
	tree.Set("/a/b/d/file.txt", entry2)

	node1, found1 := tree.Get("/a/b/c/file.txt")
	node2, found2 := tree.Get("/a/b/d/file.txt")

	if !found1 || !found2 {
		t.Fatal("Nested paths not found")
	}

	if node1.entry.Attr.Size != 1024 {
		t.Errorf("Expected size 1024, got %d", node1.entry.Attr.Size)
	}

	if node2.entry.Attr.Size != 2048 {
		t.Errorf("Expected size 2048, got %d", node2.entry.Attr.Size)
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"/", []string{}},
		{"/test", []string{"test"}},
		{"/test/file.txt", []string{"test", "file.txt"}},
		{"/a/b/c", []string{"a", "b", "c"}},
		{"test/file.txt", []string{"test", "file.txt"}},
	}

	for _, tt := range tests {
		result := splitPath(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("splitPath(%q) length: expected %d, got %d", tt.input, len(tt.expected), len(result))
			continue
		}

		for i, part := range tt.expected {
			if i >= len(result) || result[i] != part {
				t.Errorf("splitPath(%q)[%d]: expected '%s', got '%s'", tt.input, i, part, result[i])
			}
		}
	}
}
