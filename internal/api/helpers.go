package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultLimit = 100
	maxLimit     = 5000
)

func respondError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{"error": err.Error()})
}

// queryTimeRange reads "from"/"to" RFC3339 query params, defaulting to the
// last defaultWindow up to now when absent.
func queryTimeRange(c *gin.Context, defaultWindow time.Duration) (from, to time.Time, err error) {
	to = time.Now()
	from = to.Add(-defaultWindow)

	if raw := c.Query("to"); raw != "" {
		to, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return from, to, err
		}
	}
	if raw := c.Query("from"); raw != "" {
		from, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return from, to, err
		}
	}
	return from, to, nil
}

func queryLimit(c *gin.Context) int {
	raw := c.Query("limit")
	if raw == "" {
		return defaultLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultLimit
	}
	if n > maxLimit {
		return maxLimit
	}
	return n
}

func badRequest(c *gin.Context, err error) {
	respondError(c, http.StatusBadRequest, err)
}

func internalError(c *gin.Context, err error) {
	respondError(c, http.StatusInternalServerError, err)
}
