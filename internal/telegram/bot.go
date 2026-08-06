package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gopheramol/NetWatch/internal/models"
	"github.com/gopheramol/NetWatch/internal/repository"
	"go.uber.org/zap"
)

// StatusProvider exposes live status for Telegram status command.
type StatusProvider interface {
	GetCurrentStatus(ctx context.Context) (*models.CurrentStatus, error)
}

// SpeedTestRunner runs an on-demand speed test for Telegram speedtest command.
type SpeedTestRunner interface {
	Run(ctx context.Context) (*models.SpeedTestResult, error)
}

// SysMetricsProvider collects system metrics for Telegram metrics command.
type SysMetricsProvider interface {
	Collect(ctx context.Context) (*models.SystemMetrics, error)
}

// BotListener listens for Telegram bot commands using long-polling.
type BotListener struct {
	client        *Client
	settingsRepo  repository.SettingsRepository
	statusProv    StatusProvider
	speedtestRun  SpeedTestRunner
	sysMetricsProv SysMetricsProvider
	downtimeRepo  repository.DowntimeRepository
	logger        *zap.Logger
}

// NewBotListener builds a new Telegram command bot listener.
func NewBotListener(
	client *Client,
	settingsRepo repository.SettingsRepository,
	statusProv StatusProvider,
	speedtestRun SpeedTestRunner,
	sysMetricsProv SysMetricsProvider,
	downtimeRepo repository.DowntimeRepository,
	logger *zap.Logger,
) *BotListener {
	return &BotListener{
		client:         client,
		settingsRepo:   settingsRepo,
		statusProv:     statusProv,
		speedtestRun:   speedtestRun,
		sysMetricsProv: sysMetricsProv,
		downtimeRepo:   downtimeRepo,
		logger:         logger,
	}
}

// Start begins long-polling for incoming Telegram commands.
func (b *BotListener) Start(ctx context.Context) {
	go b.poll(ctx)
}

func (b *BotListener) poll(ctx context.Context) {
	b.logger.Info("telegram bot: listener started")
	offset := 0

	for {
		select {
		case <-ctx.Done():
			b.logger.Info("telegram bot: listener stopped")
			return
		default:
		}

		settings, err := b.settingsRepo.Get(ctx)
		if err != nil || !settings.TelegramEnabled || settings.TelegramBotToken == "" {
			time.Sleep(10 * time.Second)
			continue
		}

		updates, err := b.client.GetUpdates(ctx, settings.TelegramBotToken, offset, 30)
		if err != nil {
			if ctx.Err() == nil {
				b.logger.Warn("telegram bot: poll error", zap.Error(err))
			}
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}

			if update.Message == nil || update.Message.Text == "" {
				continue
			}

			senderChatID := strconv.FormatInt(update.Message.Chat.ID, 10)
			if settings.TelegramChatID != "" && senderChatID != settings.TelegramChatID {
				b.logger.Warn("telegram bot: ignoring unauthorized message", zap.String("from_chat_id", senderChatID))
				continue
			}

			b.handleCommand(ctx, settings, senderChatID, update.Message.Text)
		}
	}
}

func (b *BotListener) handleCommand(ctx context.Context, settings *models.Settings, chatID, text string) {
	cmd := strings.Fields(text)[0]
	if idx := strings.Index(cmd, "@"); idx != -1 {
		cmd = cmd[:idx]
	}

	b.logger.Info("telegram bot: handling command", zap.String("command", cmd))

	switch strings.ToLower(cmd) {
	case "/status":
		b.handleStatus(ctx, settings.TelegramBotToken, chatID)
	case "/metrics":
		b.handleMetrics(ctx, settings.TelegramBotToken, chatID)
	case "/speedtest":
		b.handleSpeedTest(ctx, settings.TelegramBotToken, chatID)
	case "/downtime":
		b.handleDowntime(ctx, settings.TelegramBotToken, chatID)
	case "/help", "/start":
		b.handleHelp(ctx, settings.TelegramBotToken, chatID)
	default:
		b.handleHelp(ctx, settings.TelegramBotToken, chatID)
	}
}

func (b *BotListener) handleStatus(ctx context.Context, token, chatID string) {
	if b.statusProv == nil {
		_ = b.client.SendMessage(ctx, token, chatID, "⚠️ Status service is not available.")
		return
	}
	status, err := b.statusProv.GetCurrentStatus(ctx)
	if err != nil {
		_ = b.client.SendMessage(ctx, token, chatID, fmt.Sprintf("⚠️ <b>Status Error:</b> %v", err))
		return
	}

	emoji := "🟢"
	if status.Status == models.StatusDown {
		emoji = "🔴"
	} else if status.Status == models.StatusDegraded {
		emoji = "🟠"
	}

	msg := fmt.Sprintf(
		"<b>%s NetWatch Status: %s</b>\n\n"+
			"⏱ <b>Last Check:</b> %s\n"+
			"⚡️ <b>Latency:</b> %.1f ms\n"+
			"📊 <b>Month Availability:</b> %.2f%%\n"+
			"⌛️ <b>Current Streak:</b> %s",
		emoji,
		strings.ToUpper(string(status.Status)),
		status.LastCheck.Format("15:04:05 MST"),
		status.LatencyMs,
		status.MonthAvailabilityPct,
		time.Duration(status.CurrentStreakSec)*time.Second,
	)

	_ = b.client.SendMessage(ctx, token, chatID, msg)
}

func (b *BotListener) handleMetrics(ctx context.Context, token, chatID string) {
	if b.sysMetricsProv == nil {
		_ = b.client.SendMessage(ctx, token, chatID, "⚠️ System metrics service is not initialized.")
		return
	}

	m, err := b.sysMetricsProv.Collect(ctx)
	if err != nil {
		_ = b.client.SendMessage(ctx, token, chatID, fmt.Sprintf("⚠️ <b>Metrics Error:</b> %v", err))
		return
	}

	tempStr := "N/A"
	if m.CPUTempC > 0 {
		tempStr = fmt.Sprintf("%.1f °C", m.CPUTempC)
	}

	batteryStr := "🔌 AC Power (No Battery)"
	if m.BatteryPresent {
		status := "Discharging"
		if m.BatteryCharging {
			status = "Charging"
		}
		batteryStr = fmt.Sprintf("%.0f%% (%s)", m.BatteryPercent, status)
	}

	msg := fmt.Sprintf(
		"<b>💻 System Metrics Snapshot</b>\n\n"+
			"⚡️ <b>CPU Usage:</b> %.1f%%\n"+
			"🌡 <b>CPU Temp:</b> %s\n"+
			"🧠 <b>Memory (RAM):</b> %.1f / %.1f MB (%.1f%%)\n"+
			"💾 <b>Disk Usage (/):</b> %.1f / %.1f GB (%.1f%%)\n"+
			"🔋 <b>Battery / UPS:</b> %s",
		m.CPUPercent,
		tempStr,
		m.RAMUsedMB, m.RAMTotalMB, m.RAMPercent,
		m.DiskUsedGB, m.DiskTotalGB, m.DiskPercent,
		batteryStr,
	)

	_ = b.client.SendMessage(ctx, token, chatID, msg)
}

func (b *BotListener) handleSpeedTest(ctx context.Context, token, chatID string) {
	if b.speedtestRun == nil {
		_ = b.client.SendMessage(ctx, token, chatID, "⚠️ Speed test service is not available.")
		return
	}

	_ = b.client.SendMessage(ctx, token, chatID, "⏳ <i>Running speed test in background... Please wait.</i>")

	go func() {
		testCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		result, err := b.speedtestRun.Run(testCtx)
		if err != nil {
			_ = b.client.SendMessage(testCtx, token, chatID, fmt.Sprintf("❌ <b>Speed test failed:</b> %v", err))
			return
		}

		msg := fmt.Sprintf(
			"✅ <b>Speed Test Result</b>\n\n"+
				"⬇️ <b>Download:</b> %.2f Mbps\n"+
				"⬆️ <b>Upload:</b> %.2f Mbps\n"+
				"⚡️ <b>Ping:</b> %.1f ms | <b>Jitter:</b> %.1f ms\n"+
				"🏢 <b>ISP:</b> %s | <b>Server:</b> %s",
			result.DownloadMbps, result.UploadMbps, result.PingMs, result.JitterMs, result.ISP, result.Server,
		)
		_ = b.client.SendMessage(testCtx, token, chatID, msg)
	}()
}

func (b *BotListener) handleDowntime(ctx context.Context, token, chatID string) {
	outages, err := b.downtimeRepo.List(ctx, 5)
	if err != nil || len(outages) == 0 {
		_ = b.client.SendMessage(ctx, token, chatID, "🎉 <b>No downtime incidents recorded!</b>")
		return
	}

	var sb strings.Builder
	sb.WriteString("<b>🔻 Recent Outage History (Last 5)</b>\n\n")

	for _, o := range outages {
		durStr := "ongoing"
		if o.Resolved {
			durStr = o.Duration.Round(time.Second).String()
		}
		sb.WriteString(fmt.Sprintf(
			"• <b>%s</b> (%s)\n  Reason: %s\n\n",
			o.StartTime.Format("2006-01-02 15:04:05"),
			durStr,
			o.Reason,
		))
	}

	_ = b.client.SendMessage(ctx, token, chatID, sb.String())
}

func (b *BotListener) handleHelp(ctx context.Context, token, chatID string) {
	msg := "<b>🤖 NetWatch Bot Commands</b>\n\n" +
		"/status - View live connection status & availability\n" +
		"/metrics - View server CPU, RAM, Disk & Temp\n" +
		"/speedtest - Trigger an on-demand speed test\n" +
		"/downtime - View recent outage history\n" +
		"/help - Show this help menu"

	_ = b.client.SendMessage(ctx, token, chatID, msg)
}
