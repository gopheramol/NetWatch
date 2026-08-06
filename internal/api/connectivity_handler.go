package api

import (
	"time"

	"github.com/gin-gonic/gin"
)

// getConnectivity handles GET /api/connectivity?from=&to=&limit=
func (h *handler) getConnectivity(c *gin.Context) {
	from, to, err := queryTimeRange(c, 24*time.Hour)
	if err != nil {
		badRequest(c, err)
		return
	}
	limit := queryLimit(c)

	checks, err := h.deps.ConnRepo.Range(c.Request.Context(), from, to, limit)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(200, gin.H{"from": from, "to": to, "count": len(checks), "data": checks})
}
