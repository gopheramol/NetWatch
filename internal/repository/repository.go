// Package repository defines persistence interfaces for every domain
// concept and implements them on top of the embedded bbolt store. Services
// depend only on these interfaces, never on bbolt directly.
package repository

import (
	"context"
	"time"

	"github.com/gopheramol/NetWatch/internal/models"
)

// ConnectivityRepository persists raw connectivity check results.
type ConnectivityRepository interface {
	Save(ctx context.Context, check *models.ConnectivityCheck) error
	Latest(ctx context.Context) (*models.ConnectivityCheck, error)
	Range(ctx context.Context, from, to time.Time, limit int) ([]models.ConnectivityCheck, error)
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int, error)
}

// SpeedTestRepository persists speed test results.
type SpeedTestRepository interface {
	Save(ctx context.Context, result *models.SpeedTestResult) error
	Latest(ctx context.Context) (*models.SpeedTestResult, error)
	Range(ctx context.Context, from, to time.Time, limit int) ([]models.SpeedTestResult, error)
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int, error)
}

// DowntimeRepository persists outage/downtime records.
type DowntimeRepository interface {
	Create(ctx context.Context, outage *models.Outage) error
	Update(ctx context.Context, outage *models.Outage) error
	GetOngoing(ctx context.Context) (*models.Outage, error)
	List(ctx context.Context, limit int) ([]models.Outage, error)
	Range(ctx context.Context, from, to time.Time) ([]models.Outage, error)
}

// DailyStatsRepository persists continuously-updated per-day summaries.
type DailyStatsRepository interface {
	Get(ctx context.Context, date string) (*models.DailyStats, error)
	Save(ctx context.Context, stats *models.DailyStats) error
	Range(ctx context.Context, fromDate, toDate string) ([]models.DailyStats, error)
}

// MonthlyStatsRepository persists continuously-updated per-month summaries.
type MonthlyStatsRepository interface {
	Get(ctx context.Context, month string) (*models.MonthlyStats, error)
	Save(ctx context.Context, stats *models.MonthlyStats) error
	Range(ctx context.Context, fromMonth, toMonth string) ([]models.MonthlyStats, error)
}

// SettingsRepository persists the single application settings record.
type SettingsRepository interface {
	Get(ctx context.Context) (*models.Settings, error)
	Save(ctx context.Context, settings *models.Settings) error
}

// NotificationRepository persists a log of sent Telegram notifications.
type NotificationRepository interface {
	Save(ctx context.Context, notification *models.Notification) error
	List(ctx context.Context, limit int) ([]models.Notification, error)
}

// SystemMetricsRepository persists system metrics snapshots.
type SystemMetricsRepository interface {
	Save(ctx context.Context, metrics *models.SystemMetrics) error
	Latest(ctx context.Context) (*models.SystemMetrics, error)
	Range(ctx context.Context, from, to time.Time, limit int) ([]models.SystemMetrics, error)
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int, error)
}

