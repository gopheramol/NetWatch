package api

import "github.com/gin-gonic/gin"

// postTelegramTest handles POST /api/telegram/test.
func (h *handler) postTelegramTest(c *gin.Context) {
	if err := h.deps.Notifier.NotifyTest(c.Request.Context()); err != nil {
		internalError(c, err)
		return
	}
	c.JSON(200, gin.H{"sent": true})
}
