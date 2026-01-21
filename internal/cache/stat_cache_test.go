package cache

import (
	"testing"
	"time"
)

func TestNewStatCache(t *testing.T) {
	cache := NewStatCache(100, 5*time.Minute)
	defer cache.Close()

	if cache == nil {
		t.Fatal("NewStatCache returned nil")
	}

	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0, got %d", cache.Size())
	}
}

func TestStatCache_SetAndGet(t *testing.T) {
	cache := NewStatCache(100, 5*time.Minute)
	defer cache.Close()

	attr := &CachedAttr{
		Mode:  0644,
		Size:  1024,
		Mtime: time.Now(),
		Uid:   1000,
		Gid:   1000,
	}

	metadata := map[string]string{
		"x-amz-meta-mode": "0644",
		"x-amz-meta-uid":  "1000",
	}

	cache.Set("/test/file.txt", attr, metadata)

	entry, found := cache.Get("/test/file.txt")
	if !found {
		t.Fatal("Entry not found in cache")
	}

	if entry.Attr.Size != attr.Size {
		t.Errorf("Expected size %d, got %d", attr.Size, entry.Attr.Size)
	}

	if entry.Attr.Mode != attr.Mode {
		t.Errorf("Expected mode %o, got %o", attr.Mode, entry.Attr.Mode)
	}
}

func TestStatCache_Expiration(t *testing.T) {
	cache := NewStatCache(100, 100*time.Millisecond)
	defer cache.Close()

	attr := &CachedAttr{
		Mode:  0644,
		Size:  1024,
		Mtime: time.Now(),
		Uid:   1000,
		Gid:   1000,
	}

	cache.Set("/test/file.txt", attr, nil)

	// Entry should be found immediately
	_, found := cache.Get("/test/file.txt")
	if !found {
		t.Fatal("Entry not found immediately after setting")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Entry should be expired
	_, found = cache.Get("/test/file.txt")
	if found {
		t.Error("Entry should be expired but was found")
	}
}

func TestStatCache_Delete(t *testing.T) {
	cache := NewStatCache(100, 5*time.Minute)
	defer cache.Close()

	attr := &CachedAttr{
		Mode:  0644,
		Size:  1024,
		Mtime: time.Now(),
		Uid:   1000,
		Gid:   1000,
	}

	cache.Set("/test/file.txt", attr, nil)
	cache.Delete("/test/file.txt")

	_, found := cache.Get("/test/file.txt")
	if found {
		t.Error("Entry should be deleted but was found")
	}
}

func TestStatCache_Clear(t *testing.T) {
	cache := NewStatCache(100, 5*time.Minute)
	defer cache.Close()

	attr := &CachedAttr{
		Mode:  0644,
		Size:  1024,
		Mtime: time.Now(),
		Uid:   1000,
		Gid:   1000,
	}

	cache.Set("/test/file1.txt", attr, nil)
	cache.Set("/test/file2.txt", attr, nil)

	if cache.Size() != 2 {
		t.Errorf("Expected cache size 2, got %d", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", cache.Size())
	}
}

func TestStatCache_Truncation(t *testing.T) {
	cache := NewStatCache(5, 5*time.Minute)
	defer cache.Close()

	attr := &CachedAttr{
		Mode:  0644,
		Size:  1024,
		Mtime: time.Now(),
		Uid:   1000,
		Gid:   1000,
	}

	// Add more entries than max size
	for i := 0; i < 10; i++ {
		path := "/test/file" + string(rune('0'+i)) + ".txt"
		cache.Set(path, attr, nil)
		time.Sleep(10 * time.Millisecond) // Ensure different last access times
	}

	// Cache should be truncated to max size
	if cache.Size() > 5 {
		t.Errorf("Expected cache size <= 5, got %d", cache.Size())
	}
}

func TestStatCache_SetSymlink(t *testing.T) {
	cache := NewStatCache(100, 5*time.Minute)
	defer cache.Close()

	cache.SetSymlink("/test/link", "/target/file.txt")

	target, found := cache.GetSymlink("/test/link")
	if !found {
		t.Fatal("Symlink not found in cache")
	}

	if target != "/target/file.txt" {
		t.Errorf("Expected symlink target '/target/file.txt', got '%s'", target)
	}
}

func TestStatCache_GetSymlink_NonExistent(t *testing.T) {
	cache := NewStatCache(100, 5*time.Minute)
	defer cache.Close()

	_, found := cache.GetSymlink("/test/nonexistent")
	if found {
		t.Error("Non-existent symlink should not be found")
	}
}

func TestStatCache_SetMaxSize(t *testing.T) {
	cache := NewStatCache(100, 5*time.Minute)
	defer cache.Close()

	attr := &CachedAttr{
		Mode:  0644,
		Size:  1024,
		Mtime: time.Now(),
		Uid:   1000,
		Gid:   1000,
	}

	// Add entries
	for i := 0; i < 10; i++ {
		path := "/test/file" + string(rune('0'+i)) + ".txt"
		cache.Set(path, attr, nil)
		time.Sleep(10 * time.Millisecond)
	}

	// Reduce max size
	cache.SetMaxSize(3)

	// Cache should be truncated
	if cache.Size() > 3 {
		t.Errorf("Expected cache size <= 3, got %d", cache.Size())
	}
}

func TestStatCache_SetTTL(t *testing.T) {
	cache := NewStatCache(100, 5*time.Minute)
	defer cache.Close()

	cache.SetTTL(200 * time.Millisecond)

	attr := &CachedAttr{
		Mode:  0644,
		Size:  1024,
		Mtime: time.Now(),
		Uid:   1000,
		Gid:   1000,
	}

	cache.Set("/test/file.txt", attr, nil)

	// Entry should be found immediately
	_, found := cache.Get("/test/file.txt")
	if !found {
		t.Fatal("Entry not found immediately after setting")
	}

	// Wait for expiration
	time.Sleep(250 * time.Millisecond)

	// Entry should be expired
	_, found = cache.Get("/test/file.txt")
	if found {
		t.Error("Entry should be expired but was found")
	}
}

func TestStatCache_LastAccess(t *testing.T) {
	cache := NewStatCache(100, 5*time.Minute)
	defer cache.Close()

	attr := &CachedAttr{
		Mode:  0644,
		Size:  1024,
		Mtime: time.Now(),
		Uid:   1000,
		Gid:   1000,
	}

	cache.Set("/test/file.txt", attr, nil)

	time.Sleep(50 * time.Millisecond)

	entry1, _ := cache.Get("/test/file.txt")
	access1 := entry1.LastAccess

	time.Sleep(50 * time.Millisecond)

	entry2, _ := cache.Get("/test/file.txt")
	access2 := entry2.LastAccess

	if !access2.After(access1) {
		t.Error("Last access time should be updated on Get")
	}
}
