package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/s3fs-fuse/s3fs-go/internal/storage/types"
)

// PostgresBackend implements storage.Backend using PostgreSQL
type PostgresBackend struct {
	db     *sql.DB
	table  string // Table name for storing files
	bucket string // "Bucket" name (namespace)
}

// NewPostgresBackend creates a new PostgreSQL backend
func NewPostgresBackend(connStr, table, bucket string) (*PostgresBackend, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	backend := &PostgresBackend{
		db:     db,
		table:  table,
		bucket: bucket,
	}

	// Initialize schema
	if err := backend.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return backend, nil
}

// initSchema creates the necessary tables
func (p *PostgresBackend) initSchema() error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			path VARCHAR(4096) PRIMARY KEY,
			bucket VARCHAR(255) NOT NULL,
			data BYTEA,
			size BIGINT NOT NULL DEFAULT 0,
			mode INTEGER NOT NULL DEFAULT 420,
			uid INTEGER NOT NULL DEFAULT 0,
			gid INTEGER NOT NULL DEFAULT 0,
			mtime TIMESTAMP NOT NULL DEFAULT NOW(),
			ctime TIMESTAMP NOT NULL DEFAULT NOW(),
			metadata JSONB,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_%s_bucket ON %s(bucket);
		CREATE INDEX IF NOT EXISTS idx_%s_prefix ON %s(path text_pattern_ops);
	`, p.table, p.table, p.table, p.table, p.table)

	_, err := p.db.Exec(query)
	return err
}

// Read reads file data
func (p *PostgresBackend) Read(ctx context.Context, path string) ([]byte, error) {
	query := fmt.Sprintf("SELECT data FROM %s WHERE path = $1 AND bucket = $2", p.table)
	var data []byte
	err := p.db.QueryRowContext(ctx, query, path, p.bucket).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not found: %w", os.ErrNotExist)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return data, nil
}

// ReadRange reads a range of file data
func (p *PostgresBackend) ReadRange(ctx context.Context, path string, start, end int64) ([]byte, error) {
	data, err := p.Read(ctx, path)
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
func (p *PostgresBackend) Write(ctx context.Context, path string, data []byte) error {
	return p.WriteWithMetadata(ctx, path, data, nil)
}

// WriteWithMetadata writes file data with metadata
func (p *PostgresBackend) WriteWithMetadata(ctx context.Context, path string, data []byte, metadata map[string]string) error {
	mode := 420 // 0644
	uid := uint32(os.Getuid())
	gid := uint32(os.Getgid())
	mtime := time.Now()
	ctime := mtime

	// Parse metadata if provided
	if metadata != nil {
		if modeStr, ok := metadata["mode"]; ok {
			var modeVal uint32
			fmt.Sscanf(modeStr, "%o", &modeVal)
			mode = int(modeVal)
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

	query := fmt.Sprintf(`
		INSERT INTO %s (path, bucket, data, size, mode, uid, gid, mtime, ctime, metadata, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (path) 
		DO UPDATE SET 
			data = EXCLUDED.data,
			size = EXCLUDED.size,
			mode = EXCLUDED.mode,
			uid = EXCLUDED.uid,
			gid = EXCLUDED.gid,
			mtime = EXCLUDED.mtime,
			ctime = EXCLUDED.ctime,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`, p.table)

	_, err := p.db.ExecContext(ctx, query, path, p.bucket, data, len(data), mode, uid, gid, mtime, ctime, metadata)
	return err
}

// Delete deletes a file
func (p *PostgresBackend) Delete(ctx context.Context, path string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE path = $1 AND bucket = $2", p.table)
	result, err := p.db.ExecContext(ctx, query, path, p.bucket)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("file not found: %w", os.ErrNotExist)
	}
	return nil
}

// List lists objects with the given prefix
func (p *PostgresBackend) List(ctx context.Context, prefix string) ([]string, error) {
	query := fmt.Sprintf("SELECT path FROM %s WHERE bucket = $1 AND path LIKE $2 ORDER BY path", p.table)
	rows, err := p.db.QueryContext(ctx, query, p.bucket, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, rows.Err()
}

// GetAttr gets file attributes
func (p *PostgresBackend) GetAttr(ctx context.Context, path string) (*types.Attr, error) {
	query := fmt.Sprintf("SELECT size, mode, uid, gid, mtime FROM %s WHERE path = $1 AND bucket = $2", p.table)
	var size int64
	var mode int
	var uid, gid uint32
	var mtime time.Time

	err := p.db.QueryRowContext(ctx, query, path, p.bucket).Scan(&size, &mode, &uid, &gid, &mtime)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not found: %w", os.ErrNotExist)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get attributes: %w", err)
	}

	return &types.Attr{
		Size:  size,
		Mode:  uint32(mode),
		Uid:   uid,
		Gid:   gid,
		Mtime: mtime,
	}, nil
}

// Rename renames a file or directory
func (p *PostgresBackend) Rename(ctx context.Context, oldPath, newPath string) error {
	query := fmt.Sprintf("UPDATE %s SET path = $1, updated_at = NOW() WHERE path = $2 AND bucket = $3", p.table)
	result, err := p.db.ExecContext(ctx, query, newPath, oldPath, p.bucket)
	if err != nil {
		return fmt.Errorf("failed to rename: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("file not found: %w", os.ErrNotExist)
	}
	return nil
}

// Exists checks if a file exists
// GetMetadata gets raw metadata map for a file
// TODO: Not implemented for PostgreSQL backend yet
// Extended attributes (xattrs) are not currently supported with PostgreSQL backend
// This would require storing metadata as JSON in a separate column or table
func (p *PostgresBackend) GetMetadata(ctx context.Context, path string) (map[string]string, error) {
	// Return empty metadata map for now
	// In the future, this could read from a metadata JSON column
	return make(map[string]string), nil
}

func (p *PostgresBackend) Exists(ctx context.Context, path string) (bool, error) {
	query := fmt.Sprintf("SELECT 1 FROM %s WHERE path = $1 AND bucket = $2 LIMIT 1", p.table)
	var exists int
	err := p.db.QueryRowContext(ctx, query, path, p.bucket).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Close closes the database connection
func (p *PostgresBackend) Close() error {
	return p.db.Close()
}
