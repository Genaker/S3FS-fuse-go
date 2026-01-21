package storage

import (
	"fmt"

	"github.com/s3fs-fuse/s3fs-go/internal/storage/mongodb"
	"github.com/s3fs-fuse/s3fs-go/internal/storage/postgres"
	"github.com/s3fs-fuse/s3fs-go/internal/storage/types"
)

// BackendType represents the type of storage backend
type BackendType string

const (
	BackendTypeS3       BackendType = "s3"
	BackendTypePostgres BackendType = "postgres"
	BackendTypeMongoDB  BackendType = "mongodb"
)

// Config holds configuration for creating a backend
type Config struct {
	Type     BackendType
	S3Backend types.Backend // For S3 backend (pre-created adapter)
	
	// Postgres config
	PostgresConnStr string
	PostgresTable   string
	PostgresBucket  string
	
	// MongoDB config
	MongoURI        string
	MongoDatabase   string
	MongoCollection string
	MongoBucket     string
}

// NewBackend creates a new storage backend based on the config
func NewBackend(config Config) (types.Backend, error) {
	switch config.Type {
	case BackendTypeS3:
		if config.S3Backend == nil {
			return nil, fmt.Errorf("S3 backend is required for S3 backend type")
		}
		return config.S3Backend, nil
		
	case BackendTypePostgres:
		if config.PostgresConnStr == "" {
			return nil, fmt.Errorf("PostgreSQL connection string is required")
		}
		table := config.PostgresTable
		if table == "" {
			table = "files"
		}
		bucket := config.PostgresBucket
		if bucket == "" {
			bucket = "default"
		}
		return postgres.NewPostgresBackend(config.PostgresConnStr, table, bucket)
		
	case BackendTypeMongoDB:
		if config.MongoURI == "" {
			return nil, fmt.Errorf("MongoDB URI is required")
		}
		database := config.MongoDatabase
		if database == "" {
			database = "s3fs"
		}
		collection := config.MongoCollection
		if collection == "" {
			collection = "files"
		}
		bucket := config.MongoBucket
		if bucket == "" {
			bucket = "default"
		}
		return mongodb.NewMongoBackend(config.MongoURI, database, collection, bucket)
		
	default:
		return nil, fmt.Errorf("unknown backend type: %s", config.Type)
	}
}
