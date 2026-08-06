package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopheramol/NetWatch/internal/database"
)

// getSpeedLatest handles GET /api/speed/latest.
func (h *handler) getSpeedLatest(c *gin.Context) {
	result, err := h.deps.SpeedRepo.Latest(c.Request.Context())
	if err != nil {
		if err == database.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "no speed test results yet"})
			return
		}
		internalError(c, err)
		return
	}
	c.JSON(200, result)
}

// getSpeedHistory handles GET /api/speed/history?from=&to=&limit=
func (h *handler) getSpeedHistory(c *gin.Context) {
	from, to, err := queryTimeRange(c, 30*24*time.Hour)
	if err != nil {
		badRequest(c, err)
		return
	}
	limit := queryLimit(c)

	results, err := h.deps.SpeedRepo.Range(c.Request.Context(), from, to, limit)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(200, gin.H{"from": from, "to": to, "count": len(results), "data": results})
}

// postSpeedTest handles POST /api/speedtest, triggering an immediate speed test.
func (h *handler) postSpeedTest(c *gin.Context) {
	result, err := h.deps.SpeedTestSvc.Run(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(200, result)
}
