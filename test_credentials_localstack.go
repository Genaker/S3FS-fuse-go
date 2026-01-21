package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/s3fs-fuse/s3fs-go/internal/credentials"
	"github.com/s3fs-fuse/s3fs-go/internal/s3client"
)

func main() {
	fmt.Println("=== Testing LocalStack Credentials ===")
	fmt.Println()

	// Check LocalStack health
	fmt.Println("1. Checking LocalStack health...")
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:4566/_localstack/health")
	if err != nil {
		fmt.Printf("   ❌ LocalStack is not available: %v\n", err)
		fmt.Println("\n   Start LocalStack with:")
		fmt.Println("   docker-compose -f docker-compose.localstack.yml up -d")
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		fmt.Println("   ✅ LocalStack is running and healthy")
	} else {
		fmt.Printf("   ⚠️  LocalStack returned status: %d\n", resp.StatusCode)
	}

	// Test credentials setup
	fmt.Println("\n2. Testing credentials setup...")
	creds := credentials.NewCredentials()
	creds.AccessKeyID = "test"
	creds.SecretAccessKey = "test"
	
	if !creds.IsValid() {
		fmt.Println("   ❌ Credentials are invalid")
		os.Exit(1)
	}
	fmt.Println("   ✅ LocalStack credentials are valid (test/test)")

	// Test S3 client creation
	fmt.Println("\n3. Testing S3 client creation...")
	s3Client := s3client.NewClientWithEndpoint(
		"test-bucket-localstack",
		"us-east-1",
		"http://localhost:4566",
		creds,
	)
	if s3Client == nil {
		fmt.Println("   ❌ Failed to create S3 client")
		os.Exit(1)
	}
	fmt.Println("   ✅ S3 client created successfully")

	// Test bucket operations
	fmt.Println("\n4. Testing bucket operations...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to list objects (will create bucket if needed)
	_, err = s3Client.ListObjects(ctx, "")
	if err != nil {
		// Try to create bucket
		err = s3Client.CreateBucket(ctx)
		if err != nil {
			fmt.Printf("   ⚠️  Bucket operation failed: %v\n", err)
			fmt.Println("   (This is OK if bucket already exists)")
		} else {
			fmt.Println("   ✅ Bucket created successfully")
			time.Sleep(500 * time.Millisecond)
		}
	} else {
		fmt.Println("   ✅ Bucket exists and is accessible")
	}

	// Test PUT operation
	fmt.Println("\n5. Testing PUT operation...")
	testKey := fmt.Sprintf("test-credentials-%d", time.Now().UnixNano())
	testData := []byte("Hello from credentials test!")
	err = s3Client.PutObject(ctx, testKey, testData)
	if err != nil {
		fmt.Printf("   ❌ PUT operation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   ✅ PUT operation successful")

	// Test GET operation
	fmt.Println("\n6. Testing GET operation...")
	data, err := s3Client.GetObject(ctx, testKey)
	if err != nil {
		fmt.Printf("   ❌ GET operation failed: %v\n", err)
		os.Exit(1)
	}
	if string(data) != string(testData) {
		fmt.Printf("   ❌ Data mismatch: expected '%s', got '%s'\n", string(testData), string(data))
		os.Exit(1)
	}
	fmt.Println("   ✅ GET operation successful")

	// Cleanup
	fmt.Println("\n7. Cleaning up test data...")
	err = s3Client.DeleteObject(ctx, testKey)
	if err != nil {
		fmt.Printf("   ⚠️  Cleanup failed: %v\n", err)
	} else {
		fmt.Println("   ✅ Test data cleaned up")
	}

	fmt.Println("\n=== All Credential Tests Passed! ===")
	fmt.Println("\nLocalStack credentials are working correctly:")
	fmt.Println("  - Access Key ID: test")
	fmt.Println("  - Secret Access Key: test")
	fmt.Println("  - Endpoint: http://localhost:4566")
	fmt.Println("  - Bucket: test-bucket-localstack")
	fmt.Println("  - Region: us-east-1")
}
