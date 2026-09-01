package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ulas-service/internal/repository"
	"ulas-service/internal/service"
	"ulas-service/models"
)

// BaselineHandler handles HTTP requests for master baselines.
type BaselineHandler struct {
	service service.BaselineService
}

// NewBaselineHandler creates a new baseline handler.
func NewBaselineHandler(service service.BaselineService) *BaselineHandler {
	return &BaselineHandler{service: service}
}

// CreateBaseline handles POST /api/baselines.
func (h *BaselineHandler) CreateBaseline(c *gin.Context) {
	var req service.CreateBaselineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.CreatedBy = c.GetString("username")

	baseline, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, baseline)
}

// GetBaseline handles GET /api/baselines/:id.
func (h *BaselineHandler) GetBaseline(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid baseline id"})
		return
	}

	baseline, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrBaselineNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "baseline not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, baseline)
}

// ListBaselines handles GET /api/baselines.
func (h *BaselineHandler) ListBaselines(c *gin.Context) {
	filters := repository.BaselineFilters{}

	if osType := c.Query("os_type"); osType != "" {
		ost := models.OSType(osType)
		filters.OSType = &ost
	}
	if fileType := c.Query("file_type"); fileType != "" {
		ft := models.FileType(fileType)
		filters.FileType = &ft
	}
	if versionStr := c.Query("version"); versionStr != "" {
		if version, err := strconv.Atoi(versionStr); err == nil {
			filters.Version = &version
		}
	}
	if activeStr := c.Query("active"); activeStr == "true" {
		active := true
		filters.IsActive = &active
	} else if activeStr == "false" {
		active := false
		filters.IsActive = &active
	}

	baselines, err := h.service.List(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, baselines)
}

// UpdateBaseline handles PUT /api/baselines/:id.
func (h *BaselineHandler) UpdateBaseline(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid baseline id"})
		return
	}

	var req service.UpdateBaselineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	baseline, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrBaselineNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "baseline not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, baseline)
}

// DeleteBaseline handles DELETE /api/baselines/:id.
func (h *BaselineHandler) DeleteBaseline(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid baseline id"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, repository.ErrBaselineNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "baseline not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// UploadMasterFile handles POST /api/baselines/upload.
func (h *BaselineHandler) UploadMasterFile(c *gin.Context) {
	var req service.UploadMasterFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.CreatedBy = c.GetString("username")

	version, err := h.service.UploadMasterFile(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"version": version})
}

// ListBaselineVersions handles GET /api/baselines/versions.
// Supports optional ?page=&limit= query parameters for pagination.
func (h *BaselineHandler) ListBaselineVersions(c *gin.Context) {
	ctx := c.Request.Context()
	page, limit, hasPagination := ParsePagination(c)

	if hasPagination {
		paginated, err := h.service.ListVersionsPaginated(ctx, page, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, paginated)
		return
	}

	versions, err := h.service.ListVersions(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, versions)
}

// scopeActionRequest identifies a versioned master file scope.
type scopeActionRequest struct {
	OSType   models.OSType   `json:"os_type" binding:"required"`
	FileType models.FileType `json:"file_type" binding:"required"`
	Version  int             `json:"version" binding:"required"`
}

// ActivateBaselineVersion handles POST /api/baselines/versions/activate.
func (h *BaselineHandler) ActivateBaselineVersion(c *gin.Context) {
	var req scopeActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.ActivateVersion(c.Request.Context(), req.OSType, req.FileType, req.Version); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// DeactivateBaselineScope handles POST /api/baselines/versions/deactivate.
func (h *BaselineHandler) DeactivateBaselineScope(c *gin.Context) {
	var req scopeActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.DeactivateScope(c.Request.Context(), req.OSType, req.FileType, req.Version); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListOSVersions handles GET /api/os-versions.
func (h *BaselineHandler) ListOSVersions(c *gin.Context) {
	// The list is returned as a map of os_type -> []major_version.
	c.JSON(http.StatusOK, h.service.OSVersions())
}
