package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ulas-service/internal/repository"
	"ulas-service/internal/service"
	"ulas-service/models"
)

// IncidentHandler handles HTTP requests for incidents.
type IncidentHandler struct {
	service     service.IncidentService
	snowService service.ServiceNowService
}

// NewIncidentHandler creates a new incident handler.
func NewIncidentHandler(service service.IncidentService, snowService service.ServiceNowService) *IncidentHandler {
	return &IncidentHandler{service: service, snowService: snowService}
}

// GetIncident handles GET /api/incidents/:id.
func (h *IncidentHandler) GetIncident(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid incident id"})
		return
	}

	incident, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrIncidentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "incident not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, incident)
}

// ListIncidents handles GET /api/incidents.
func (h *IncidentHandler) ListIncidents(c *gin.Context) {
	filters := repository.IncidentFilters{}

	if hostIDStr := c.Query("host_id"); hostIDStr != "" {
		if hostID, err := uuid.Parse(hostIDStr); err == nil {
			filters.HostID = &hostID
		}
	}
	if statusStr := c.Query("status"); statusStr != "" {
		status := models.IncidentStatus(statusStr)
		filters.Status = &status
	}
	if scanResultIDStr := c.Query("scan_result_id"); scanResultIDStr != "" {
		if scanResultID, err := uuid.Parse(scanResultIDStr); err == nil {
			filters.ScanResultID = &scanResultID
		}
	}

	incidents, err := h.service.List(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, incidents)
}

// UpdateIncidentRequest is the body for PUT /api/incidents/:id/status.
type UpdateIncidentRequest struct {
	Status     string  `json:"status" binding:"required"`
	Notes      *string `json:"notes"`
	Resolution *string `json:"resolution"`
}

// OpenServiceNowTicket handles POST /api/incidents/:id/servicenow.
func (h *IncidentHandler) OpenServiceNowTicket(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid incident id"})
		return
	}

	ticket, err := h.snowService.OpenTicket(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ticket)
}

// BulkOpenServiceNowTicketsRequest is the body for POST /api/incidents/bulk-servicenow.
type BulkOpenServiceNowTicketsRequest struct {
	IncidentIDs []string `json:"incident_ids" binding:"required"`
}

// BulkOpenServiceNowTickets handles POST /api/incidents/bulk-servicenow.
func (h *IncidentHandler) BulkOpenServiceNowTickets(c *gin.Context) {
	var req BulkOpenServiceNowTicketsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var ids []uuid.UUID
	for _, idStr := range req.IncidentIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid incident id: " + idStr})
			return
		}
		ids = append(ids, id)
	}

	tickets, err := h.snowService.BulkOpenTickets(c.Request.Context(), ids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tickets)
}

// UpdateIncidentStatus handles PUT /api/incidents/:id/status.
func (h *IncidentHandler) UpdateIncidentStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid incident id"})
		return
	}

	var req UpdateIncidentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	incident, err := h.service.UpdateStatus(c.Request.Context(), id, models.IncidentStatus(req.Status), req.Notes, req.Resolution)
	if err != nil {
		if errors.Is(err, repository.ErrIncidentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "incident not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, incident)
}
