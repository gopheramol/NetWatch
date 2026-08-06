package api

import (
	"time"

	"github.com/gin-gonic/gin"
)

const (
	dateFormat  = "2006-01-02"
	monthFormat = "2006-01"
)

// getAnalyticsDaily handles GET /api/analytics/daily.
// With ?date=YYYY-MM-DD it returns a single day; otherwise it returns a
// range bounded by ?from=&to= (YYYY-MM-DD), defaulting to the last 30 days.
func (h *handler) getAnalyticsDaily(c *gin.Context) {
	ctx := c.Request.Context()

	if date := c.Query("date"); date != "" {
		stats, err := h.deps.Analytics.GetDaily(ctx, date)
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(200, stats)
		return
	}

	now := time.Now()
	from, to := now.AddDate(0, 0, -30), now

	if raw := c.Query("from"); raw != "" {
		parsed, err := time.Parse(dateFormat, raw)
		if err != nil {
			badRequest(c, err)
			return
		}
		from = parsed
	}
	if raw := c.Query("to"); raw != "" {
		parsed, err := time.Parse(dateFormat, raw)
		if err != nil {
			badRequest(c, err)
			return
		}
		to = parsed
	}

	stats, err := h.deps.Analytics.GetDailyRange(ctx, from, to)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(200, gin.H{"from": from.Format(dateFormat), "to": to.Format(dateFormat), "count": len(stats), "data": stats})
}

// getAnalyticsMonthly handles GET /api/analytics/monthly.
// With ?month=YYYY-MM it returns a single month; otherwise it returns a
// range bounded by ?from=&to= (YYYY-MM), defaulting to the last 12 months.
func (h *handler) getAnalyticsMonthly(c *gin.Context) {
	ctx := c.Request.Context()

	if month := c.Query("month"); month != "" {
		stats, err := h.deps.Analytics.GetMonthly(ctx, month)
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(200, stats)
		return
	}

	now := time.Now()
	from, to := now.AddDate(0, -12, 0), now

	if raw := c.Query("from"); raw != "" {
		parsed, err := time.Parse(monthFormat, raw)
		if err != nil {
			badRequest(c, err)
			return
		}
		from = parsed
	}
	if raw := c.Query("to"); raw != "" {
		parsed, err := time.Parse(monthFormat, raw)
		if err != nil {
			badRequest(c, err)
			return
		}
		to = parsed
	}

	stats, err := h.deps.Analytics.GetMonthlyRange(ctx, from, to)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(200, gin.H{"from": from.Format(monthFormat), "to": to.Format(monthFormat), "count": len(stats), "data": stats})
}
