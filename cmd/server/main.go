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
	"strconv"
	"syscall"
	"time"

	"github.com/gopheramol/NetWatch/internal/analytics"
	"github.com/gopheramol/NetWatch/internal/api"
	"github.com/gopheramol/NetWatch/internal/battery"
	"github.com/gopheramol/NetWatch/internal/config"
	"github.com/gopheramol/NetWatch/internal/database"
	"github.com/gopheramol/NetWatch/internal/models"
	"github.com/gopheramol/NetWatch/internal/monitor"
	"github.com/gopheramol/NetWatch/internal/repository"
	"github.com/gopheramol/NetWatch/internal/scheduler"
	"github.com/gopheramol/NetWatch/internal/services/retention"
	"github.com/gopheramol/NetWatch/internal/services/speedtest"
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

	monitorSvc := monitor.NewService(cfg.Monitor, connRepo, downtimeRepo, speedRepo, engine, notifier, logger)
	speedtestSvc := speedtest.NewService(
		speedtest.NewOoklaProvider(), speedRepo, engine, notifier, settingsRepo,
		cfg.SpeedTest.MinDownloadMbps, cfg.SpeedTest.MinUploadMbps, cfg.SpeedTest.ReportEnabled, logger,
	)
	retentionSvc := retention.NewService(connRepo, settingsRepo, cfg.Retention.RawDataDays, logger)
	batterySvc := battery.NewService(battery.NewLinuxReader(), settingsRepo, notifier, cfg.Battery.LowThresholdPct, logger)

	sched := scheduler.New(scheduler.Config{
		MonitorInterval:   time.Duration(cfg.Monitor.IntervalSeconds) * time.Second,
		SpeedTestInterval: time.Duration(cfg.SpeedTest.IntervalMinutes) * time.Minute,
		CleanupInterval:   cfg.Retention.CleanupInterval,
		BatteryEnabled:    cfg.Battery.Enabled,
		BatteryInterval:   cfg.Battery.CheckInterval,
	}, monitorSvc, speedtestSvc, retentionSvc, batterySvc, engine, notifier, settingsRepo, logger)

	router := api.NewRouter(api.Dependencies{
		MonitorSvc:   monitorSvc,
		SpeedTestSvc: speedtestSvc,
		Analytics:    engine,
		Notifier:     notifier,
		ConnRepo:     connRepo,
		SpeedRepo:    speedRepo,
		DowntimeRepo: downtimeRepo,
		SettingsRepo: settingsRepo,
		Logger:       logger,
		CORSOrigins:  cfg.Server.CORSOrigins,
	})

	httpServer := &http.Server{
		Addr:    net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port)),
		Handler: router,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	sched.Start(ctx)
	if err := notifier.NotifyServiceStarted(ctx); err != nil {
		logger.Warn("failed to send service-started notification", zap.Error(err))
	}

	serverErrCh := make(chan error, 1)
	go func() {
		logger.Info("server: listening", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Info("server: shutdown signal received")
	case err := <-serverErrCh:
		if err != nil {
			return fmt.Errorf("http server error: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := notifier.NotifyServiceStopped(shutdownCtx); err != nil {
		logger.Warn("failed to send service-stopped notification", zap.Error(err))
	}
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("server: graceful shutdown failed", zap.Error(err))
	}
	sched.Stop(shutdownCtx)

	logger.Info("server: shutdown complete")
	return nil
}
