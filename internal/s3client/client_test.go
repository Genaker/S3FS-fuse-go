package s3client

import (
	"context"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-bucket", "us-east-1", nil)
	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.bucket != "test-bucket" {
		t.Errorf("Expected bucket 'test-bucket', got '%s'", client.bucket)
	}

	if client.region != "us-east-1" {
		t.Errorf("Expected region 'us-east-1', got '%s'", client.region)
	}
}

func TestListObjects(t *testing.T) {
	// This is a test that will fail until we implement ListObjects
	// Following TDD: write test first, then implement
	client := NewClient("test-bucket", "us-east-1", nil)
	
	ctx := context.Background()
	objects, err := client.ListObjects(ctx, "prefix/")
	
	// For now, we expect this to fail or return empty
	// In real implementation, this would connect to S3
	_ = objects
	_ = err
}

func TestGetObject(t *testing.T) {
	client := NewClient("test-bucket", "us-east-1", nil)
	
	ctx := context.Background()
	data, err := client.GetObject(ctx, "test-key")
	
	// Test will fail until implemented
	_ = data
	_ = err
}

func TestPutObject(t *testing.T) {
	client := NewClient("test-bucket", "us-east-1", nil)
	
	ctx := context.Background()
	err := client.PutObject(ctx, "test-key", []byte("test data"))
	
	// Test will fail until implemented
	_ = err
}

func TestDeleteObject(t *testing.T) {
	client := NewClient("test-bucket", "us-east-1", nil)
	
	ctx := context.Background()
	err := client.DeleteObject(ctx, "test-key")
	
	// Test will fail until implemented
	_ = err
}
