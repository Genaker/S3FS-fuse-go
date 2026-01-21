package mongodb

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/s3fs-fuse/s3fs-go/internal/storage/types"
)

// FileDocument represents a file document in MongoDB
type FileDocument struct {
	Path     string                 `bson:"_id"`
	Bucket   string                 `bson:"bucket"`
	Data     []byte                 `bson:"data"`
	Size     int64                  `bson:"size"`
	Mode     uint32                 `bson:"mode"`
	Uid      uint32                 `bson:"uid"`
	Gid      uint32                 `bson:"gid"`
	Mtime    time.Time              `bson:"mtime"`
	Ctime    time.Time              `bson:"ctime"`
	Metadata map[string]interface{} `bson:"metadata,omitempty"`
	CreatedAt time.Time            `bson:"created_at"`
	UpdatedAt time.Time            `bson:"updated_at"`
}

// MongoBackend implements storage.Backend using MongoDB
type MongoBackend struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
	bucket     string
}

// NewMongoBackend creates a new MongoDB backend
func NewMongoBackend(uri, database, collection, bucket string) (*MongoBackend, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Verify connection
	if err := client.Ping(context.Background(), nil); err != nil {
		client.Disconnect(context.Background())
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(database)
	coll := db.Collection(collection)

	// Create indexes
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "bucket", Value: 1},
			{Key: "path", Value: 1},
		},
	}
	coll.Indexes().CreateOne(context.Background(), indexModel)

	return &MongoBackend{
		client:     client,
		db:         db,
		collection: coll,
		bucket:     bucket,
	}, nil
}

// Read reads file data
func (m *MongoBackend) Read(ctx context.Context, path string) ([]byte, error) {
	filter := bson.M{"_id": path, "bucket": m.bucket}
	var doc FileDocument
	err := m.collection.FindOne(ctx, filter).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, fmt.Errorf("file not found: %w", os.ErrNotExist)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return doc.Data, nil
}

// ReadRange reads a range of file data
func (m *MongoBackend) ReadRange(ctx context.Context, path string, start, end int64) ([]byte, error) {
	data, err := m.Read(ctx, path)
	if err != nil {
		return nil, err
	}
	
	if start < 0 {
		start = 0
	}
	if end < 0 || end > int64(len(data)) {
		end = int64(len(data))
	}
	if start > int64(len(data)) {
		return []byte{}, nil
	}
	
	return data[start:end], nil
}

// Write writes file data
func (m *MongoBackend) Write(ctx context.Context, path string, data []byte) error {
	return m.WriteWithMetadata(ctx, path, data, nil)
}

// WriteWithMetadata writes file data with metadata
func (m *MongoBackend) WriteWithMetadata(ctx context.Context, path string, data []byte, metadata map[string]string) error {
	mode := uint32(420) // 0644
	uid := uint32(os.Getuid())
	gid := uint32(os.Getgid())
	mtime := time.Now()
	ctime := mtime

	// Parse metadata if provided
	metaMap := make(map[string]interface{})
	if metadata != nil {
		for k, v := range metadata {
			metaMap[k] = v
		}
		
		if modeStr, ok := metadata["mode"]; ok {
			var modeVal uint32
			fmt.Sscanf(modeStr, "%o", &modeVal)
			mode = modeVal
		}
		if uidStr, ok := metadata["uid"]; ok {
			fmt.Sscanf(uidStr, "%d", &uid)
		}
		if gidStr, ok := metadata["gid"]; ok {
			fmt.Sscanf(gidStr, "%d", &gid)
		}
		if mtimeStr, ok := metadata["mtime"]; ok {
			var unixTime int64
			if _, err := fmt.Sscanf(mtimeStr, "%d", &unixTime); err == nil {
				mtime = time.Unix(unixTime, 0)
			}
		}
		if ctimeStr, ok := metadata["ctime"]; ok {
			var unixTime int64
			if _, err := fmt.Sscanf(ctimeStr, "%d", &unixTime); err == nil {
				ctime = time.Unix(unixTime, 0)
			}
		}
	}

	now := time.Now()
	doc := FileDocument{
		Path:      path,
		Bucket:    m.bucket,
		Data:      data,
		Size:      int64(len(data)),
		Mode:      mode,
		Uid:       uid,
		Gid:       gid,
		Mtime:     mtime,
		Ctime:     ctime,
		Metadata:  metaMap,
		UpdatedAt: now,
	}

	// Check if document exists
	filter := bson.M{"_id": path, "bucket": m.bucket}
	var existing FileDocument
	err := m.collection.FindOne(ctx, filter).Decode(&existing)
	if err == mongo.ErrNoDocuments {
		// New document
		doc.CreatedAt = now
		_, err = m.collection.InsertOne(ctx, doc)
	} else if err == nil {
		// Update existing
		update := bson.M{
			"$set": bson.M{
				"data":       doc.Data,
				"size":       doc.Size,
				"mode":       doc.Mode,
				"uid":        doc.Uid,
				"gid":        doc.Gid,
				"mtime":      doc.Mtime,
				"ctime":      doc.Ctime,
				"metadata":   doc.Metadata,
				"updated_at": doc.UpdatedAt,
			},
		}
		_, err = m.collection.UpdateOne(ctx, filter, update)
	} else {
		return fmt.Errorf("failed to check existing file: %w", err)
	}

	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

// Delete deletes a file
func (m *MongoBackend) Delete(ctx context.Context, path string) error {
	filter := bson.M{"_id": path, "bucket": m.bucket}
	result, err := m.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("file not found: %w", os.ErrNotExist)
	}
	return nil
}

// List lists objects with the given prefix
func (m *MongoBackend) List(ctx context.Context, prefix string) ([]string, error) {
	filter := bson.M{
		"bucket": m.bucket,
		"_id":    bson.M{"$regex": "^" + prefix},
	}
	
	cursor, err := m.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"_id": 1}))
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}
	defer cursor.Close(ctx)

	var paths []string
	for cursor.Next(ctx) {
		var doc FileDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		paths = append(paths, doc.Path)
	}
	return paths, cursor.Err()
}

// GetAttr gets file attributes
func (m *MongoBackend) GetAttr(ctx context.Context, path string) (*types.Attr, error) {
	filter := bson.M{"_id": path, "bucket": m.bucket}
	var doc FileDocument
	err := m.collection.FindOne(ctx, filter).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, fmt.Errorf("file not found: %w", os.ErrNotExist)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get attributes: %w", err)
	}

	return &types.Attr{
		Size:  doc.Size,
		Mode:  doc.Mode,
		Uid:   doc.Uid,
		Gid:   doc.Gid,
		Mtime: doc.Mtime,
	}, nil
}

// Rename renames a file or directory
func (m *MongoBackend) Rename(ctx context.Context, oldPath, newPath string) error {
	filter := bson.M{"_id": oldPath, "bucket": m.bucket}
	update := bson.M{
		"$set": bson.M{
			"_id":        newPath,
			"updated_at": time.Now(),
		},
	}
	result, err := m.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to rename: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("file not found: %w", os.ErrNotExist)
	}
	return nil
}

// Exists checks if a file exists
// GetMetadata gets raw metadata map for a file
// TODO: Not implemented for MongoDB backend yet
// Extended attributes (xattrs) are not currently supported with MongoDB backend
// This would require reading from the Metadata field in FileDocument and converting
// map[string]interface{} to map[string]string
func (m *MongoBackend) GetMetadata(ctx context.Context, path string) (map[string]string, error) {
	// Return empty metadata map for now
	// In the future, this could read from FileDocument.Metadata field
	return make(map[string]string), nil
}

func (m *MongoBackend) Exists(ctx context.Context, path string) (bool, error) {
	filter := bson.M{"_id": path, "bucket": m.bucket}
	count, err := m.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return count > 0, nil
}

// Close closes the MongoDB connection
func (m *MongoBackend) Close() error {
	return m.client.Disconnect(context.Background())
}
