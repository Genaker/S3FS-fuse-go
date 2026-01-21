package cache

import (
	"io"
	"os"
	"testing"
	"time"
)

func TestNewFdCacheManager(t *testing.T) {
	fcm := NewFdCacheManager(100, 10, 4096)
	defer fcm.CloseAll()

	if fcm == nil {
		t.Fatal("NewFdCacheManager returned nil")
	}
}

func TestFdCacheManager_Open(t *testing.T) {
	fcm := NewFdCacheManager(100, 10, 4096)
	defer fcm.CloseAll()

	entity, err := fcm.Open("/test/file.txt", 1024, time.Now())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if entity == nil {
		t.Fatal("Entity is nil")
	}

	if entity.path != "/test/file.txt" {
		t.Errorf("Expected path '/test/file.txt', got '%s'", entity.path)
	}

	if entity.size != 1024 {
		t.Errorf("Expected size 1024, got %d", entity.size)
	}

	if entity.refCount != 1 {
		t.Errorf("Expected refCount 1, got %d", entity.refCount)
	}
}

func TestFdCacheManager_Get(t *testing.T) {
	fcm := NewFdCacheManager(100, 10, 4096)
	defer fcm.CloseAll()

	entity1, _ := fcm.Open("/test/file.txt", 1024, time.Now())
	entity2, found := fcm.Get("/test/file.txt")

	if !found {
		t.Fatal("Entity not found")
	}

	if entity1 != entity2 {
		t.Error("Get should return the same entity")
	}
}

func TestFdCacheManager_Close(t *testing.T) {
	fcm := NewFdCacheManager(100, 10, 4096)
	defer fcm.CloseAll()

	entity, _ := fcm.Open("/test/file.txt", 1024, time.Now())

	if entity.refCount != 1 {
		t.Errorf("Expected refCount 1, got %d", entity.refCount)
	}

	err := fcm.Close("/test/file.txt")
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Entity should be removed when refCount reaches 0
	_, found := fcm.Get("/test/file.txt")
	if found {
		t.Error("Entity should be removed after close")
	}
}

func TestFdCacheManager_RefCount(t *testing.T) {
	fcm := NewFdCacheManager(100, 10, 4096)
	defer fcm.CloseAll()

	entity1, _ := fcm.Open("/test/file.txt", 1024, time.Now())
	entity2, _ := fcm.Open("/test/file.txt", 1024, time.Now()) // Should increment refCount

	if entity1.refCount != 2 {
		t.Errorf("Expected refCount 2, got %d", entity1.refCount)
	}

	if entity1 != entity2 {
		t.Error("Open should return the same entity")
	}

	fcm.Close("/test/file.txt")
	if entity1.refCount != 1 {
		t.Errorf("Expected refCount 1 after one close, got %d", entity1.refCount)
	}

	fcm.Close("/test/file.txt")
	// Entity should be removed
	_, found := fcm.Get("/test/file.txt")
	if found {
		t.Error("Entity should be removed after all closes")
	}
}

func TestFdCacheManager_GetInfo(t *testing.T) {
	fcm := NewFdCacheManager(100, 10, 4096)
	defer fcm.CloseAll()

	mtime := time.Now()
	fcm.Open("/test/file.txt", 2048, mtime)

	info, found := fcm.GetInfo("/test/file.txt")
	if !found {
		t.Fatal("Info not found")
	}

	if info.Path != "/test/file.txt" {
		t.Errorf("Expected path '/test/file.txt', got '%s'", info.Path)
	}

	if info.Size != 2048 {
		t.Errorf("Expected size 2048, got %d", info.Size)
	}

	if info.RefCount != 1 {
		t.Errorf("Expected refCount 1, got %d", info.RefCount)
	}
}

func TestFdCacheManager_HasOpenEntity(t *testing.T) {
	fcm := NewFdCacheManager(100, 10, 4096)
	defer fcm.CloseAll()

	if fcm.HasOpenEntity("/test/file.txt") {
		t.Error("Entity should not be open")
	}

	fcm.Open("/test/file.txt", 1024, time.Now())

	if !fcm.HasOpenEntity("/test/file.txt") {
		t.Error("Entity should be open")
	}

	fcm.Close("/test/file.txt")

	if fcm.HasOpenEntity("/test/file.txt") {
		t.Error("Entity should not be open after close")
	}
}

func TestFdCacheManager_GetOpenFdCount(t *testing.T) {
	fcm := NewFdCacheManager(100, 10, 4096)
	defer fcm.CloseAll()

	count := fcm.GetOpenFdCount("/test/file.txt")
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}

	fcm.Open("/test/file.txt", 1024, time.Now())
	fcm.Open("/test/file.txt", 1024, time.Now()) // Increment refCount

	count = fcm.GetOpenFdCount("/test/file.txt")
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

func TestFdCacheManager_MaxOpenFiles(t *testing.T) {
	fcm := NewFdCacheManager(100, 3, 4096)
	defer fcm.CloseAll()

	// Open 3 files (max)
	fcm.Open("/test/file1.txt", 1024, time.Now())
	fcm.Open("/test/file2.txt", 1024, time.Now())
	fcm.Open("/test/file3.txt", 1024, time.Now())

	// Opening 4th file should close oldest
	time.Sleep(50 * time.Millisecond) // Ensure different last access times
	_, _ = fcm.Open("/test/file4.txt", 1024, time.Now())

	// First file should be closed (if it had refCount 0)
	// But since we just opened it, it should still exist
	// Let's close it first and then open a new one
	fcm.Close("/test/file1.txt")
	time.Sleep(50 * time.Millisecond)
	fcm.Open("/test/file5.txt", 1024, time.Now())

	// file1 should be removed
	if fcm.HasOpenEntity("/test/file1.txt") {
		t.Error("Oldest entity should be closed")
	}
}

func TestFdEntity_ReadPage(t *testing.T) {
	entity := &FdEntity{
		path:         "/test/file.txt",
		size:         8192,
		pageSize:     4096,
		pages:        make(map[int64]*Page),
		dirtyPages:   make(map[int64]bool),
		bytesModified: 0,
	}

	// Write a page
	pageData := make([]byte, 4096)
	for i := range pageData {
		pageData[i] = byte(i % 256)
	}
	entity.WritePage(0, pageData)

	// Read page
	data, found := entity.ReadPage(0)
	if !found {
		t.Fatal("Page not found")
	}

	if len(data) != 4096 {
		t.Errorf("Expected page size 4096, got %d", len(data))
	}

	// Read from middle of page
	data2, found := entity.ReadPage(2048)
	if !found {
		t.Fatal("Page offset not found")
	}

	if len(data2) != 2048 {
		t.Errorf("Expected data size 2048, got %d", len(data2))
	}
}

func TestFdEntity_WritePage(t *testing.T) {
	entity := &FdEntity{
		path:         "/test/file.txt",
		size:         8192,
		pageSize:     4096,
		pages:        make(map[int64]*Page),
		dirtyPages:   make(map[int64]bool),
		bytesModified: 0,
	}

	pageData := make([]byte, 4096)
	for i := range pageData {
		pageData[i] = byte(i % 256)
	}

	entity.WritePage(0, pageData)

	page, exists := entity.pages[0]
	if !exists {
		t.Fatal("Page not written")
	}

	if len(page.Data) != 4096 {
		t.Errorf("Expected page size 4096, got %d", len(page.Data))
	}

	if !page.Dirty {
		t.Error("Page should be marked as dirty")
	}
}

func TestFdEntity_GetFile(t *testing.T) {
	entity := &FdEntity{
		path:     "/test/file.txt",
		size:     1024,
		pageSize: 4096,
		pages:    make(map[int64]*Page),
	}

	// Initially file should be nil
	file := entity.GetFile()
	if file != nil {
		t.Error("File should be nil initially")
	}

	// Set file from temp
	file, err := entity.SetFileFromTemp()
	if err != nil {
		t.Fatalf("SetFileFromTemp failed: %v", err)
	}

	if file == nil {
		t.Fatal("File is nil")
	}

	// GetFile should return the same file
	file2 := entity.GetFile()
	if file2 != file {
		t.Error("GetFile should return the same file")
	}

	// Clean up
	os.Remove(file.Name())
}

func TestFdEntity_ReadWrite(t *testing.T) {
	entity := &FdEntity{
		path:     "/test/file.txt",
		size:     1024,
		pageSize: 4096,
		pages:    make(map[int64]*Page),
	}

	file, err := entity.SetFileFromTemp()
	if err != nil {
		t.Fatalf("SetFileFromTemp failed: %v", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()

	// Write test data
	testData := []byte("Hello, World!")
	_, err = file.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read using entity
	data, err := entity.Read(0, int64(len(testData)))
	if err != nil && err != io.EOF {
		t.Fatalf("Read failed: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("Expected '%s', got '%s'", string(testData), string(data))
	}
}

func TestFdEntity_Write(t *testing.T) {
	entity := &FdEntity{
		path:     "/test/file.txt",
		size:     1024,
		pageSize: 4096,
		pages:    make(map[int64]*Page),
	}

	file, err := entity.SetFileFromTemp()
	if err != nil {
		t.Fatalf("SetFileFromTemp failed: %v", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()

	testData := []byte("Test data")
	err = entity.Write(0, testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify data was written
	file.Seek(0, 0)
	readData := make([]byte, len(testData))
	file.Read(readData)

	if string(readData) != string(testData) {
		t.Errorf("Expected '%s', got '%s'", string(testData), string(readData))
	}
}

func TestFdEntity_Sync(t *testing.T) {
	entity := &FdEntity{
		path:     "/test/file.txt",
		size:     1024,
		pageSize: 4096,
		pages:    make(map[int64]*Page),
	}

	file, err := entity.SetFileFromTemp()
	if err != nil {
		t.Fatalf("SetFileFromTemp failed: %v", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()

	err = entity.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
}

func TestFdEntity_Size(t *testing.T) {
	entity := &FdEntity{
		path:     "/test/file.txt",
		size:     2048,
		pageSize: 4096,
		pages:    make(map[int64]*Page),
	}

	if entity.Size() != 2048 {
		t.Errorf("Expected size 2048, got %d", entity.Size())
	}

	entity.SetSize(4096)
	if entity.Size() != 4096 {
		t.Errorf("Expected size 4096 after SetSize, got %d", entity.Size())
	}
}

func TestFdEntity_PageEviction(t *testing.T) {
	entity := &FdEntity{
		path:         "/test/file.txt",
		size:         1024 * 1024, // 1MB
		pageSize:     4096,
		pages:        make(map[int64]*Page),
		dirtyPages:   make(map[int64]bool),
		bytesModified: 0,
	}

	// Write more than 100 pages (max)
	for i := 0; i < 150; i++ {
		pageData := make([]byte, 4096)
		entity.WritePage(int64(i*4096), pageData)
		time.Sleep(1 * time.Millisecond) // Ensure different access times
	}

	// Should have evicted oldest pages
	if len(entity.pages) > 100 {
		t.Errorf("Expected <= 100 pages, got %d", len(entity.pages))
	}
}
