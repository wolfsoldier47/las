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

// SnapshotHandler handles HTTP requests for snapshots.
type SnapshotHandler struct {
	service service.SnapshotService
}

// NewSnapshotHandler creates a new snapshot handler.
func NewSnapshotHandler(service service.SnapshotService) *SnapshotHandler {
	return &SnapshotHandler{service: service}
}

// GetHistory handles GET /api/snapshots/:hostId/:fileType/history.
func (h *SnapshotHandler) GetHistory(c *gin.Context) {
	hostID, err := uuid.Parse(c.Param("hostId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host id"})
		return
	}

	fileType := models.FileType(c.Param("fileType"))
	if fileType != models.FileTypePasswd && fileType != models.FileTypeGroup {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file type"})
		return
	}

	history, err := h.service.GetHistory(c.Request.Context(), hostID, fileType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

// GetChanges handles GET /api/snapshots/:hostId/:fileType/changes.
func (h *SnapshotHandler) GetChanges(c *gin.Context) {
	hostID, err := uuid.Parse(c.Param("hostId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host id"})
		return
	}

	fileType := models.FileType(c.Param("fileType"))
	if fileType != models.FileTypePasswd && fileType != models.FileTypeGroup {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file type"})
		return
	}

	changes, err := h.service.GetChanges(c.Request.Context(), hostID, fileType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, changes)
}

// GetSnapshot handles GET /api/snapshots/:id.
func (h *SnapshotHandler) GetSnapshot(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot id"})
		return
	}

	snapshot, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrSnapshotNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}
