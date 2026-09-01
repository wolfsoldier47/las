package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ulas-service/internal/repository"
	"ulas-service/internal/service"
)

// HostHandler handles HTTP requests for hosts.
type HostHandler struct {
	service service.HostService
}

// NewHostHandler creates a new host handler.
func NewHostHandler(service service.HostService) *HostHandler {
	return &HostHandler{service: service}
}

// CreateHost handles POST /api/hosts.
func (h *HostHandler) CreateHost(c *gin.Context) {
	var req service.CreateHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	host, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, host)
}

// GetHost handles GET /api/hosts/:id.
func (h *HostHandler) GetHost(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host id"})
		return
	}

	host, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrHostNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, host)
}

// ListHosts handles GET /api/hosts.
// Supports optional ?page=&limit=&search= query parameters for pagination and search.
func (h *HostHandler) ListHosts(c *gin.Context) {
	ctx := c.Request.Context()
	page, limit, hasPagination := ParsePagination(c)

	filters := repository.HostFilters{}
	if search := c.Query("search"); search != "" {
		filters.Search = &search
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

	hosts, err := h.service.List(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, hosts)
}

// UpdateHost handles PUT /api/hosts/:id.
func (h *HostHandler) UpdateHost(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host id"})
		return
	}

	var req service.UpdateHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	host, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrHostNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, host)
}

// DeleteHost handles DELETE /api/hosts/:id.
func (h *HostHandler) DeleteHost(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host id"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, repository.ErrHostNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
