package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/s3fs-fuse/s3fs-go/internal/credentials"
	"github.com/s3fs-fuse/s3fs-go/internal/s3client"
)

func main() {
	// LocalStack credentials (dummy values)
	creds := credentials.NewCredentials()
	creds.AccessKeyID = "test"
	creds.SecretAccessKey = "test"

	bucket := "test-bucket"
	region := "us-east-1"
	endpoint := "http://localhost:4566"

	fmt.Printf("Connecting to LocalStack at %s\n", endpoint)
	fmt.Printf("Bucket: %s, Region: %s\n", bucket, region)

	client := s3client.NewClientWithEndpoint(bucket, region, endpoint, creds)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create bucket first using AWS SDK directly
	fmt.Println("\nCreating bucket...")
	// We'll need to access the internal s3Client, but for now let's use a workaround
	// Create bucket via HTTP or use AWS SDK directly
	// For LocalStack, we can create bucket via PutObject or CreateBucket API
	
	// Test listing objects (bucket might not exist yet)
	fmt.Println("\nTesting ListObjects (bucket may not exist)...")
	objects, err := client.ListObjects(ctx, "")
	if err != nil {
		log.Printf("ListObjects error (expected if bucket doesn't exist): %v", err)
		fmt.Println("Will create bucket by uploading first object...")
	} else {
		fmt.Printf("Found %d objects\n", len(objects))
		for _, obj := range objects {
			fmt.Printf("  - %s\n", obj)
		}
	}

	// Test creating a file
	fmt.Println("\nTesting PutObject...")
	testData := []byte("Hello from s3fs-go!")
	err = client.PutObject(ctx, "test.txt", testData)
	if err != nil {
		log.Fatalf("PutObject failed: %v", err)
	}
	fmt.Println("✓ File uploaded successfully")

	// Test reading the file back
	fmt.Println("\nTesting GetObject...")
	data, err := client.GetObject(ctx, "test.txt")
	if err != nil {
		log.Fatalf("GetObject failed: %v", err)
	}
	fmt.Printf("✓ File read successfully: %s\n", string(data))

	// Test listing again
	fmt.Println("\nTesting ListObjects again...")
	objects, err = client.ListObjects(ctx, "")
	if err != nil {
		log.Fatalf("ListObjects failed: %v", err)
	}
	fmt.Printf("Found %d objects:\n", len(objects))
	for _, obj := range objects {
		fmt.Printf("  - %s\n", obj)
	}

	fmt.Println("\n✓ All tests passed!")
}
