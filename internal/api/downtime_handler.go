package api

import "github.com/gin-gonic/gin"

// getDowntime handles GET /api/downtime?limit=
func (h *handler) getDowntime(c *gin.Context) {
	limit := queryLimit(c)

	outages, err := h.deps.DowntimeRepo.List(c.Request.Context(), limit)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(200, gin.H{"count": len(outages), "data": outages})
}
