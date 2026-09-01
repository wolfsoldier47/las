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

// DeviationHandler handles HTTP requests for allowed deviations.
type DeviationHandler struct {
	service service.DeviationService
}

// NewDeviationHandler creates a new deviation handler.
func NewDeviationHandler(service service.DeviationService) *DeviationHandler {
	return &DeviationHandler{service: service}
}

// CreateDeviation handles POST /api/deviations.
func (h *DeviationHandler) CreateDeviation(c *gin.Context) {
	var req service.CreateDeviationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deviation, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateDeviation) {
			c.JSON(http.StatusConflict, gin.H{"error": "a deviation already exists for this host, file, and key"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, deviation)
}

// GetDeviation handles GET /api/deviations/:id.
func (h *DeviationHandler) GetDeviation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deviation id"})
		return
	}

	deviation, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrDeviationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "deviation not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, deviation)
}

// ListDeviations handles GET /api/deviations.
// Supports optional ?page=&limit=&search= query parameters for pagination and search.
func (h *DeviationHandler) ListDeviations(c *gin.Context) {
	ctx := c.Request.Context()
	page, limit, hasPagination := ParsePagination(c)

	filters := repository.DeviationFilters{}
	if hostname := c.Query("hostname"); hostname != "" {
		filters.Hostname = &hostname
	}
	if search := c.Query("search"); search != "" {
		filters.Search = &search
	}
	if fileType := c.Query("file_type"); fileType != "" {
		ft := models.FileType(fileType)
		filters.FileType = &ft
	}
	if activeStr := c.Query("active"); activeStr != "" {
		active := activeStr == "true"
		filters.Active = &active
	}

	if hasPagination {
		paginated, err := h.service.ListPaginated(ctx, filters, page, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, paginated)
		return
	}

	deviations, err := h.service.List(ctx, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, deviations)
}

// UpdateDeviation handles PUT /api/deviations/:id.
func (h *DeviationHandler) UpdateDeviation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deviation id"})
		return
	}

	var req service.UpdateDeviationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deviation, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrDeviationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "deviation not found"})
			return
		}
		if errors.Is(err, repository.ErrDuplicateDeviation) {
			c.JSON(http.StatusConflict, gin.H{"error": "a deviation already exists for this host, file, and key"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, deviation)
}

// DeleteDeviation handles DELETE /api/deviations/:id.
func (h *DeviationHandler) DeleteDeviation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deviation id"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, repository.ErrDeviationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "deviation not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
