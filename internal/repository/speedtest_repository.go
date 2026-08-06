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

type boltSpeedTestRepository struct {
	bolt *bbolt.DB
}

// NewSpeedTestRepository builds a bbolt-backed SpeedTestRepository.
func NewSpeedTestRepository(db *database.DB) SpeedTestRepository {
	return &boltSpeedTestRepository{bolt: db.Bolt()}
}

func (r *boltSpeedTestRepository) Save(ctx context.Context, result *models.SpeedTestResult) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key := database.TimeKey(result.Timestamp)
	return database.Put(r.bolt, database.BucketSpeedTests, key, result)
}

func (r *boltSpeedTestRepository) Latest(ctx context.Context) (*models.SpeedTestResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var result *models.SpeedTestResult
	err := r.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketSpeedTests))
		k, v := b.Cursor().Last()
		if k == nil {
			return database.ErrNotFound
		}
		var st models.SpeedTestResult
		if err := json.Unmarshal(v, &st); err != nil {
			return fmt.Errorf("unmarshalling speed test: %w", err)
		}
		result = &st
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *boltSpeedTestRepository) Range(ctx context.Context, from, to time.Time, limit int) ([]models.SpeedTestResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var results []models.SpeedTestResult
	fromKey := []byte(database.TimeKey(from))
	toKey := []byte(database.TimeKey(to))

	err := r.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketSpeedTests))
		c := b.Cursor()
		for k, v := c.Seek(fromKey); k != nil && (limit <= 0 || len(results) < limit); k, v = c.Next() {
			if string(k) > string(toKey) {
				break
			}
			var st models.SpeedTestResult
			if err := json.Unmarshal(v, &st); err != nil {
				return fmt.Errorf("unmarshalling speed test: %w", err)
			}
			results = append(results, st)
		}
		return nil
	})
	return results, err
}

func (r *boltSpeedTestRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	cutoffKey := []byte(database.TimeKey(cutoff))
	deleted := 0

	err := r.bolt.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketSpeedTests))
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
