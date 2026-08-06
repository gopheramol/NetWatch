package speedtest

import (
	"context"
	"fmt"
	"time"

	"github.com/gopheramol/NetWatch/internal/analytics"
	"github.com/gopheramol/NetWatch/internal/models"
	"github.com/gopheramol/NetWatch/internal/repository"
	"github.com/gopheramol/NetWatch/internal/telegram"
	"github.com/gopheramol/NetWatch/internal/utils"
	"go.uber.org/zap"
)

// Service runs speed tests (scheduled or manually triggered), persists the
// result, and folds it into the analytics engine.
type Service struct {
	provider        Provider
	repo            repository.SpeedTestRepository
	analytics       analytics.Engine
	notifier        telegram.Notifier
	minDownloadMbps float64
	minUploadMbps   float64
	logger          *zap.Logger
}

// NewService builds the speed test Service around the given Provider.
// minDownloadMbps/minUploadMbps of 0 disable the slow-speed alert.
func NewService(
	provider Provider,
	repo repository.SpeedTestRepository,
	engine analytics.Engine,
	notifier telegram.Notifier,
	minDownloadMbps, minUploadMbps float64,
	logger *zap.Logger,
) *Service {
	return &Service{
		provider:        provider,
		repo:            repo,
		analytics:       engine,
		notifier:        notifier,
		minDownloadMbps: minDownloadMbps,
		minUploadMbps:   minUploadMbps,
		logger:          logger,
	}
}

// Run executes a speed test now, persists it, and updates analytics. It is
// safe to call both from the scheduler and from the manual trigger API.
func (s *Service) Run(ctx context.Context) (*models.SpeedTestResult, error) {
	s.logger.Info("speedtest: starting test")
	start := time.Now()

	result, err := s.provider.RunTest(ctx)
	if err != nil {
		s.logger.Error("speedtest: test failed", zap.Error(err))
		return nil, fmt.Errorf("running speed test: %w", err)
	}

	result.ID = utils.NewID()
	result.Timestamp = time.Now()

	if err := s.repo.Save(ctx, result); err != nil {
		return nil, fmt.Errorf("saving speed test result: %w", err)
	}
	if err := s.analytics.RecordSpeedTest(ctx, *result); err != nil {
		s.logger.Error("speedtest: failed to record analytics", zap.Error(err))
	}

	s.logger.Info("speedtest: completed",
		zap.Float64("download_mbps", result.DownloadMbps),
		zap.Float64("upload_mbps", result.UploadMbps),
		zap.Float64("ping_ms", result.PingMs),
		zap.Duration("elapsed", time.Since(start)),
	)

	s.checkSlowSpeed(ctx, *result)

	return result, nil
}

// checkSlowSpeed fires a one-shot alert when a completed test falls below
// the configured minimums. There's no state machine here (unlike outages or
// high latency) since speed tests are infrequent enough that each low result
// is worth its own alert.
func (s *Service) checkSlowSpeed(ctx context.Context, result models.SpeedTestResult) {
	belowDownload := s.minDownloadMbps > 0 && result.DownloadMbps < s.minDownloadMbps
	belowUpload := s.minUploadMbps > 0 && result.UploadMbps < s.minUploadMbps
	if !belowDownload && !belowUpload {
		return
	}

	s.logger.Warn("speedtest: result below configured minimum",
		zap.Float64("download_mbps", result.DownloadMbps),
		zap.Float64("upload_mbps", result.UploadMbps),
	)
	if err := s.notifier.NotifySlowSpeed(ctx, result, s.minDownloadMbps, s.minUploadMbps); err != nil {
		s.logger.Error("speedtest: failed to send slow speed notification", zap.Error(err))
	}
}
