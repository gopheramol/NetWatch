package battery

import (
	"context"
	"sync"

	"github.com/gopheramol/NetWatch/internal/repository"
	"github.com/gopheramol/NetWatch/internal/telegram"
	"go.uber.org/zap"
)

// Service periodically checks battery level and alerts on the low/restored
// transition (not on every check), the same pattern used for outages.
type Service struct {
	reader        Reader
	settingsRepo  repository.SettingsRepository
	notifier      telegram.Notifier
	logger        *zap.Logger
	defaultLowPct float64

	mu    sync.Mutex
	isLow bool
}

// NewService builds the battery Service. defaultLowPct is used until
// Settings.BatteryLowThresholdPct is persisted.
func NewService(reader Reader, settingsRepo repository.SettingsRepository, notifier telegram.Notifier, defaultLowPct float64, logger *zap.Logger) *Service {
	return &Service{
		reader:        reader,
		settingsRepo:  settingsRepo,
		notifier:      notifier,
		defaultLowPct: defaultLowPct,
		logger:        logger,
	}
}

// Run takes one reading and, if a battery is present, alerts on a
// low/restored transition.
func (s *Service) Run(ctx context.Context) error {
	reading, err := s.reader.Read(ctx)
	if err != nil {
		s.logger.Error("battery: read failed", zap.Error(err))
		return err
	}
	if !reading.Present {
		return nil
	}

	threshold := s.defaultLowPct
	if settings, err := s.settingsRepo.Get(ctx); err == nil && settings.BatteryLowThresholdPct > 0 {
		threshold = settings.BatteryLowThresholdPct
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	switch {
	case reading.Percent <= threshold && !s.isLow:
		s.isLow = true
		s.logger.Warn("battery: low", zap.Float64("percent", reading.Percent), zap.Float64("threshold", threshold))
		if err := s.notifier.NotifyBatteryLow(ctx, reading.Percent, threshold); err != nil {
			s.logger.Error("battery: failed to send low notification", zap.Error(err))
		}
	case reading.Percent > threshold && s.isLow:
		s.isLow = false
		s.logger.Info("battery: restored", zap.Float64("percent", reading.Percent))
		if err := s.notifier.NotifyBatteryRestored(ctx, reading.Percent); err != nil {
			s.logger.Error("battery: failed to send restored notification", zap.Error(err))
		}
	}

	return nil
}
