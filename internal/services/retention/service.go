// Package retention deletes raw time-series data past its retention window
// while leaving daily/monthly summaries untouched, so bbolt's file size
// stays bounded on a long-running home server.
package retention

import (
	"context"
	"time"

	"github.com/gopheramol/NetWatch/internal/repository"
	"go.uber.org/zap"
)

// Service enforces the configured raw-data retention window. The window
// itself is read from Settings on every run, so changing retention_days via
// POST /api/settings takes effect on the next cleanup cycle without a restart.
type Service struct {
	connRepo       repository.ConnectivityRepository
	sysRepo        repository.SystemMetricsRepository
	settingsRepo   repository.SettingsRepository
	defaultRawDays int
	logger         *zap.Logger
}

// NewService builds the retention Service. defaultRawDays is used only if
// settings have not yet been persisted.
func NewService(connRepo repository.ConnectivityRepository, sysRepo repository.SystemMetricsRepository, settingsRepo repository.SettingsRepository, defaultRawDays int, logger *zap.Logger) *Service {
	return &Service{connRepo: connRepo, sysRepo: sysRepo, settingsRepo: settingsRepo, defaultRawDays: defaultRawDays, logger: logger}
}

// Run deletes raw connectivity checks and system metrics older than the retention window.
// Daily and monthly summary buckets are never touched here — they are kept
// forever by design.
func (s *Service) Run(ctx context.Context) error {
	rawDataDays := s.defaultRawDays
	if settings, err := s.settingsRepo.Get(ctx); err == nil && settings.RetentionDays > 0 {
		rawDataDays = settings.RetentionDays
	}
	cutoff := time.Now().AddDate(0, 0, -rawDataDays)

	deleted, err := s.connRepo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		s.logger.Error("retention: failed to delete old connectivity checks", zap.Error(err))
		return err
	}

	sysDeleted, sysErr := s.sysRepo.DeleteOlderThan(ctx, cutoff)
	if sysErr != nil {
		s.logger.Error("retention: failed to delete old system metrics", zap.Error(sysErr))
	}

	s.logger.Info("retention: cleanup complete",
		zap.Int("deleted_connectivity_checks", deleted),
		zap.Int("deleted_sys_metrics", sysDeleted),
		zap.Time("cutoff", cutoff),
	)
	return nil
}

