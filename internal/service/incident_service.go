package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"ulas-service/internal/repository"
	"ulas-service/models"
)

// IncidentService defines business operations for incidents.
type IncidentService interface {
	Get(ctx context.Context, id uuid.UUID) (*models.Incident, error)
	List(ctx context.Context, filters repository.IncidentFilters) ([]models.Incident, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.IncidentStatus, notes, resolution *string) (*models.Incident, error)
}

// DefaultIncidentService is the default implementation of IncidentService.
type DefaultIncidentService struct {
	repo repository.IncidentRepository
}

// NewDefaultIncidentService creates a new incident service.
func NewDefaultIncidentService(repo repository.IncidentRepository) *DefaultIncidentService {
	return &DefaultIncidentService{repo: repo}
}

// Get retrieves an incident by ID.
func (s *DefaultIncidentService) Get(ctx context.Context, id uuid.UUID) (*models.Incident, error) {
	incident, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get incident: %w", err)
	}
	return incident, nil
}

// List returns incidents matching filters.
func (s *DefaultIncidentService) List(ctx context.Context, filters repository.IncidentFilters) ([]models.Incident, error) {
	incidents, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("list incidents: %w", err)
	}
	return incidents, nil
}

// UpdateStatus updates the status and optional notes/resolution of an incident.
func (s *DefaultIncidentService) UpdateStatus(ctx context.Context, id uuid.UUID, status models.IncidentStatus, notes, resolution *string) (*models.Incident, error) {
	incident, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get incident: %w", err)
	}

	incident.Status = status
	if notes != nil {
		incident.Notes = notes
	}
	if resolution != nil {
		incident.Resolution = resolution
	}

	if err := s.repo.Update(ctx, incident); err != nil {
		return nil, fmt.Errorf("update incident: %w", err)
	}
	return incident, nil
}
