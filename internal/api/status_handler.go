package api

import "github.com/gin-gonic/gin"

// getStatus handles GET /api/status.
func (h *handler) getStatus(c *gin.Context) {
	status, err := h.deps.MonitorSvc.GetCurrentStatus(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(200, status)
}
