package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *handler) getMetricsLatest(c *gin.Context) {
	if h.deps.SysMetricsSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "metrics service not enabled"})
		return
	}

	metrics, err := h.deps.SysMetricsSvc.Latest(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *handler) getMetricsHistory(c *gin.Context) {
	if h.deps.SysMetricsSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "metrics service not enabled"})
		return
	}

	fromStr := c.Query("from")
	toStr := c.Query("to")
	limitStr := c.Query("limit")

	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	limit := 100

	if fromStr != "" {
		if parsed, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = parsed
		}
	}
	if toStr != "" {
		if parsed, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = parsed
		}
	}
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	history, err := h.deps.SysMetricsSvc.History(c.Request.Context(), from, to, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}
