package telegram

import (
	"context"
	"time"

	"github.com/gopheramol/NetWatch/internal/models"
	"github.com/gopheramol/NetWatch/internal/repository"
	"github.com/gopheramol/NetWatch/internal/utils"
	"go.uber.org/zap"
)

// Notifier sends the notification types the rest of the application cares
// about. Implementations are responsible for honoring the current on/off
// setting and recording an audit trail via the notification repository.
type Notifier interface {
	NotifyDown(ctx context.Context, reason string, since time.Time) error
	NotifyRestored(ctx context.Context, outage models.Outage) error
	NotifyDailySummary(ctx context.Context, stats models.DailyStats) error
	NotifyWeeklySummary(ctx context.Context, days []models.DailyStats) error
	NotifyTest(ctx context.Context) error
	NotifyServiceStarted(ctx context.Context) error
	NotifyServiceStopped(ctx context.Context) error
	NotifyBatteryLow(ctx context.Context, pct, thresholdPct float64) error
	NotifyBatteryRestored(ctx context.Context, pct float64) error
	NotifyHighLatency(ctx context.Context, latencyMs, thresholdMs float64) error
	NotifyLatencyNormal(ctx context.Context, latencyMs float64) error
	NotifySlowSpeed(ctx context.Context, result models.SpeedTestResult, minDownloadMbps, minUploadMbps float64) error
	NotifySpeedReport(ctx context.Context, result models.SpeedTestResult) error
}

type service struct {
	client       *Client
	settingsRepo repository.SettingsRepository
	notifRepo    repository.NotificationRepository
	logger       *zap.Logger
}

// NewService builds the default Notifier implementation.
func NewService(settingsRepo repository.SettingsRepository, notifRepo repository.NotificationRepository, logger *zap.Logger) Notifier {
	return &service{
		client:       NewClient(),
		settingsRepo: settingsRepo,
		notifRepo:    notifRepo,
		logger:       logger,
	}
}

func (s *service) send(ctx context.Context, notifType models.NotificationType, text string) error {
	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		s.logger.Error("telegram: failed to load settings", zap.Error(err))
		return err
	}

	if !settings.TelegramEnabled {
		s.logger.Debug("telegram: notifications disabled, skipping", zap.String("type", string(notifType)))
		return nil
	}

	sendErr := s.client.SendMessage(ctx, settings.TelegramBotToken, settings.TelegramChatID, text)

	record := &models.Notification{
		ID:        utils.NewID(),
		Timestamp: time.Now(),
		Type:      notifType,
		Message:   text,
		Success:   sendErr == nil,
	}
	if sendErr != nil {
		record.Error = sendErr.Error()
		s.logger.Error("telegram: failed to send notification",
			zap.String("type", string(notifType)), zap.Error(sendErr))
	} else {
		s.logger.Info("telegram: notification sent", zap.String("type", string(notifType)))
	}

	if saveErr := s.notifRepo.Save(ctx, record); saveErr != nil {
		s.logger.Error("telegram: failed to persist notification record", zap.Error(saveErr))
	}

	return sendErr
}

func (s *service) NotifyDown(ctx context.Context, reason string, since time.Time) error {
	return s.send(ctx, models.NotificationDown, downMessage(reason, since))
}

func (s *service) NotifyRestored(ctx context.Context, outage models.Outage) error {
	return s.send(ctx, models.NotificationRestored, restoredMessage(outage))
}

func (s *service) NotifyDailySummary(ctx context.Context, stats models.DailyStats) error {
	return s.send(ctx, models.NotificationDailySummary, dailySummaryMessage(stats))
}

func (s *service) NotifyWeeklySummary(ctx context.Context, days []models.DailyStats) error {
	return s.send(ctx, models.NotificationWeeklySummary, weeklySummaryMessage(days))
}

func (s *service) NotifyTest(ctx context.Context) error {
	return s.send(ctx, models.NotificationTest, testMessage())
}

func (s *service) NotifyServiceStarted(ctx context.Context) error {
	return s.send(ctx, models.NotificationServiceUp, serviceStartedMessage())
}

func (s *service) NotifyServiceStopped(ctx context.Context) error {
	return s.send(ctx, models.NotificationServiceDown, serviceStoppedMessage())
}

func (s *service) NotifyBatteryLow(ctx context.Context, pct, thresholdPct float64) error {
	return s.send(ctx, models.NotificationBatteryLow, batteryLowMessage(pct, thresholdPct))
}

func (s *service) NotifyBatteryRestored(ctx context.Context, pct float64) error {
	return s.send(ctx, models.NotificationBatteryOk, batteryRestoredMessage(pct))
}

func (s *service) NotifyHighLatency(ctx context.Context, latencyMs, thresholdMs float64) error {
	return s.send(ctx, models.NotificationHighLatency, highLatencyMessage(latencyMs, thresholdMs))
}

func (s *service) NotifyLatencyNormal(ctx context.Context, latencyMs float64) error {
	return s.send(ctx, models.NotificationLatencyNormal, latencyNormalMessage(latencyMs))
}

func (s *service) NotifySlowSpeed(ctx context.Context, result models.SpeedTestResult, minDownloadMbps, minUploadMbps float64) error {
	return s.send(ctx, models.NotificationSlowSpeed, slowSpeedMessage(result, minDownloadMbps, minUploadMbps))
}

func (s *service) NotifySpeedReport(ctx context.Context, result models.SpeedTestResult) error {
	return s.send(ctx, models.NotificationSpeedReport, speedReportMessage(result))
}
