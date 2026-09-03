package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ulas-service/internal/repository"
	"ulas-service/internal/service"
	"ulas-service/models"
)

// ScanHandler handles HTTP requests for scans and callbacks.
type ScanHandler struct {
	scanService service.ScanService
}

// NewScanHandler creates a new scan handler.
func NewScanHandler(scanService service.ScanService) *ScanHandler {
	return &ScanHandler{scanService: scanService}
}

// InitiateScanRequest is the body for POST /api/scans.
// The job template name is read from the backend configuration.
type InitiateScanRequest struct {
	Limit  string        `json:"limit"`
	OSType models.OSType `json:"os_type"`
}

// ListScans handles GET /api/scans.
// Supports optional ?page=&limit= query parameters for pagination,
// ?search= for free-text search, and ?from_date=/&to_date= for date range filtering.
// Dates must be provided in RFC3339 format.
func (h *ScanHandler) ListScans(c *gin.Context) {
	ctx := c.Request.Context()
	page, limit, hasPagination := ParsePagination(c)

	if hasPagination {
		onlyWithDeviations := c.Query("has_deviations") == "true"
		search := c.Query("search")
		fromDate, toDate, err := parseScanDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		paginated, err := h.scanService.ListScanJobsPaginated(ctx, page, limit, onlyWithDeviations, search, fromDate, toDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, paginated)
		return
	}

	jobs, err := h.scanService.ListScanJobs(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, jobs)
}

// GetScan handles GET /api/scans/:id.
// Supports optional ?page=&limit= query parameters to paginate host results.
// Pass ?include_incidents=true to include incidents in the response; otherwise
// host results do not include incidents. Use /api/scans/:id/hosts/:hostId to
// load incidents for a specific host on demand.
func (h *ScanHandler) GetScan(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scan id"})
		return
	}

	ctx := c.Request.Context()
	page, limit, hasPagination := ParsePagination(c)
	includeIncidents := c.Query("include_incidents") == "true"

	var detail interface{}
	if hasPagination {
		detail, err = h.scanService.GetScanDetailPaginated(ctx, id, page, limit, includeIncidents)
	} else {
		detail, err = h.scanService.GetScanDetail(ctx, id, includeIncidents)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// GetHostResult handles GET /api/scans/:id/hosts/:hostId.
// Returns the full details for a single host within a scan job, including incidents.
func (h *ScanHandler) GetHostResult(c *gin.Context) {
	scanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scan id"})
		return
	}

	hostID, err := uuid.Parse(c.Param("hostId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host id"})
		return
	}

	ctx := c.Request.Context()
	result, err := h.scanService.GetHostResult(ctx, scanID, hostID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// parsePagination reads page and limit query parameters.
func ParsePagination(c *gin.Context) (page, limit int, ok bool) {
	pageStr := c.Query("page")
	limitStr := c.Query("limit")
	if pageStr == "" && limitStr == "" {
		return 1, 10, false
	}

	page = 1
	limit = 10

	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	return page, limit, true
}

// parseScanDateRange reads optional from_date and to_date query parameters in RFC3339 format.
func parseScanDateRange(c *gin.Context) (fromDate, toDate *time.Time, err error) {
	if v := c.Query("from_date"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid from_date: %w", err)
		}
		fromDate = &t
	}
	if v := c.Query("to_date"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid to_date: %w", err)
		}
		toDate = &t
	}
	return fromDate, toDate, nil
}

// InitiateScan handles POST /api/scans.
func (h *ScanHandler) InitiateScan(c *gin.Context) {
	var req InitiateScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	initiatedBy := c.GetString("username") // placeholder for auth
	if initiatedBy == "" {
		initiatedBy = "unknown"
	}

	osType := req.OSType
	if osType == "" {
		osType = models.OSTypeLinux
	}

	job, err := h.scanService.InitiateScan(c.Request.Context(), req.Limit, initiatedBy, osType)
	if err != nil {
		if err == service.ErrScanAlreadyRunning {
			c.JSON(http.StatusConflict, gin.H{"error": "a scan is already running"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, job)
}

// ScanCallback handles POST /api/callbacks/scan.
// The callback body may be:
//   - a new envelope: { "ansible_job_id": 123, "hosts": [ ... ] }
//   - an array of host payloads (legacy)
//   - a single host payload (legacy)
func (h *ScanHandler) ScanCallback(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer c.Request.Body.Close()

	// Receipt log: confirms the callback reached the handler at all. Every
	// other log line ([ENVELOPE], callback errors) only appears after this.
	slog.Info("scan callback received", "bytes", len(body))

	// Try the new envelope format first.
	// A final summary callback may contain only ansible_job_id + failed_hosts with no hosts.
	var envelope models.CallbackEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.AnsibleJobID != nil {
		jobID := normalizeAnsibleJobID(envelope.AnsibleJobID)
		if jobID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ansible_job_id"})
			return
		}

		// Process the envelope asynchronously with a worker pool so large inventories
		// (50k+ hosts) do not time out the HTTP request.
		if err := h.scanService.ProcessCallbackEnvelope(c.Request.Context(), jobID, envelope.Hosts, envelope.FailedHosts); err != nil {
			if errors.Is(err, repository.ErrScanJobNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "no scan job found for ansible_job_id " + jobID})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusAccepted)
		return
	}

	// Legacy: array of host payloads.
	var payloads []models.CallbackPayload
	if err := json.Unmarshal(body, &payloads); err != nil {
		// Fallback to a single host payload.
		var single models.CallbackPayload
		if err := json.Unmarshal(body, &single); err != nil {
			slog.Warn("scan callback rejected: body is neither an envelope, an array, nor a host payload",
				"error", err,
				"body", truncateForLog(string(body), 256),
			)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		payloads = []models.CallbackPayload{single}
	}

	if len(payloads) == 0 {
		c.Status(http.StatusAccepted)
		return
	}

	// Group legacy payloads by job_id and dispatch each group through the
	// asynchronous envelope processor. This keeps the HTTP response fast even
	// when the DB is the bottleneck.
	groups := make(map[string][]models.CallbackPayload)
	for _, payload := range payloads {
		if payload.JobID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing job_id in callback payload"})
			return
		}
		groups[payload.JobID] = append(groups[payload.JobID], payload)
	}

	for jobID, hosts := range groups {
		if err := h.scanService.ProcessCallbackEnvelope(c.Request.Context(), jobID, hosts, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.Status(http.StatusAccepted)
}

// truncateForLog shortens a string for safe inclusion in log output.
func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// normalizeAnsibleJobID converts an int or string ansible_job_id to a string.
func normalizeAnsibleJobID(v interface{}) string {
	switch id := v.(type) {
	case string:
		return id
	case float64:
		return fmt.Sprintf("%.0f", id)
	case int:
		return fmt.Sprintf("%d", id)
	case int64:
		return fmt.Sprintf("%d", id)
	case json.Number:
		return id.String()
	default:
		return fmt.Sprintf("%v", id)
	}
}
