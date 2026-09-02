package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"ulas-service/models"
)

// AAPHealthChecker abstracts the AAP health check operation.
type AAPHealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// HealthHandler exposes liveness and readiness probes.
type HealthHandler struct {
	db            DBPinger
	aap           AAPHealthChecker
	aapSolaris    AAPHealthChecker
}

// DBPinger abstracts the database ping operation.
type DBPinger interface {
	Ping() error
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(db DBPinger, aap, aapSolaris AAPHealthChecker) *HealthHandler {
	return &HealthHandler{db: db, aap: aap, aapSolaris: aapSolaris}
}

// HealthResponse is the shape returned by the health endpoint.
type HealthResponse struct {
	Status   string `json:"status"`
	DBStatus string `json:"db_status"`
	Version  string `json:"version"`
}

// HandleHealth responds with the current service health.
func (h *HealthHandler) HandleHealth(c *gin.Context) {
	resp := HealthResponse{
		Status:   "ok",
		DBStatus: "ok",
		Version:  "0.1.0",
	}

	if err := h.db.Ping(); err != nil {
		resp.Status = "degraded"
		resp.DBStatus = "unreachable"
		c.JSON(http.StatusServiceUnavailable, resp)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// AAPHealthResponse is the shape returned by the AAP health endpoint.
type AAPHealthResponse struct {
	Status    string `json:"status"`
	AAPStatus string `json:"aap_status"`
	Error     string `json:"error,omitempty"`
}

// HandleAAPHealth responds with the current AAP connectivity status.
// Pass ?os_type=linux|solaris to check a specific instance; defaults to linux.
func (h *HealthHandler) HandleAAPHealth(c *gin.Context) {
	osType := models.OSType(c.Query("os_type"))
	if osType == "" {
		osType = models.OSTypeLinux
	}

	var checker AAPHealthChecker
	switch osType {
	case models.OSTypeSolaris:
		checker = h.aapSolaris
	default:
		checker = h.aap
	}

	if checker == nil {
		c.JSON(http.StatusServiceUnavailable, AAPHealthResponse{
			Status:    "degraded",
			AAPStatus: "not_configured",
		})
		return
	}

	if err := checker.HealthCheck(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, AAPHealthResponse{
			Status:    "degraded",
			AAPStatus: "unreachable",
			Error:     err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AAPHealthResponse{
		Status:    "ok",
		AAPStatus: "ok",
	})
}
