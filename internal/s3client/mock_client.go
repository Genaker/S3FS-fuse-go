package s3client

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockClient is an in-memory mock implementation of the S3 client for unit tests
type MockClient struct {
	bucket   string
	region   string
	objects  map[string]*MockObject
	mu       sync.RWMutex
}

// MockObject represents a mock S3 object
type MockObject struct {
	Key        string
	Data       []byte
	Metadata   map[string]string
	Size       int64
	LastModified time.Time
}

// NewMockClient creates a new mock S3 client
func NewMockClient(bucket, region string) *MockClient {
	return &MockClient{
		bucket:  bucket,
		region:  region,
		objects: make(map[string]*MockObject),
	}
}

// ListObjects lists objects with the given prefix
func (m *MockClient) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var keys []string
	for key := range m.objects {
		if prefix == "" || (len(key) >= len(prefix) && key[:len(prefix)] == prefix) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

// GetObject retrieves an object
func (m *MockClient) GetObject(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	obj, exists := m.objects[key]
	if !exists {
		return nil, fmt.Errorf("object not found: %s", key)
	}
	
	// Return a copy of the data
	data := make([]byte, len(obj.Data))
	copy(data, obj.Data)
	return data, nil
}

// PutObject uploads an object
func (m *MockClient) PutObject(ctx context.Context, key string, data []byte) error {
	return m.PutObjectWithMetadata(ctx, key, data, nil)
}

// PutObjectWithMetadata uploads an object with metadata
func (m *MockClient) PutObjectWithMetadata(ctx context.Context, key string, data []byte, metadata map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Copy data
	objData := make([]byte, len(data))
	copy(objData, data)
	
	// Copy metadata
	objMetadata := make(map[string]string)
	if metadata != nil {
		for k, v := range metadata {
			objMetadata[k] = v
		}
	}
	
	m.objects[key] = &MockObject{
		Key:          key,
		Data:         objData,
		Metadata:     objMetadata,
		Size:         int64(len(data)),
		LastModified: time.Now(),
	}
	return nil
}

// DeleteObject deletes an object
func (m *MockClient) DeleteObject(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.objects, key)
	return nil
}

// HeadObject retrieves object metadata
func (m *MockClient) HeadObject(ctx context.Context, key string) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	obj, exists := m.objects[key]
	if !exists {
		return nil, fmt.Errorf("object not found: %s", key)
	}
	
	// Return a copy of metadata
	metadata := make(map[string]string)
	for k, v := range obj.Metadata {
		metadata[k] = v
	}
	return metadata, nil
}

// CopyObject copies an object (not used by filesystem, but for completeness)
func (m *MockClient) CopyObject(ctx context.Context, sourceKey, destKey string) error {
	return m.CopyObjectWithMetadata(ctx, sourceKey, destKey, nil)
}

// HeadObjectSize retrieves object size from metadata
func (m *MockClient) HeadObjectSize(ctx context.Context, key string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	obj, exists := m.objects[key]
	if !exists {
		return 0, fmt.Errorf("object not found: %s", key)
	}
	return obj.Size, nil
}

// CopyObjectWithMetadata copies an object with metadata
func (m *MockClient) CopyObjectWithMetadata(ctx context.Context, sourceKey, destKey string, metadata map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	sourceObj, exists := m.objects[sourceKey]
	if !exists {
		return fmt.Errorf("source object not found: %s", sourceKey)
	}
	
	// Copy data
	destData := make([]byte, len(sourceObj.Data))
	copy(destData, sourceObj.Data)
	
	// Replace metadata (not merge) - matching S3 behavior with MetadataDirectiveReplace
	destMetadata := make(map[string]string)
	if metadata != nil {
		// Use provided metadata as-is (replaces all metadata)
		for k, v := range metadata {
			destMetadata[k] = v
		}
	} else {
		// If no metadata provided, copy existing metadata
		for k, v := range sourceObj.Metadata {
			destMetadata[k] = v
		}
	}
	
	m.objects[destKey] = &MockObject{
		Key:          destKey,
		Data:         destData,
		Metadata:     destMetadata,
		Size:         sourceObj.Size,
		LastModified: time.Now(),
	}
	return nil
}

// GetObjectRange retrieves a range of bytes from an object
func (m *MockClient) GetObjectRange(ctx context.Context, key string, start, end int64) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	obj, exists := m.objects[key]
	if !exists {
		return nil, fmt.Errorf("object not found: %s", key)
	}
	
	// If end is 0 and start is 0, read entire file (same as GetObject)
	if start == 0 && end == 0 {
		data := make([]byte, len(obj.Data))
		copy(data, obj.Data)
		return data, nil
	}
	
	if start < 0 || start >= int64(len(obj.Data)) {
		return nil, fmt.Errorf("invalid range start: %d", start)
	}
	if end < start {
		return nil, fmt.Errorf("invalid range: end (%d) < start (%d)", end, start)
	}
	if end >= int64(len(obj.Data)) {
		end = int64(len(obj.Data)) - 1
	}
	
	return obj.Data[start : end+1], nil
}

// CreateBucket creates a bucket (no-op for mock)
func (m *MockClient) CreateBucket(ctx context.Context) error {
	return nil
}

// PutObjectMultipart uploads a large object using multipart upload (simplified for mock)
func (m *MockClient) PutObjectMultipart(ctx context.Context, key string, data []byte) error {
	return m.PutObject(ctx, key, data)
}

// CopyObjectMultipart copies a large object using multipart copy (simplified for mock)
func (m *MockClient) CopyObjectMultipart(ctx context.Context, sourceKey, destKey string) error {
	return m.CopyObjectWithMetadata(ctx, sourceKey, destKey, nil)
}
