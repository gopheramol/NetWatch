// Command server is the NetWatch entry point: it wires configuration,
// storage, services, the scheduler, and the REST API together, then runs
// until it receives a shutdown signal.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/gopheramol/NetWatch/internal/analytics"
	"github.com/gopheramol/NetWatch/internal/api"
	"github.com/gopheramol/NetWatch/internal/battery"
	"github.com/gopheramol/NetWatch/internal/config"
	"github.com/gopheramol/NetWatch/internal/database"
	"github.com/gopheramol/NetWatch/internal/heartbeat"
	"github.com/gopheramol/NetWatch/internal/models"
	"github.com/gopheramol/NetWatch/internal/monitor"
	"github.com/gopheramol/NetWatch/internal/repository"
	"github.com/gopheramol/NetWatch/internal/scheduler"
	"github.com/gopheramol/NetWatch/internal/services/retention"
	"github.com/gopheramol/NetWatch/internal/services/speedtest"
	"github.com/gopheramol/NetWatch/internal/sysmetrics"
	"github.com/gopheramol/NetWatch/internal/telegram"
	"github.com/gopheramol/NetWatch/internal/utils"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "", "directory containing config.yaml")
	flag.Parse()

	// Optional: load NETWATCH_* overrides (e.g. Telegram credentials) from a
	// local .env file. Silently ignored if absent — production deployments
	// are expected to set real environment variables instead.
	_ = godotenv.Load()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger, err := utils.NewLogger(cfg.Logging.Level, cfg.Logging.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	if err := run(cfg, logger); err != nil {
		logger.Fatal("server exited with error", zap.Error(err))
	}
}

func run(cfg *config.Config, logger *zap.Logger) error {
	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("failed to close database", zap.Error(err))
		}
	}()

	connRepo := repository.NewConnectivityRepository(db)
	speedRepo := repository.NewSpeedTestRepository(db)
	downtimeRepo := repository.NewDowntimeRepository(db)
	dailyRepo := repository.NewDailyStatsRepository(db)
	monthlyRepo := repository.NewMonthlyStatsRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	sysRepo := repository.NewSystemMetricsRepository(db)
	settingsRepo := repository.NewSettingsRepository(db, models.Settings{
		TelegramEnabled:        cfg.Telegram.Enabled,
		TelegramBotToken:       cfg.Telegram.BotToken,
		TelegramChatID:         cfg.Telegram.ChatID,
		MonitorIntervalSec:     cfg.Monitor.IntervalSeconds,
		SpeedIntervalMinutes:   cfg.SpeedTest.IntervalMinutes,
		SpeedReportEnabled:     cfg.SpeedTest.ReportEnabled,
		RetentionDays:          cfg.Retention.RawDataDays,
		BatteryLowThresholdPct: cfg.Battery.LowThresholdPct,
	})

	engine := analytics.NewEngine(dailyRepo, monthlyRepo, logger)
	notifier := telegram.NewService(settingsRepo, notifRepo, logger)
	batteryReader := battery.NewLinuxReader()
	sysMetricsSvc := sysmetrics.NewService(sysRepo, batteryReader, logger)

	monitorSvc := monitor.NewService(cfg.Monitor, connRepo, downtimeRepo, speedRepo, engine, notifier, logger)
	speedtestSvc := speedtest.NewService(
		speedtest.NewOoklaProvider(), speedRepo, engine, notifier, settingsRepo,
		cfg.SpeedTest.MinDownloadMbps, cfg.SpeedTest.MinUploadMbps, cfg.SpeedTest.ReportEnabled, logger,
	)
	retentionSvc := retention.NewService(connRepo, sysRepo, settingsRepo, cfg.Retention.RawDataDays, logger)
	batterySvc := battery.NewService(battery.NewLinuxReader(), settingsRepo, notifier, cfg.Battery.LowThresholdPct, logger)

	sched := scheduler.New(scheduler.Config{
		MonitorInterval:    time.Duration(cfg.Monitor.IntervalSeconds) * time.Second,
		SpeedTestInterval:  time.Duration(cfg.SpeedTest.IntervalMinutes) * time.Minute,
		CleanupInterval:    cfg.Retention.CleanupInterval,
		BatteryEnabled:     cfg.Battery.Enabled,
		BatteryInterval:    cfg.Battery.CheckInterval,
		SysMetricsEnabled:  cfg.SysMetrics.Enabled,
		SysMetricsInterval: time.Duration(cfg.SysMetrics.IntervalSeconds) * time.Second,
	}, monitorSvc, speedtestSvc, retentionSvc, batterySvc, sysMetricsSvc, engine, notifier, settingsRepo, connRepo, logger)

	router := api.NewRouter(api.Dependencies{
		MonitorSvc:    monitorSvc,
		SpeedTestSvc:  speedtestSvc,
		SysMetricsSvc: sysMetricsSvc,
		Analytics:     engine,
		Notifier:      notifier,
		ConnRepo:      connRepo,
		SpeedRepo:     speedRepo,
		DowntimeRepo:  downtimeRepo,
		SettingsRepo:  settingsRepo,
		Logger:        logger,
		CORSOrigins:   cfg.Server.CORSOrigins,
	})

	botListener := telegram.NewBotListener(telegram.NewClient(), settingsRepo, monitorSvc, speedtestSvc, sysMetricsSvc, downtimeRepo, logger)

	httpServer := &http.Server{
		Addr:    net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port)),
		Handler: router,
	}

	// sigCh (rather than signal.NotifyContext) lets us report *which* signal
	// triggered shutdown, so the Telegram message can say SIGTERM (docker
	// stop/restart) vs SIGINT (manual interrupt) instead of a generic reason.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var caughtSignal os.Signal
	go func() {
		caughtSignal = <-sigCh
		cancel()
	}()

	sched.Start(ctx)
	botListener.Start(ctx)

	// A heartbeat file survives process restarts. If it says the previous
	// run never shut down cleanly, that run was killed without a chance to
	// run any code — a SIGKILL from an OOM-kill, `docker kill`, a host
	// crash, or a power/network loss. We can't know which, but we can say
	// "not a normal stop" and report when it was last seen alive.
	heartbeatPath := filepath.Join(filepath.Dir(cfg.Database.Path), ".heartbeat.json")
	tracker := heartbeat.New(heartbeatPath)
	if prev, ok := tracker.Previous(); ok && !prev.Clean {
		if err := notifier.NotifyServiceRecovered(ctx, prev.LastSeen); err != nil {
			logger.Warn("failed to send service-recovered notification", zap.Error(err))
		}
	} else if err := notifier.NotifyServiceStarted(ctx); err != nil {
		logger.Warn("failed to send service-started notification", zap.Error(err))
	}
	if err := tracker.Start(time.Now()); err != nil {
		logger.Warn("heartbeat: failed to record startup", zap.Error(err))
	}

	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := tracker.Touch(time.Now()); err != nil {
					logger.Warn("heartbeat: failed to update", zap.Error(err))
				}
			}
		}
	}()

	serverErrCh := make(chan error, 1)
	go func() {
		logger.Info("server: listening", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	stopReason := "graceful shutdown, unknown trigger"
	select {
	case <-ctx.Done():
		stopReason = shutdownReasonFromSignal(caughtSignal)
		logger.Info("server: shutdown signal received", zap.String("reason", stopReason))
	case err := <-serverErrCh:
		if err != nil {
			stopReason = fmt.Sprintf("server error: %v", err)
			notifyCtx, notifyCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
			if notifyErr := notifier.NotifyServiceStopped(notifyCtx, stopReason); notifyErr != nil {
				logger.Warn("failed to send service-stopped notification", zap.Error(notifyErr))
			}
			notifyCancel()
			if hbErr := tracker.MarkClean(time.Now()); hbErr != nil {
				logger.Warn("heartbeat: failed to mark clean shutdown", zap.Error(hbErr))
			}
			cancel()
			<-heartbeatDone
			return fmt.Errorf("http server error: %w", err)
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := notifier.NotifyServiceStopped(shutdownCtx, stopReason); err != nil {
		logger.Warn("failed to send service-stopped notification", zap.Error(err))
	}
	if err := tracker.MarkClean(time.Now()); err != nil {
		logger.Warn("heartbeat: failed to mark clean shutdown", zap.Error(err))
	}
	<-heartbeatDone
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("server: graceful shutdown failed", zap.Error(err))
	}
	sched.Stop(shutdownCtx)

	logger.Info("server: shutdown complete")
	return nil
}

// shutdownReasonFromSignal turns the OS signal that triggered shutdown into
// a human-readable reason for the Telegram notification.
func shutdownReasonFromSignal(sig os.Signal) string {
	switch sig {
	case syscall.SIGTERM:
		return "SIGTERM — container/service stop requested (docker stop, compose down, or redeploy)"
	case syscall.SIGINT:
		return "SIGINT — manual interrupt (Ctrl+C)"
	case nil:
		return "graceful shutdown, unknown trigger"
	default:
		return fmt.Sprintf("signal: %v", sig)
	}
}
