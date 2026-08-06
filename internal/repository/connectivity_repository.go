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

type boltConnectivityRepository struct {
	bolt *bbolt.DB
}

// NewConnectivityRepository builds a bbolt-backed ConnectivityRepository.
func NewConnectivityRepository(db *database.DB) ConnectivityRepository {
	return &boltConnectivityRepository{bolt: db.Bolt()}
}

func (r *boltConnectivityRepository) Save(ctx context.Context, check *models.ConnectivityCheck) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key := database.TimeKey(check.Timestamp)
	return database.Put(r.bolt, database.BucketConnectivity, key, check)
}

func (r *boltConnectivityRepository) Latest(ctx context.Context) (*models.ConnectivityCheck, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var result *models.ConnectivityCheck
	err := r.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketConnectivity))
		k, v := b.Cursor().Last()
		if k == nil {
			return database.ErrNotFound
		}
		var check models.ConnectivityCheck
		if err := json.Unmarshal(v, &check); err != nil {
			return fmt.Errorf("unmarshalling connectivity check: %w", err)
		}
		result = &check
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *boltConnectivityRepository) Range(ctx context.Context, from, to time.Time, limit int) ([]models.ConnectivityCheck, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var results []models.ConnectivityCheck
	fromKey := []byte(database.TimeKey(from))
	toKey := []byte(database.TimeKey(to))

	err := r.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketConnectivity))
		c := b.Cursor()
		for k, v := c.Seek(fromKey); k != nil && (limit <= 0 || len(results) < limit); k, v = c.Next() {
			if string(k) > string(toKey) {
				break
			}
			var check models.ConnectivityCheck
			if err := json.Unmarshal(v, &check); err != nil {
				return fmt.Errorf("unmarshalling connectivity check: %w", err)
			}
			results = append(results, check)
		}
		return nil
	})
	return results, err
}

func (r *boltConnectivityRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	cutoffKey := []byte(database.TimeKey(cutoff))
	deleted := 0

	err := r.bolt.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketConnectivity))
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
