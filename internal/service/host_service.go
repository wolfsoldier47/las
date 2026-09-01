package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"ulas-service/internal/repository"
	"ulas-service/models"
)

// HostService defines business operations for hosts.
type HostService interface {
	Create(ctx context.Context, req CreateHostRequest) (*models.Host, error)
	Get(ctx context.Context, id uuid.UUID) (*models.Host, error)
	List(ctx context.Context) ([]models.Host, error)
	ListPaginated(ctx context.Context, filters repository.HostFilters, page, limit int) (*PaginatedHosts, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateHostRequest) (*models.Host, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// PaginatedHosts is a page of hosts.
type PaginatedHosts struct {
	Items []models.Host `json:"items"`
	Total int           `json:"total"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
}

// CreateHostRequest is the input for creating a host.
type CreateHostRequest struct {
	Hostname    string        `json:"hostname" binding:"required"`
	OSType      models.OSType `json:"os_type" binding:"required"`
	OSName      string        `json:"os_name"`
	OSVersion   string        `json:"os_version"`
	Environment string        `json:"environment"`
	Datacenter  string        `json:"datacenter"`
	Description string        `json:"description"`
}

// UpdateHostRequest is the input for updating a host.
type UpdateHostRequest struct {
	Hostname    string        `json:"hostname" binding:"required"`
	OSType      models.OSType `json:"os_type" binding:"required"`
	OSName      string        `json:"os_name"`
	OSVersion   string        `json:"os_version"`
	Environment string        `json:"environment"`
	Datacenter  string        `json:"datacenter"`
	Description string        `json:"description"`
}

// DefaultHostService is the default implementation of HostService.
type DefaultHostService struct {
	repo repository.HostRepository
}

// NewDefaultHostService creates a new host service.
func NewDefaultHostService(repo repository.HostRepository) *DefaultHostService {
	return &DefaultHostService{repo: repo}
}

// Create registers a new host.
func (s *DefaultHostService) Create(ctx context.Context, req CreateHostRequest) (*models.Host, error) {
	now := time.Now().UTC()
	host := &models.Host{
		ID:          uuid.New(),
		Hostname:    req.Hostname,
		OSType:      req.OSType,
		OSName:      req.OSName,
		OSVersion:   req.OSVersion,
		Environment: req.Environment,
		Datacenter:  req.Datacenter,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, host); err != nil {
		return nil, fmt.Errorf("create host: %w", err)
	}
	return host, nil
}

// Get retrieves a host by ID.
func (s *DefaultHostService) Get(ctx context.Context, id uuid.UUID) (*models.Host, error) {
	host, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get host: %w", err)
	}
	return host, nil
}

// List returns all hosts.
func (s *DefaultHostService) List(ctx context.Context) ([]models.Host, error) {
	hosts, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list hosts: %w", err)
	}
	return hosts, nil
}

// ListPaginated returns a page of hosts matching optional filters.
func (s *DefaultHostService) ListPaginated(ctx context.Context, filters repository.HostFilters, page, limit int) (*PaginatedHosts, error) {
	hosts, total, err := s.repo.ListPaginated(ctx, filters, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list hosts paginated: %w", err)
	}
	return &PaginatedHosts{
		Items: hosts,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

// Update modifies an existing host.
func (s *DefaultHostService) Update(ctx context.Context, id uuid.UUID, req UpdateHostRequest) (*models.Host, error) {
	host, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get host: %w", err)
	}

	host.Hostname = req.Hostname
	host.OSType = req.OSType
	host.OSName = req.OSName
	host.OSVersion = req.OSVersion
	host.Environment = req.Environment
	host.Datacenter = req.Datacenter
	host.Description = req.Description
	host.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, host); err != nil {
		return nil, fmt.Errorf("update host: %w", err)
	}
	return host, nil
}

// Delete removes a host.
func (s *DefaultHostService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete host: %w", err)
	}
	return nil
}
