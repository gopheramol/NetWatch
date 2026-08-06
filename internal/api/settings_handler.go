package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gopheramol/NetWatch/internal/models"
)

// getSettings handles GET /api/settings.
func (h *handler) getSettings(c *gin.Context) {
	settings, err := h.deps.SettingsRepo.Get(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(200, settings)
}

// postSettings handles POST /api/settings, replacing the stored settings.
func (h *handler) postSettings(c *gin.Context) {
	var settings models.Settings
	if err := c.ShouldBindJSON(&settings); err != nil {
		badRequest(c, err)
		return
	}

	if settings.MonitorIntervalSec <= 0 {
		badRequest(c, fmt.Errorf("monitor_interval_sec must be positive"))
		return
	}
	if settings.SpeedIntervalMinutes <= 0 {
		badRequest(c, fmt.Errorf("speed_interval_minutes must be positive"))
		return
	}
	if settings.RetentionDays <= 0 {
		badRequest(c, fmt.Errorf("retention_days must be positive"))
		return
	}
	if settings.TelegramEnabled && (settings.TelegramBotToken == "" || settings.TelegramChatID == "") {
		badRequest(c, fmt.Errorf("telegram_bot_token and telegram_chat_id are required when telegram_enabled is true"))
		return
	}
	if settings.BatteryLowThresholdPct < 0 || settings.BatteryLowThresholdPct > 100 {
		badRequest(c, fmt.Errorf("battery_low_threshold_pct must be between 0 and 100"))
		return
	}

	if err := h.deps.SettingsRepo.Save(c.Request.Context(), &settings); err != nil {
		internalError(c, err)
		return
	}
	c.JSON(200, settings)
}
