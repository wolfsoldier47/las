package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ulas-service/internal/handler"
	"ulas-service/internal/token"
)

// corsMiddleware allows browser clients served from a different origin to call the API.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// New builds the Gin HTTP router with all application routes.
func New(
	auth *handler.AuthHandler,
	tokenMaker token.Maker,
	health *handler.HealthHandler,
	hosts *handler.HostHandler,
	baselines *handler.BaselineHandler,
	deviations *handler.DeviationHandler,
	scans *handler.ScanHandler,
	reports *handler.ReportHandler,
	incidents *handler.IncidentHandler,
	snapshots *handler.SnapshotHandler,
	scanSchedules *handler.ScanScheduleHandler,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), corsMiddleware())

	// Public API routes (no authentication required).
	r.POST("/api/login", auth.Login)
	r.GET("/api/health", health.HandleHealth)
	r.GET("/api/health/aap", health.HandleAAPHealth)

	// AAP callback endpoint must remain unauthenticated.
	r.POST("/api/callbacks/scan", scans.ScanCallback)

	// OS versions are public so the upload form can populate its dropdown before login.
	r.GET("/api/os-versions", baselines.ListOSVersions)

	// Protected API routes — any authenticated user (read or admin) can view.
	api := r.Group("/api")
	api.Use(handler.AuthMiddleware(tokenMaker))
	{
		api.GET("/me", auth.GetCurrentUser)

		api.GET("/hosts", hosts.ListHosts)
		api.GET("/hosts/:id", hosts.GetHost)

		api.GET("/baselines", baselines.ListBaselines)
		api.GET("/baselines/:id", baselines.GetBaseline)
		api.GET("/baselines/versions", baselines.ListBaselineVersions)

		api.GET("/deviations", deviations.ListDeviations)
		api.GET("/deviations/:id", deviations.GetDeviation)

		api.GET("/scans", scans.ListScans)
		api.GET("/scans/:id", scans.GetScan)
		api.GET("/scans/:id/hosts/:hostId", scans.GetHostResult)
		api.GET("/scans/:id/report", reports.DownloadScanReport)

		api.GET("/incidents", incidents.ListIncidents)
		api.GET("/incidents/:id", incidents.GetIncident)

		api.GET("/snapshots/:hostId/:fileType/history", snapshots.GetHistory)
		api.GET("/snapshots/:hostId/:fileType/changes", snapshots.GetChanges)
		api.GET("/snapshots/detail/:id", snapshots.GetSnapshot)

		api.GET("/scan-schedules", scanSchedules.ListScanSchedules)
		api.GET("/scan-schedules/:id", scanSchedules.GetScanSchedule)
		api.GET("/scan-schedules/:id/runs", scanSchedules.ListScanScheduleRuns)
	}

	// Admin-only routes — everything that mutates state or triggers actions.
	admin := api.Group("").Use(handler.AdminMiddleware())
	{
		admin.POST("/hosts", hosts.CreateHost)
		admin.PUT("/hosts/:id", hosts.UpdateHost)
		admin.DELETE("/hosts/:id", hosts.DeleteHost)

		admin.POST("/baselines", baselines.CreateBaseline)
		admin.PUT("/baselines/:id", baselines.UpdateBaseline)
		admin.DELETE("/baselines/:id", baselines.DeleteBaseline)
		admin.POST("/baselines/upload", baselines.UploadMasterFile)
		admin.POST("/baselines/versions/activate", baselines.ActivateBaselineVersion)
		admin.POST("/baselines/versions/deactivate", baselines.DeactivateBaselineScope)

		admin.POST("/deviations", deviations.CreateDeviation)
		admin.PUT("/deviations/:id", deviations.UpdateDeviation)
		admin.DELETE("/deviations/:id", deviations.DeleteDeviation)

		admin.POST("/scans", scans.InitiateScan)

		admin.POST("/incidents/:id/servicenow", incidents.OpenServiceNowTicket)
		admin.POST("/incidents/bulk-servicenow", incidents.BulkOpenServiceNowTickets)
		admin.PUT("/incidents/:id/status", incidents.UpdateIncidentStatus)

		admin.POST("/scan-schedules", scanSchedules.CreateScanSchedule)
		admin.PUT("/scan-schedules/:id", scanSchedules.UpdateScanSchedule)
		admin.DELETE("/scan-schedules/:id", scanSchedules.DeleteScanSchedule)
	}

	return r
}
