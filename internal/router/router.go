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

	// Protected API routes.
	api := r.Group("/api").Use(handler.AuthMiddleware(tokenMaker))
	{
		api.GET("/me", auth.GetCurrentUser)

		api.POST("/hosts", hosts.CreateHost)
		api.GET("/hosts", hosts.ListHosts)
		api.GET("/hosts/:id", hosts.GetHost)
		api.PUT("/hosts/:id", hosts.UpdateHost)
		api.DELETE("/hosts/:id", hosts.DeleteHost)

		api.POST("/baselines", baselines.CreateBaseline)
		api.GET("/baselines", baselines.ListBaselines)
		api.GET("/baselines/:id", baselines.GetBaseline)
		api.PUT("/baselines/:id", baselines.UpdateBaseline)
		api.DELETE("/baselines/:id", baselines.DeleteBaseline)
		api.POST("/baselines/upload", baselines.UploadMasterFile)
		api.GET("/baselines/versions", baselines.ListBaselineVersions)
		api.POST("/baselines/versions/activate", baselines.ActivateBaselineVersion)
		api.POST("/baselines/versions/deactivate", baselines.DeactivateBaselineScope)

		api.POST("/deviations", deviations.CreateDeviation)
		api.GET("/deviations", deviations.ListDeviations)
		api.GET("/deviations/:id", deviations.GetDeviation)
		api.PUT("/deviations/:id", deviations.UpdateDeviation)
		api.DELETE("/deviations/:id", deviations.DeleteDeviation)

		api.GET("/scans", scans.ListScans)
		api.POST("/scans", scans.InitiateScan)
		api.GET("/scans/:id", scans.GetScan)
		api.GET("/scans/:id/hosts/:hostId", scans.GetHostResult)
		api.GET("/scans/:id/report", reports.DownloadScanReport)

		api.GET("/incidents", incidents.ListIncidents)
		api.GET("/incidents/:id", incidents.GetIncident)
		api.POST("/incidents/:id/servicenow", incidents.OpenServiceNowTicket)
		api.POST("/incidents/bulk-servicenow", incidents.BulkOpenServiceNowTickets)
		api.PUT("/incidents/:id/status", incidents.UpdateIncidentStatus)

		api.GET("/snapshots/:hostId/:fileType/history", snapshots.GetHistory)
		api.GET("/snapshots/:hostId/:fileType/changes", snapshots.GetChanges)
		api.GET("/snapshots/detail/:id", snapshots.GetSnapshot)

		api.GET("/scan-schedules", scanSchedules.ListScanSchedules)
		api.POST("/scan-schedules", scanSchedules.CreateScanSchedule)
		api.GET("/scan-schedules/:id", scanSchedules.GetScanSchedule)
		api.PUT("/scan-schedules/:id", scanSchedules.UpdateScanSchedule)
		api.DELETE("/scan-schedules/:id", scanSchedules.DeleteScanSchedule)
		api.GET("/scan-schedules/:id/runs", scanSchedules.ListScanScheduleRuns)
	}

	return r
}
