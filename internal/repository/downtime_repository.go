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

// ongoingOutageKey is stored in the settings bucket (not the downtime bucket)
// so its non-numeric key never interferes with the time-sortable outage keys.
const ongoingOutageKey = "ongoing_outage_id"

type boltDowntimeRepository struct {
	bolt *bbolt.DB
}

// NewDowntimeRepository builds a bbolt-backed DowntimeRepository.
func NewDowntimeRepository(db *database.DB) DowntimeRepository {
	return &boltDowntimeRepository{bolt: db.Bolt()}
}

func (r *boltDowntimeRepository) Create(ctx context.Context, outage *models.Outage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key := database.TimeKey(outage.StartTime)
	outage.ID = key

	return r.bolt.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketDowntime))
		data, err := json.Marshal(outage)
		if err != nil {
			return fmt.Errorf("marshalling outage: %w", err)
		}
		if err := b.Put([]byte(key), data); err != nil {
			return err
		}
		settings := tx.Bucket([]byte(database.BucketSettings))
		return settings.Put([]byte(ongoingOutageKey), []byte(key))
	})
}

func (r *boltDowntimeRepository) Update(ctx context.Context, outage *models.Outage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.bolt.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketDowntime))
		data, err := json.Marshal(outage)
		if err != nil {
			return fmt.Errorf("marshalling outage: %w", err)
		}
		if err := b.Put([]byte(outage.ID), data); err != nil {
			return err
		}
		if outage.Resolved {
			settings := tx.Bucket([]byte(database.BucketSettings))
			current := settings.Get([]byte(ongoingOutageKey))
			if string(current) == outage.ID {
				return settings.Delete([]byte(ongoingOutageKey))
			}
		}
		return nil
	})
}

func (r *boltDowntimeRepository) GetOngoing(ctx context.Context) (*models.Outage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var result *models.Outage
	err := r.bolt.View(func(tx *bbolt.Tx) error {
		settings := tx.Bucket([]byte(database.BucketSettings))
		id := settings.Get([]byte(ongoingOutageKey))
		if id == nil {
			return database.ErrNotFound
		}
		b := tx.Bucket([]byte(database.BucketDowntime))
		data := b.Get(id)
		if data == nil {
			return database.ErrNotFound
		}
		var outage models.Outage
		if err := json.Unmarshal(data, &outage); err != nil {
			return fmt.Errorf("unmarshalling outage: %w", err)
		}
		result = &outage
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *boltDowntimeRepository) List(ctx context.Context, limit int) ([]models.Outage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var results []models.Outage
	err := r.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketDowntime))
		c := b.Cursor()
		for k, v := c.Last(); k != nil && (limit <= 0 || len(results) < limit); k, v = c.Prev() {
			var outage models.Outage
			if err := json.Unmarshal(v, &outage); err != nil {
				return fmt.Errorf("unmarshalling outage: %w", err)
			}
			results = append(results, outage)
		}
		return nil
	})
	return results, err
}

func (r *boltDowntimeRepository) Range(ctx context.Context, from, to time.Time) ([]models.Outage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var results []models.Outage
	fromKey := []byte(database.TimeKey(from))
	toKey := []byte(database.TimeKey(to))

	err := r.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketDowntime))
		c := b.Cursor()
		for k, v := c.Seek(fromKey); k != nil; k, v = c.Next() {
			if string(k) > string(toKey) {
				break
			}
			var outage models.Outage
			if err := json.Unmarshal(v, &outage); err != nil {
				return fmt.Errorf("unmarshalling outage: %w", err)
			}
			results = append(results, outage)
		}
		return nil
	})
	return results, err
}
