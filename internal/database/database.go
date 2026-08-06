// Package database manages the embedded bbolt key-value store: opening the
// file, creating buckets, and providing generic JSON get/put/scan helpers
// used by the repository layer.
package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

// Bucket names, one per top-level domain concept.
const (
	BucketSettings      = "settings"
	BucketConnectivity  = "connectivity"
	BucketSpeedTests    = "speedtests"
	BucketDowntime      = "downtime"
	BucketDailyStats    = "daily_stats"
	BucketMonthlyStats  = "monthly_stats"
	BucketNotifications = "notifications"
	BucketSysMetrics    = "sys_metrics"
)

var allBuckets = []string{
	BucketSettings,
	BucketConnectivity,
	BucketSpeedTests,
	BucketDowntime,
	BucketDailyStats,
	BucketMonthlyStats,
	BucketNotifications,
	BucketSysMetrics,
}

// DB wraps a bbolt database handle.
type DB struct {
	bolt *bbolt.DB
}

// Open opens (creating if needed) the bbolt database at path and ensures all
// required buckets exist.
func Open(path string) (*DB, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating database directory: %w", err)
		}
	}

	bdb, err := bbolt.Open(path, 0o600, &bbolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("opening bbolt database: %w", err)
	}

	err = bdb.Update(func(tx *bbolt.Tx) error {
		for _, name := range allBuckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(name)); err != nil {
				return fmt.Errorf("creating bucket %q: %w", name, err)
			}
		}
		return nil
	})
	if err != nil {
		_ = bdb.Close()
		return nil, err
	}

	return &DB{bolt: bdb}, nil
}

// Close closes the underlying bbolt database.
func (d *DB) Close() error {
	return d.bolt.Close()
}

// Bolt exposes the raw bbolt handle for callers that need direct transaction access.
func (d *DB) Bolt() *bbolt.DB {
	return d.bolt
}

// Put JSON-marshals value and stores it under key in bucket.
func Put(bolt *bbolt.DB, bucket, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshalling value: %w", err)
	}
	return bolt.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %q not found", bucket)
		}
		return b.Put([]byte(key), data)
	})
}

// Get retrieves the value under key in bucket and unmarshals it into dest.
// Returns ErrNotFound if the key does not exist.
func Get(bolt *bbolt.DB, bucket, key string, dest interface{}) error {
	return bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %q not found", bucket)
		}
		data := b.Get([]byte(key))
		if data == nil {
			return ErrNotFound
		}
		return json.Unmarshal(data, dest)
	})
}

// Delete removes key from bucket.
func Delete(bolt *bbolt.DB, bucket, key string) error {
	return bolt.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %q not found", bucket)
		}
		return b.Delete([]byte(key))
	})
}

// ErrNotFound is returned by Get when the requested key does not exist.
var ErrNotFound = fmt.Errorf("key not found")

// TimeKey converts a timestamp into a fixed-width, lexically-sortable string
// suitable as a bbolt key, so that byte-order iteration equals chronological order.
func TimeKey(t time.Time) string {
	return fmt.Sprintf("%020d", t.UTC().UnixNano())
}
