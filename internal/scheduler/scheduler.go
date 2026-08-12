// Package scheduler wires all recurring background jobs — connectivity
// checks, speed tests, retention cleanup, and daily/weekly summary
// notifications — and coordinates their graceful shutdown.
package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/gopheramol/NetWatch/internal/analytics"
	"github.com/gopheramol/NetWatch/internal/battery"
	"github.com/gopheramol/NetWatch/internal/models"
	"github.com/gopheramol/NetWatch/internal/monitor"
	"github.com/gopheramol/NetWatch/internal/repository"
	"github.com/gopheramol/NetWatch/internal/services/retention"
	"github.com/gopheramol/NetWatch/internal/services/speedtest"
	"github.com/gopheramol/NetWatch/internal/sysmetrics"
	"github.com/gopheramol/NetWatch/internal/telegram"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Scheduler owns every recurring background job in the application.
type Scheduler struct {
	monitorSvc    *monitor.Service
	speedtestSvc  *speedtest.Service
	retentionSvc  *retention.Service
	batterySvc    *battery.Service
	sysMetricsSvc sysmetrics.Service
	analytics     analytics.Engine
	notifier      telegram.Notifier
	settingsRepo  repository.SettingsRepository
	connRepo      repository.ConnectivityRepository
	logger        *zap.Logger

	defaultMonitorInterval   time.Duration
	defaultSpeedtestInterval time.Duration
	cleanupInterval          time.Duration
	batteryEnabled           bool
	batteryInterval          time.Duration
	sysMetricsEnabled        bool
	sysMetricsInterval       time.Duration

	cron   *cron.Cron
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// Config controls the interval-based jobs. Daily/weekly summaries run on
// fixed wall-clock schedules (see Start) rather than intervals.
type Config struct {
	MonitorInterval    time.Duration
	SpeedTestInterval  time.Duration
	CleanupInterval    time.Duration
	BatteryEnabled     bool
	BatteryInterval    time.Duration
	SysMetricsEnabled  bool
	SysMetricsInterval time.Duration
}

// New builds a Scheduler with all its dependencies.
func New(
	cfg Config,
	monitorSvc *monitor.Service,
	speedtestSvc *speedtest.Service,
	retentionSvc *retention.Service,
	batterySvc *battery.Service,
	sysMetricsSvc sysmetrics.Service,
	engine analytics.Engine,
	notifier telegram.Notifier,
	settingsRepo repository.SettingsRepository,
	connRepo repository.ConnectivityRepository,
	logger *zap.Logger,
) *Scheduler {
	return &Scheduler{
		monitorSvc:               monitorSvc,
		speedtestSvc:             speedtestSvc,
		retentionSvc:             retentionSvc,
		batterySvc:               batterySvc,
		sysMetricsSvc:            sysMetricsSvc,
		analytics:                engine,
		notifier:                 notifier,
		settingsRepo:             settingsRepo,
		connRepo:                 connRepo,
		logger:                   logger,
		defaultMonitorInterval:   cfg.MonitorInterval,
		defaultSpeedtestInterval: cfg.SpeedTestInterval,
		cleanupInterval:          cfg.CleanupInterval,
		batteryEnabled:           cfg.BatteryEnabled,
		batteryInterval:          cfg.BatteryInterval,
		sysMetricsEnabled:        cfg.SysMetricsEnabled,
		sysMetricsInterval:       cfg.SysMetricsInterval,
		cron:                     cron.New(),
	}
}

// Start launches every background job. It returns immediately; jobs run in
// their own goroutines until the parent context is canceled.
func (s *Scheduler) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	s.wg.Add(3)
	go s.runLoop(ctx, "monitor", s.monitorIntervalFromSettings, true, func(loopCtx context.Context) {
		if err := s.monitorSvc.RunCheck(loopCtx); err != nil {
			s.logger.Error("scheduler: monitor check failed", zap.Error(err))
		}
	})
	go s.runLoop(ctx, "speedtest", s.speedtestIntervalFromSettings, false, func(loopCtx context.Context) {
		if _, err := s.speedtestSvc.Run(loopCtx); err != nil {
			s.logger.Error("scheduler: speed test failed", zap.Error(err))
		}
	})
	go s.runLoop(ctx, "retention", func() time.Duration { return s.cleanupInterval }, true, func(loopCtx context.Context) {
		if err := s.retentionSvc.Run(loopCtx); err != nil {
			s.logger.Error("scheduler: retention cleanup failed", zap.Error(err))
		}
	})

	if s.batteryEnabled {
		s.wg.Add(1)
		go s.runLoop(ctx, "battery", func() time.Duration { return s.batteryInterval }, true, func(loopCtx context.Context) {
			if err := s.batterySvc.Run(loopCtx); err != nil {
				s.logger.Error("scheduler: battery check failed", zap.Error(err))
			}
		})
	}

	if s.sysMetricsEnabled && s.sysMetricsSvc != nil {
		s.wg.Add(1)
		go s.runLoop(ctx, "sysmetrics", func() time.Duration { return s.sysMetricsInterval }, true, func(loopCtx context.Context) {
			if _, err := s.sysMetricsSvc.Collect(loopCtx); err != nil {
				s.logger.Error("scheduler: sysmetrics collection failed", zap.Error(err))
			}
		})
	}


	if _, err := s.cron.AddFunc("0 * * * *", func() { s.sendHourlyLatencySummary(ctx) }); err != nil {
		s.logger.Error("scheduler: failed to register hourly latency summary job", zap.Error(err))
	}
	if _, err := s.cron.AddFunc("59 23 * * *", func() { s.sendDailySummary(ctx) }); err != nil {
		s.logger.Error("scheduler: failed to register daily summary job", zap.Error(err))
	}
	if _, err := s.cron.AddFunc("59 23 * * SUN", func() { s.sendWeeklySummary(ctx) }); err != nil {
		s.logger.Error("scheduler: failed to register weekly summary job", zap.Error(err))
	}
	s.cron.Start()

	s.logger.Info("scheduler: started",
		zap.Duration("monitor_interval", s.defaultMonitorInterval),
		zap.Duration("speedtest_interval", s.defaultSpeedtestInterval),
		zap.Duration("cleanup_interval", s.cleanupInterval),
	)
}

// monitorIntervalFromSettings and speedtestIntervalFromSettings let
// POST /api/settings change background job cadence without a restart: each
// tick re-reads the current setting and runLoop resets its ticker if it changed.
func (s *Scheduler) monitorIntervalFromSettings() time.Duration {
	settings, err := s.settingsRepo.Get(context.Background())
	if err != nil || settings.MonitorIntervalSec <= 0 {
		return s.defaultMonitorInterval
	}
	return time.Duration(settings.MonitorIntervalSec) * time.Second
}

func (s *Scheduler) speedtestIntervalFromSettings() time.Duration {
	settings, err := s.settingsRepo.Get(context.Background())
	if err != nil || settings.SpeedIntervalMinutes <= 0 {
		return s.defaultSpeedtestInterval
	}
	return time.Duration(settings.SpeedIntervalMinutes) * time.Minute
}

// runLoop runs fn on a ticker until ctx is canceled, re-resolving the
// interval via getInterval after every tick so a live settings change takes
// effect without restarting the process. If runImmediately is true, fn also
// runs once right away instead of waiting for the first tick.
func (s *Scheduler) runLoop(ctx context.Context, name string, getInterval func() time.Duration, runImmediately bool, fn func(context.Context)) {
	defer s.wg.Done()

	if runImmediately {
		fn(ctx)
	}

	currentInterval := getInterval()
	ticker := time.NewTicker(currentInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler: stopping loop", zap.String("job", name))
			return
		case <-ticker.C:
			fn(ctx)
			if next := getInterval(); next != currentInterval {
				currentInterval = next
				ticker.Reset(currentInterval)
				s.logger.Info("scheduler: interval updated", zap.String("job", name), zap.Duration("new_interval", currentInterval))
			}
		}
	}
}

func (s *Scheduler) sendDailySummary(ctx context.Context) {
	today := time.Now().Format("2006-01-02")
	stats, err := s.analytics.GetDaily(ctx, today)
	if err != nil {
		s.logger.Error("scheduler: failed to load daily stats for summary", zap.Error(err))
		return
	}
	if err := s.notifier.NotifyDailySummary(ctx, *stats); err != nil {
		s.logger.Error("scheduler: failed to send daily summary", zap.Error(err))
	}
}

func (s *Scheduler) sendWeeklySummary(ctx context.Context) {
	now := time.Now()
	days, err := s.analytics.GetDailyRange(ctx, now.AddDate(0, 0, -6), now)
	if err != nil {
		s.logger.Error("scheduler: failed to load week of daily stats", zap.Error(err))
		return
	}
	if err := s.notifier.NotifyWeeklySummary(ctx, days); err != nil {
		s.logger.Error("scheduler: failed to send weekly summary", zap.Error(err))
	}
}

func (s *Scheduler) sendHourlyLatencySummary(ctx context.Context) {
	now := time.Now()
	from := now.Add(-1 * time.Hour)
	checks, err := s.connRepo.Range(ctx, from, now, 0)
	if err != nil {
		s.logger.Error("scheduler: failed to load hourly checks for latency summary", zap.Error(err))
		return
	}

	var (
		sum        float64
		minLatency float64 = -1
		maxLatency float64 = 0
		count      int
	)
	for _, c := range checks {
		if c.Status == models.StatusUp && c.LatencyMs > 0 {
			if minLatency < 0 || c.LatencyMs < minLatency {
				minLatency = c.LatencyMs
			}
			if c.LatencyMs > maxLatency {
				maxLatency = c.LatencyMs
			}
			sum += c.LatencyMs
			count++
		}
	}

	if count == 0 {
		s.logger.Info("scheduler: no valid latency checks in the past hour for summary")
		return
	}

	avgLatency := sum / float64(count)
	if err := s.notifier.NotifyHourlyLatencySummary(ctx, avgLatency, minLatency, maxLatency, count); err != nil {
		s.logger.Error("scheduler: failed to send hourly latency summary", zap.Error(err))
	}
}

// Stop cancels every background loop, stops the cron scheduler, and waits
// for in-flight jobs to finish (bounded by the caller's context deadline).
func (s *Scheduler) Stop(ctx context.Context) {
	if s.cancel != nil {
		s.cancel()
	}
	cronCtx := s.cron.Stop()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		<-cronCtx.Done()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("scheduler: stopped cleanly")
	case <-ctx.Done():
		s.logger.Warn("scheduler: shutdown deadline exceeded, some jobs may still be running")
	}
}
