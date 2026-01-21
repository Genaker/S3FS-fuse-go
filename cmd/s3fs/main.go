package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/s3fs-fuse/s3fs-go/internal/credentials"
	"github.com/s3fs-fuse/s3fs-go/internal/fuse"
	"github.com/s3fs-fuse/s3fs-go/internal/s3client"
)

func main() {
	var (
		bucket        = flag.String("bucket", "", "S3 bucket name")
		mountpoint    = flag.String("mountpoint", "", "Mount point directory")
		region        = flag.String("region", "us-east-1", "AWS region")
		endpoint      = flag.String("endpoint", "", "S3 endpoint URL (for LocalStack or other S3-compatible services)")
		passwdFile    = flag.String("passwd_file", "", "Path to passwd file")
		enableFileLock = flag.Bool("enable_file_lock", false, "Enable file-level advisory locking for stricter coordination (default: false, uses entity-level locking)")
	)
	flag.Parse()

	if *bucket == "" {
		log.Fatal("bucket is required")
	}
	if *mountpoint == "" {
		log.Fatal("mountpoint is required")
	}

	// Load credentials
	creds := credentials.NewCredentials()
	
	if *passwdFile != "" {
		if err := creds.LoadFromPasswdFile(*passwdFile); err != nil {
			log.Fatalf("Failed to load credentials from file: %v", err)
		}
	} else {
		if err := creds.LoadFromEnvironment(); err != nil {
			log.Fatalf("Failed to load credentials from environment: %v", err)
		}
	}

	if !creds.IsValid() {
		log.Fatal("Invalid credentials")
	}

	// Create S3 client
	var client *s3client.Client
	if *endpoint != "" {
		client = s3client.NewClientWithEndpoint(*bucket, *region, *endpoint, creds)
		fmt.Printf("Using endpoint: %s\n", *endpoint)
	} else {
		client = s3client.NewClient(*bucket, *region, creds)
	}

	// Mount filesystem with options
	options := fuse.MountOptions{
		EnableFileLock: *enableFileLock,
	}
	fmt.Printf("Mounting bucket %s to %s\n", *bucket, *mountpoint)
	if *enableFileLock {
		fmt.Println("File-level advisory locking enabled")
	}
	if err := fuse.MountWithOptions(*mountpoint, client, options); err != nil {
		log.Fatalf("Failed to mount filesystem: %v", err)
	}
}
