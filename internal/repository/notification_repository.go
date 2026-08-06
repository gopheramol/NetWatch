package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gopheramol/NetWatch/internal/database"
	"github.com/gopheramol/NetWatch/internal/models"
	"go.etcd.io/bbolt"
)

type boltNotificationRepository struct {
	bolt *bbolt.DB
}

// NewNotificationRepository builds a bbolt-backed NotificationRepository.
func NewNotificationRepository(db *database.DB) NotificationRepository {
	return &boltNotificationRepository{bolt: db.Bolt()}
}

func (r *boltNotificationRepository) Save(ctx context.Context, notification *models.Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key := database.TimeKey(notification.Timestamp)
	return database.Put(r.bolt, database.BucketNotifications, key, notification)
}

func (r *boltNotificationRepository) List(ctx context.Context, limit int) ([]models.Notification, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var results []models.Notification
	err := r.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketNotifications))
		c := b.Cursor()
		for k, v := c.Last(); k != nil && (limit <= 0 || len(results) < limit); k, v = c.Prev() {
			var n models.Notification
			if err := json.Unmarshal(v, &n); err != nil {
				return fmt.Errorf("unmarshalling notification: %w", err)
			}
			results = append(results, n)
		}
		return nil
	})
	return results, err
}
