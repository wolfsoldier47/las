package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ulas-service/internal/repository"
	"ulas-service/internal/service"
)

// ScanScheduleHandler handles HTTP requests for scan schedules.
type ScanScheduleHandler struct {
	service service.ScanScheduleService
}

// NewScanScheduleHandler creates a new scan schedule handler.
func NewScanScheduleHandler(service service.ScanScheduleService) *ScanScheduleHandler {
	return &ScanScheduleHandler{service: service}
}

// ListScanSchedules handles GET /api/scan-schedules.
func (h *ScanScheduleHandler) ListScanSchedules(c *gin.Context) {
	schedules, err := h.service.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, schedules)
}

// CreateScanSchedule handles POST /api/scan-schedules.
func (h *ScanScheduleHandler) CreateScanSchedule(c *gin.Context) {
	var req service.CreateScanScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.CreatedBy = c.GetString("username")

	schedule, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, schedule)
}

// GetScanSchedule handles GET /api/scan-schedules/:id.
func (h *ScanScheduleHandler) GetScanSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
		return
	}

	schedule, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrScanScheduleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, schedule)
}

// UpdateScanSchedule handles PUT /api/scan-schedules/:id.
func (h *ScanScheduleHandler) UpdateScanSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
		return
	}

	var req service.UpdateScanScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	schedule, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrScanScheduleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, schedule)
}

// ListScanScheduleRuns handles GET /api/scan-schedules/:id/runs.
func (h *ScanScheduleHandler) ListScanScheduleRuns(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
		return
	}

	runs, err := h.service.ListRuns(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, runs)
}

// DeleteScanSchedule handles DELETE /api/scan-schedules/:id.
func (h *ScanScheduleHandler) DeleteScanSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, repository.ErrScanScheduleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
