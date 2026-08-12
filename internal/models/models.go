// Package models defines the core domain types shared across repository,
// service, and API layers.
package models

import "time"

// ConnectivityStatus represents the observed state of the Internet connection.
type ConnectivityStatus string

const (
	StatusUp       ConnectivityStatus = "up"
	StatusDown     ConnectivityStatus = "down"
	StatusDegraded ConnectivityStatus = "degraded"
)

// ConnectivityCheck is a single point-in-time connectivity measurement.
type ConnectivityCheck struct {
	Timestamp     time.Time          `json:"timestamp"`
	Status        ConnectivityStatus `json:"status"`
	LatencyMs     float64            `json:"latency_ms"`
	DNSOk         bool               `json:"dns_ok"`
	HTTPOk        bool               `json:"http_ok"`
	PingOk        bool               `json:"ping_ok"`
	FailureReason string             `json:"failure_reason,omitempty"`
	PacketLoss    float64            `json:"packet_loss"`
}

// Outage represents a single downtime incident, from detection to recovery.
type Outage struct {
	ID        string        `json:"id"`
	StartTime time.Time     `json:"start_time"`
	EndTime   *time.Time    `json:"end_time,omitempty"`
	Duration  time.Duration `json:"duration"`
	Reason    string        `json:"reason"`
	Resolved  bool          `json:"resolved"`
}

// SpeedTestResult stores the outcome of a bandwidth measurement.
type SpeedTestResult struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	DownloadMbps float64   `json:"download_mbps"`
	UploadMbps   float64   `json:"upload_mbps"`
	PingMs       float64   `json:"ping_ms"`
	JitterMs     float64   `json:"jitter_ms"`
	ISP          string    `json:"isp"`
	Server       string    `json:"server"`
	Provider     string    `json:"provider"`
}

// DailyStats is a continuously-maintained summary of a single calendar day.
type DailyStats struct {
	Date            string  `json:"date"` // YYYY-MM-DD
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
	MinSpeedMbps    float64 `json:"min_speed_mbps"`
	MaxSpeedMbps    float64 `json:"max_speed_mbps"`
	AvgSpeedMbps    float64 `json:"avg_speed_mbps"`
	DowntimeSeconds float64 `json:"downtime_seconds"`
	AvailabilityPct float64 `json:"availability_pct"`
	OutageCount     int     `json:"outage_count"`
	CheckCount      int64   `json:"check_count"`
	LatencySum      float64 `json:"latency_sum"`
	SpeedTestCount  int64   `json:"speed_test_count"`
	SpeedSum        float64 `json:"speed_sum"`
}

// MonthlyStats is a continuously-maintained summary of a single calendar month.
type MonthlyStats struct {
	Month            string  `json:"month"` // YYYY-MM
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	AvgSpeedMbps     float64 `json:"avg_speed_mbps"`
	DowntimeSeconds  float64 `json:"downtime_seconds"`
	AvailabilityPct  float64 `json:"availability_pct"`
	OutageCount      int     `json:"outage_count"`
	LongestOutageSec float64 `json:"longest_outage_seconds"`
	AvgOutageSec     float64 `json:"avg_outage_seconds"`
	CheckCount       int64   `json:"check_count"`
	LatencySum       float64 `json:"latency_sum"`
	SpeedTestCount   int64   `json:"speed_test_count"`
	SpeedSum         float64 `json:"speed_sum"`
}

// Settings holds runtime-configurable application settings persisted in the DB.
type Settings struct {
	TelegramEnabled        bool    `json:"telegram_enabled"`
	TelegramBotToken       string  `json:"telegram_bot_token,omitempty"`
	TelegramChatID         string  `json:"telegram_chat_id,omitempty"`
	MonitorIntervalSec     int     `json:"monitor_interval_sec"`
	SpeedIntervalMinutes   int     `json:"speed_interval_minutes"`
	SpeedReportEnabled     bool    `json:"speed_report_enabled"`
	RetentionDays          int     `json:"retention_days"`
	BatteryLowThresholdPct float64 `json:"battery_low_threshold_pct"`
}

// NotificationType enumerates the kinds of Telegram notifications sent.
type NotificationType string

const (
	NotificationDown          NotificationType = "internet_down"
	NotificationRestored      NotificationType = "internet_restored"
	NotificationDailySummary  NotificationType = "daily_summary"
	NotificationWeeklySummary NotificationType = "weekly_summary"
	NotificationTest          NotificationType = "test"
	NotificationServiceUp     NotificationType = "service_started"
	NotificationServiceDown   NotificationType = "service_stopped"
	NotificationBatteryLow    NotificationType = "battery_low"
	NotificationBatteryOk     NotificationType = "battery_restored"
	NotificationHighLatency   NotificationType = "high_latency"
	NotificationLatencyNormal NotificationType = "latency_normal"
	NotificationSlowSpeed     NotificationType = "slow_speed"
	NotificationSpeedReport   NotificationType = "speed_report"
	NotificationHourlyLatency NotificationType = "hourly_latency"
)

// Notification records a sent (or attempted) Telegram notification for audit purposes.
type Notification struct {
	ID        string           `json:"id"`
	Timestamp time.Time        `json:"timestamp"`
	Type      NotificationType `json:"type"`
	Message   string           `json:"message"`
	Success   bool             `json:"success"`
	Error     string           `json:"error,omitempty"`
}

// CurrentStatus is the live snapshot returned by GET /api/status.
type CurrentStatus struct {
	Status               ConnectivityStatus `json:"status"`
	LastCheck            time.Time          `json:"last_check"`
	LatencyMs            float64            `json:"latency_ms"`
	ISP                  string             `json:"isp"`
	CurrentUptimeStart   *time.Time         `json:"current_uptime_start,omitempty"`
	CurrentStreakSec     float64            `json:"current_streak_seconds"`
	TodayDowntimeSec     float64            `json:"today_downtime_seconds"`
	MonthAvailabilityPct float64            `json:"month_availability_pct"`
	OngoingOutage        *Outage            `json:"ongoing_outage,omitempty"`
}

// SystemMetrics holds CPU, Memory, Disk, CPU Temperature, and Battery measurements.
type SystemMetrics struct {
	ID              string    `json:"id"`
	Timestamp       time.Time `json:"timestamp"`
	CPUPercent      float64   `json:"cpu_percent"`
	RAMUsedMB       float64   `json:"ram_used_mb"`
	RAMTotalMB      float64   `json:"ram_total_mb"`
	RAMPercent      float64   `json:"ram_percent"`
	DiskUsedGB      float64   `json:"disk_used_gb"`
	DiskTotalGB     float64   `json:"disk_total_gb"`
	DiskPercent     float64   `json:"disk_percent"`
	CPUTempC        float64   `json:"cpu_temp_c,omitempty"`
	BatteryPresent  bool      `json:"battery_present"`
	BatteryPercent  float64   `json:"battery_percent"`
	BatteryCharging bool      `json:"battery_charging"`
}

