package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gopheramol/NetWatch/internal/database"
	"github.com/gopheramol/NetWatch/internal/models"
	"go.etcd.io/bbolt"
)

type boltSystemMetricsRepository struct {
	bolt *bbolt.DB
}

// NewSystemMetricsRepository builds a bbolt-backed SystemMetricsRepository.
func NewSystemMetricsRepository(db *database.DB) SystemMetricsRepository {
	return &boltSystemMetricsRepository{bolt: db.Bolt()}
}

func (r *boltSystemMetricsRepository) Save(ctx context.Context, metrics *models.SystemMetrics) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key := database.TimeKey(metrics.Timestamp)
	return database.Put(r.bolt, database.BucketSysMetrics, key, metrics)
}

func (r *boltSystemMetricsRepository) Latest(ctx context.Context) (*models.SystemMetrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var result *models.SystemMetrics
	err := r.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketSysMetrics))
		if b == nil {
			return database.ErrNotFound
		}
		k, v := b.Cursor().Last()
		if k == nil {
			return database.ErrNotFound
		}
		var metrics models.SystemMetrics
		if err := json.Unmarshal(v, &metrics); err != nil {
			return fmt.Errorf("unmarshalling sys metrics: %w", err)
		}
		result = &metrics
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *boltSystemMetricsRepository) Range(ctx context.Context, from, to time.Time, limit int) ([]models.SystemMetrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var results []models.SystemMetrics
	fromKey := []byte(database.TimeKey(from))
	toKey := []byte(database.TimeKey(to))

	err := r.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketSysMetrics))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.Seek(fromKey); k != nil && (limit <= 0 || len(results) < limit); k, v = c.Next() {
			if string(k) > string(toKey) {
				break
			}
			var metrics models.SystemMetrics
			if err := json.Unmarshal(v, &metrics); err != nil {
				return fmt.Errorf("unmarshalling sys metrics: %w", err)
			}
			results = append(results, metrics)
		}
		return nil
	})
	return results, err
}

func (r *boltSystemMetricsRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	cutoffKey := []byte(database.TimeKey(cutoff))
	deleted := 0

	err := r.bolt.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketSysMetrics))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		var keysToDelete [][]byte
		for k, _ := c.First(); k != nil && string(k) < string(cutoffKey); k, _ = c.Next() {
			keysToDelete = append(keysToDelete, append([]byte{}, k...))
		}
		for _, k := range keysToDelete {
			if err := b.Delete(k); err != nil {
				return err
			}
			deleted++
		}
		return nil
	})
	return deleted, err
}
