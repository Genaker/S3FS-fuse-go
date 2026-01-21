//go:build integration

package tests

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/s3fs-fuse/s3fs-go/internal/credentials"
	"github.com/s3fs-fuse/s3fs-go/internal/s3client"
	"github.com/s3fs-fuse/s3fs-go/internal/fuse"
)

const (
	LocalStackEndpoint = "http://localhost:4566"
	LocalStackBucket   = "test-bucket-localstack"
	LocalStackRegion   = "us-east-1"
)

// Provider represents the S3 provider type
type Provider string

const (
	ProviderLocalStack Provider = "localstack"
	ProviderS3         Provider = "s3"
	ProviderR2         Provider = "r2"
)

// GetProvider returns the S3 provider from environment variable
func GetProvider() Provider {
	provider := strings.ToLower(os.Getenv("S3_PROVIDER"))
	switch provider {
	case "s3", "aws":
		return ProviderS3
	case "r2", "cloudflare":
		return ProviderR2
	default:
		return ProviderLocalStack // Default to LocalStack
	}
}

// IsLocalStackAvailable checks if LocalStack is running
func IsLocalStackAvailable() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(LocalStackEndpoint + "/_localstack/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// RequireLocalStack checks if LocalStack is available and fails the test if not
func RequireLocalStack(t *testing.T) {
	if !IsLocalStackAvailable() {
		t.Fatalf("LocalStack is not available. Start it with: docker-compose -f docker-compose.localstack.yml up -d")
	}
}

// SetupTestClient sets up an S3 client based on provider
func SetupTestClient(t *testing.T, bucket, region string) *s3client.Client {
	provider := GetProvider()

	switch provider {
	case ProviderLocalStack:
		RequireLocalStack(t)
		creds := credentials.NewCredentials()
		creds.AccessKeyID = "test"
		creds.SecretAccessKey = "test"
		client := s3client.NewClientWithEndpoint(bucket, region, LocalStackEndpoint, creds)
		
		// Create bucket if it doesn't exist
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		_, err := client.ListObjects(ctx, "")
		if err != nil {
			err = client.CreateBucket(ctx)
			if err != nil {
				if !strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") &&
					!strings.Contains(err.Error(), "BucketAlreadyExists") {
					t.Fatalf("Failed to create bucket: %v", err)
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
		return client

	case ProviderS3:
		// Use AWS S3
		creds := credentials.NewCredentials()
		if err := creds.LoadFromEnvironment(); err != nil {
			t.Fatalf("Failed to load AWS credentials: %v", err)
		}
		if !creds.IsValid() {
			t.Fatal("Invalid AWS credentials. Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY")
		}
		return s3client.NewClient(bucket, region, creds)

	case ProviderR2:
		// Use Cloudflare R2
		endpoint := os.Getenv("R2_ENDPOINT")
		if endpoint == "" {
			t.Fatal("R2_ENDPOINT environment variable is required for R2 provider")
		}
		creds := credentials.NewCredentials()
		if err := creds.LoadFromEnvironment(); err != nil {
			t.Fatalf("Failed to load R2 credentials: %v", err)
		}
		if !creds.IsValid() {
			t.Fatal("Invalid R2 credentials. Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY")
		}
		return s3client.NewClientWithEndpoint(bucket, region, endpoint, creds)

	default:
		t.Fatalf("Unknown provider: %s", provider)
		return nil
	}
}

// SetupTestFilesystem sets up a filesystem for testing
func SetupTestFilesystem(t *testing.T, bucket, region string) *fuse.Filesystem {
	client := SetupTestClient(t, bucket, region)
	return fuse.NewFilesystem(client)
}

// SetupLocalStackTestClient sets up a LocalStack client (for backward compatibility)
func SetupLocalStackTestClient(t *testing.T) *s3client.Client {
	return SetupTestClient(t, LocalStackBucket, LocalStackRegion)
}

// SetupLocalStackTestFilesystem sets up a LocalStack filesystem (for backward compatibility)
func SetupLocalStackTestFilesystem(t *testing.T) *fuse.Filesystem {
	return SetupTestFilesystem(t, LocalStackBucket, LocalStackRegion)
}
