// Package api exposes the application's functionality over a REST interface
// built with Gin. Handlers depend only on service/repository interfaces,
// never on bbolt or Telegram directly.
package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopheramol/NetWatch/internal/analytics"
	"github.com/gopheramol/NetWatch/internal/middleware"
	"github.com/gopheramol/NetWatch/internal/monitor"
	"github.com/gopheramol/NetWatch/internal/repository"
	"github.com/gopheramol/NetWatch/internal/services/speedtest"
	"github.com/gopheramol/NetWatch/internal/telegram"
	"go.uber.org/zap"
)

// Dependencies bundles everything the API layer needs to serve requests.
type Dependencies struct {
	MonitorSvc   *monitor.Service
	SpeedTestSvc *speedtest.Service
	Analytics    analytics.Engine
	Notifier     telegram.Notifier
	ConnRepo     repository.ConnectivityRepository
	SpeedRepo    repository.SpeedTestRepository
	DowntimeRepo repository.DowntimeRepository
	SettingsRepo repository.SettingsRepository
	Logger       *zap.Logger
	CORSOrigins  []string
}

type handler struct {
	deps Dependencies
}

// NewRouter builds the fully-wired Gin engine for the application.
func NewRouter(deps Dependencies) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(middleware.Recovery(deps.Logger))
	r.Use(middleware.Logging(deps.Logger))
	r.Use(middleware.CORS(deps.CORSOrigins))

	h := &handler{deps: deps}

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "time": time.Now()})
	})

	apiGroup := r.Group("/api")
	{
		apiGroup.GET("/status", h.getStatus)

		apiGroup.GET("/connectivity", h.getConnectivity)

		apiGroup.GET("/speed/latest", h.getSpeedLatest)
		apiGroup.GET("/speed/history", h.getSpeedHistory)
		apiGroup.POST("/speedtest", h.postSpeedTest)

		apiGroup.GET("/downtime", h.getDowntime)

		apiGroup.GET("/analytics/daily", h.getAnalyticsDaily)
		apiGroup.GET("/analytics/monthly", h.getAnalyticsMonthly)

		apiGroup.GET("/settings", h.getSettings)
		apiGroup.POST("/settings", h.postSettings)

		apiGroup.POST("/telegram/test", h.postTelegramTest)
	}

	return r
}
