package telegram

import (
	"fmt"
	"time"

	"github.com/gopheramol/NetWatch/internal/models"
)

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", h, m)
}

func downMessage(reason string, since time.Time) string {
	return fmt.Sprintf(
		"🔴 <b>Internet Down</b>\n\n"+
			"⏰ Since: %s\n"+
			"❗ Reason: %s\n\n"+
			"Monitoring will notify you when the connection is restored.",
		since.Format("2006-01-02 15:04:05"),
		reason,
	)
}

func restoredMessage(outage models.Outage) string {
	end := ""
	if outage.EndTime != nil {
		end = outage.EndTime.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf(
		"🟢 <b>Internet Restored</b>\n\n"+
			"⏰ Down since: %s\n"+
			"✅ Restored at: %s\n"+
			"⏱ Duration: %s\n"+
			"❗ Reason: %s",
		outage.StartTime.Format("2006-01-02 15:04:05"),
		end,
		formatDuration(outage.Duration),
		outage.Reason,
	)
}

func dailySummaryMessage(stats models.DailyStats) string {
	return fmt.Sprintf(
		"📊 <b>Daily Summary — %s</b>\n\n"+
			"📶 Availability: %.2f%%\n"+
			"⚡ Avg latency: %.1f ms\n"+
			"🚀 Avg speed: %.1f Mbps (min %.1f / max %.1f)\n"+
			"🔻 Downtime: %s\n"+
			"🔁 Outages: %d",
		stats.Date,
		stats.AvailabilityPct,
		stats.AvgLatencyMs,
		stats.AvgSpeedMbps, stats.MinSpeedMbps, stats.MaxSpeedMbps,
		formatDuration(time.Duration(stats.DowntimeSeconds)*time.Second),
		stats.OutageCount,
	)
}

func weeklySummaryMessage(days []models.DailyStats) string {
	var totalDowntime float64
	var latencySum, speedSum float64
	var outageCount int
	var validSpeedDays, validLatencyDays int

	for _, d := range days {
		totalDowntime += d.DowntimeSeconds
		outageCount += d.OutageCount
		if d.CheckCount > 0 {
			latencySum += d.AvgLatencyMs
			validLatencyDays++
		}
		if d.SpeedTestCount > 0 {
			speedSum += d.AvgSpeedMbps
			validSpeedDays++
		}
	}

	avgLatency := 0.0
	if validLatencyDays > 0 {
		avgLatency = latencySum / float64(validLatencyDays)
	}
	avgSpeed := 0.0
	if validSpeedDays > 0 {
		avgSpeed = speedSum / float64(validSpeedDays)
	}
	availability := 100.0
	totalSeconds := float64(len(days)) * 24 * 3600
	if totalSeconds > 0 {
		availability = 100 * (1 - totalDowntime/totalSeconds)
	}

	return fmt.Sprintf(
		"📈 <b>Weekly Summary</b>\n\n"+
			"📶 Availability: %.2f%%\n"+
			"⚡ Avg latency: %.1f ms\n"+
			"🚀 Avg speed: %.1f Mbps\n"+
			"🔻 Total downtime: %s\n"+
			"🔁 Total outages: %d",
		availability,
		avgLatency,
		avgSpeed,
		formatDuration(time.Duration(totalDowntime)*time.Second),
		outageCount,
	)
}

func testMessage() string {
	return "✅ <b>Test Notification</b>\n\nYour NetWatch Telegram integration is working correctly. 🎉"
}
