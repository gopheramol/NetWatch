package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gopheramol/NetWatch/internal/database"
	"github.com/gopheramol/NetWatch/internal/models"
	"go.etcd.io/bbolt"
)

type boltDailyStatsRepository struct {
	bolt *bbolt.DB
}

// NewDailyStatsRepository builds a bbolt-backed DailyStatsRepository.
func NewDailyStatsRepository(db *database.DB) DailyStatsRepository {
	return &boltDailyStatsRepository{bolt: db.Bolt()}
}

func (r *boltDailyStatsRepository) Get(ctx context.Context, date string) (*models.DailyStats, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var stats models.DailyStats
	if err := database.Get(r.bolt, database.BucketDailyStats, date, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *boltDailyStatsRepository) Save(ctx context.Context, stats *models.DailyStats) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return database.Put(r.bolt, database.BucketDailyStats, stats.Date, stats)
}

func (r *boltDailyStatsRepository) Range(ctx context.Context, fromDate, toDate string) ([]models.DailyStats, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var results []models.DailyStats
	err := r.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketDailyStats))
		c := b.Cursor()
		for k, v := c.Seek([]byte(fromDate)); k != nil && string(k) <= toDate; k, v = c.Next() {
			var s models.DailyStats
			if err := json.Unmarshal(v, &s); err != nil {
				return fmt.Errorf("unmarshalling daily stats: %w", err)
			}
			results = append(results, s)
		}
		return nil
	})
	return results, err
}

type boltMonthlyStatsRepository struct {
	bolt *bbolt.DB
}

// NewMonthlyStatsRepository builds a bbolt-backed MonthlyStatsRepository.
func NewMonthlyStatsRepository(db *database.DB) MonthlyStatsRepository {
	return &boltMonthlyStatsRepository{bolt: db.Bolt()}
}

func (r *boltMonthlyStatsRepository) Get(ctx context.Context, month string) (*models.MonthlyStats, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var stats models.MonthlyStats
	if err := database.Get(r.bolt, database.BucketMonthlyStats, month, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *boltMonthlyStatsRepository) Save(ctx context.Context, stats *models.MonthlyStats) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return database.Put(r.bolt, database.BucketMonthlyStats, stats.Month, stats)
}

func (r *boltMonthlyStatsRepository) Range(ctx context.Context, fromMonth, toMonth string) ([]models.MonthlyStats, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var results []models.MonthlyStats
	err := r.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketMonthlyStats))
		c := b.Cursor()
		for k, v := c.Seek([]byte(fromMonth)); k != nil && string(k) <= toMonth; k, v = c.Next() {
			var s models.MonthlyStats
			if err := json.Unmarshal(v, &s); err != nil {
				return fmt.Errorf("unmarshalling monthly stats: %w", err)
			}
			results = append(results, s)
		}
		return nil
	})
	return results, err
}
