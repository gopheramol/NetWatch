package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gopheramol/NetWatch/internal/analytics"
	"github.com/gopheramol/NetWatch/internal/config"
	"github.com/gopheramol/NetWatch/internal/database"
	"github.com/gopheramol/NetWatch/internal/models"
	"github.com/gopheramol/NetWatch/internal/repository"
	"github.com/gopheramol/NetWatch/internal/telegram"
	"go.uber.org/zap"
)

// Service runs connectivity checks, detects outages from consecutive
// failures, and answers live status queries. It is the single writer of
// in-memory connectivity state; all reads go through GetCurrentStatus.
type Service struct {
	checker                *Checker
	connRepo               repository.ConnectivityRepository
	downtimeRepo           repository.DowntimeRepository
	speedRepo              repository.SpeedTestRepository
	analytics              analytics.Engine
	notifier               telegram.Notifier
	logger                 *zap.Logger
	threshold              int
	highLatencyThresholdMs float64
	highLatencyOccurrences int
	highLatencyCooldown    time.Duration

	mu                     sync.Mutex
	consecutiveFailures    int
	failureStreakStart     time.Time
	currentStatus          models.ConnectivityStatus
	lastCheck              *models.ConnectivityCheck
	uptimeSince            time.Time
	consecutiveHighLatency int
	consecutiveNormalChecks int
	isHighLatency          bool
	lastHighLatencyAlert   time.Time
}

// NewService builds the connectivity Service, restoring in-memory state
// from the database so a service restart doesn't lose an in-progress outage.
func NewService(
	cfg config.MonitorConfig,
	connRepo repository.ConnectivityRepository,
	downtimeRepo repository.DowntimeRepository,
	speedRepo repository.SpeedTestRepository,
	engine analytics.Engine,
	notifier telegram.Notifier,
	logger *zap.Logger,
) *Service {
	cooldownMin := cfg.HighLatencyCooldownMin
	if cooldownMin <= 0 {
		cooldownMin = 60
	}

	s := &Service{
		checker:                NewChecker(cfg),
		connRepo:               connRepo,
		downtimeRepo:           downtimeRepo,
		speedRepo:              speedRepo,
		analytics:              engine,
		notifier:               notifier,
		logger:                 logger,
		threshold:              cfg.FailureThreshold,
		highLatencyThresholdMs: cfg.HighLatencyThresholdMs,
		highLatencyOccurrences: cfg.HighLatencyOccurrences,
		highLatencyCooldown:    time.Duration(cooldownMin) * time.Minute,
		currentStatus:          models.StatusUp,
		uptimeSince:            time.Now(),
	}

	ctx := context.Background()
	if ongoing, err := downtimeRepo.GetOngoing(ctx); err == nil && ongoing != nil {
		s.currentStatus = models.StatusDown
		s.failureStreakStart = ongoing.StartTime
		s.consecutiveFailures = s.threshold
	}
	if latest, err := connRepo.Latest(ctx); err == nil && latest != nil {
		s.lastCheck = latest
	}

	return s
}

// RunCheck executes one connectivity probe, persists it, feeds analytics,
// and opens/closes outage records as the connection goes down or recovers.
func (s *Service) RunCheck(ctx context.Context) error {
	check := s.checker.Check(ctx)

	if err := s.connRepo.Save(ctx, &check); err != nil {
		return fmt.Errorf("saving connectivity check: %w", err)
	}
	if err := s.analytics.RecordCheck(ctx, check); err != nil {
		s.logger.Error("monitor: failed to record analytics for check", zap.Error(err))
	}

	s.logger.Info("monitor: connectivity check",
		zap.String("status", string(check.Status)),
		zap.Float64("latency_ms", check.LatencyMs),
		zap.Bool("dns_ok", check.DNSOk),
		zap.Bool("http_ok", check.HTTPOk),
		zap.Bool("tcp_ok", check.PingOk),
	)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastCheck = &check

	isFailure := check.Status != models.StatusUp
	if isFailure {
		if s.consecutiveFailures == 0 {
			s.failureStreakStart = check.Timestamp
		}
		s.consecutiveFailures++

		if s.consecutiveFailures == s.threshold && s.currentStatus != models.StatusDown {
			s.currentStatus = models.StatusDown
			outage := &models.Outage{
				StartTime: s.failureStreakStart,
				Reason:    check.FailureReason,
				Resolved:  false,
			}
			if err := s.downtimeRepo.Create(ctx, outage); err != nil {
				s.logger.Error("monitor: failed to create outage record", zap.Error(err))
			}
			if err := s.notifier.NotifyDown(ctx, check.FailureReason, s.failureStreakStart); err != nil {
				s.logger.Error("monitor: failed to send down notification", zap.Error(err))
			}
			s.logger.Warn("monitor: internet down", zap.Time("since", s.failureStreakStart), zap.String("reason", check.FailureReason))
		}
		return nil
	}

	wasDown := s.currentStatus == models.StatusDown
	s.consecutiveFailures = 0
	s.currentStatus = models.StatusUp
	s.uptimeSince = check.Timestamp

	s.evaluateLatency(ctx, check)

	if wasDown {
		ongoing, err := s.downtimeRepo.GetOngoing(ctx)
		if err != nil {
			s.logger.Error("monitor: no ongoing outage found while recovering", zap.Error(err))
			return nil
		}
		now := check.Timestamp
		ongoing.EndTime = &now
		ongoing.Duration = now.Sub(ongoing.StartTime)
		ongoing.Resolved = true

		if err := s.downtimeRepo.Update(ctx, ongoing); err != nil {
			s.logger.Error("monitor: failed to update outage record", zap.Error(err))
		}
		if err := s.analytics.RecordOutageClosed(ctx, *ongoing); err != nil {
			s.logger.Error("monitor: failed to record outage analytics", zap.Error(err))
		}
		if err := s.notifier.NotifyRestored(ctx, *ongoing); err != nil {
			s.logger.Error("monitor: failed to send restored notification", zap.Error(err))
		}
		s.logger.Warn("monitor: internet restored", zap.Duration("duration", ongoing.Duration))
	}

	return nil
}

// evaluateLatency tracks high latency occurrences with smart throttling (1-hour cooldown)
// and hysteresis stabilization to avoid alert flapping. Caller must hold s.mu.
func (s *Service) evaluateLatency(ctx context.Context, check models.ConnectivityCheck) {
	if s.highLatencyThresholdMs <= 0 || s.highLatencyOccurrences <= 0 {
		return
	}

	cooldown := s.highLatencyCooldown
	if cooldown <= 0 {
		cooldown = 1 * time.Hour
	}

	if check.LatencyMs <= s.highLatencyThresholdMs {
		s.consecutiveNormalChecks++
		// Require consecutive normal checks to prevent alert flapping
		if s.isHighLatency && s.consecutiveNormalChecks >= s.highLatencyOccurrences {
			s.isHighLatency = false
			s.consecutiveHighLatency = 0
			s.consecutiveNormalChecks = 0
			if err := s.notifier.NotifyLatencyNormal(ctx, check.LatencyMs); err != nil {
				s.logger.Error("monitor: failed to send latency normal notification", zap.Error(err))
			}
			s.logger.Info("monitor: latency back to normal", zap.Float64("latency_ms", check.LatencyMs))
		}
		return
	}

	s.consecutiveNormalChecks = 0
	s.consecutiveHighLatency++

	if s.consecutiveHighLatency >= s.highLatencyOccurrences {
		s.isHighLatency = true

		// Smart Throttling: Only send high latency alert at most once per hour (cooldown)
		if time.Since(s.lastHighLatencyAlert) >= cooldown {
			s.lastHighLatencyAlert = time.Now()
			if err := s.notifier.NotifyHighLatency(ctx, check.LatencyMs, s.highLatencyThresholdMs); err != nil {
				s.logger.Error("monitor: failed to send high latency notification", zap.Error(err))
			}
			s.logger.Warn("monitor: high latency detected (throttled)",
				zap.Float64("latency_ms", check.LatencyMs),
				zap.Float64("threshold_ms", s.highLatencyThresholdMs),
				zap.Duration("cooldown", cooldown),
			)
		}
	}
}

// GetCurrentStatus builds a live snapshot combining in-memory state,
// pre-aggregated analytics, and the ISP reported by the latest speed test.
func (s *Service) GetCurrentStatus(ctx context.Context) (*models.CurrentStatus, error) {
	s.mu.Lock()
	status := s.currentStatus
	lastCheck := s.lastCheck
	uptimeSince := s.uptimeSince
	s.mu.Unlock()

	now := time.Now()
	result := &models.CurrentStatus{
		Status:    status,
		LastCheck: now,
	}
	if lastCheck != nil {
		result.LastCheck = lastCheck.Timestamp
		result.LatencyMs = lastCheck.LatencyMs
	}

	if isp, err := s.latestISP(ctx); err == nil {
		result.ISP = isp
	}

	if status == models.StatusUp {
		start := uptimeSince
		result.CurrentUptimeStart = &start
		result.CurrentStreakSec = now.Sub(uptimeSince).Seconds()
	}

	if ongoing, err := s.downtimeRepo.GetOngoing(ctx); err == nil && ongoing != nil {
		result.OngoingOutage = ongoing
	}

	today := now.Format("2006-01-02")
	if daily, err := s.analytics.GetDaily(ctx, today); err == nil {
		result.TodayDowntimeSec = daily.DowntimeSeconds
	}

	month := now.Format("2006-01")
	if monthly, err := s.analytics.GetMonthly(ctx, month); err == nil {
		result.MonthAvailabilityPct = monthly.AvailabilityPct
	}

	return result, nil
}

func (s *Service) latestISP(ctx context.Context) (string, error) {
	latest, err := s.speedRepo.Latest(ctx)
	if err != nil {
		if err == database.ErrNotFound {
			return "", nil
		}
		return "", err
	}
	return latest.ISP, nil
}
