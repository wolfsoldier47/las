package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"ulas-service/internal/repository"
	"ulas-service/models"
)

// ScanScheduleService manages recurring scan definitions.
type ScanScheduleService interface {
	Create(ctx context.Context, req CreateScanScheduleRequest) (*models.ScanSchedule, error)
	Get(ctx context.Context, id uuid.UUID) (*models.ScanSchedule, error)
	List(ctx context.Context) ([]models.ScanSchedule, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateScanScheduleRequest) (*models.ScanSchedule, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ListRuns(ctx context.Context, scheduleID uuid.UUID) ([]models.ScanScheduleRun, error)
}

// CreateScanScheduleRequest is the input for creating a scan schedule.
type CreateScanScheduleRequest struct {
	Name      string                        `json:"name" binding:"required"`
	Frequency models.ScanScheduleFrequency  `json:"frequency" binding:"required"`
	Limit     string                        `json:"limit"`
	Enabled   bool                          `json:"enabled"`
	StartAt   *time.Time                    `json:"start_at"`
	CreatedBy string                        `json:"-"`
}

// UpdateScanScheduleRequest is the input for updating a scan schedule.
type UpdateScanScheduleRequest struct {
	Name      string                        `json:"name" binding:"required"`
	Frequency models.ScanScheduleFrequency  `json:"frequency" binding:"required"`
	Limit     string                        `json:"limit"`
	Enabled   bool                          `json:"enabled"`
	NextRunAt *time.Time                    `json:"next_run_at"`
}

// DefaultScanScheduleService is the default implementation of ScanScheduleService.
type DefaultScanScheduleService struct {
	repo repository.ScanScheduleRepository
}

// NewDefaultScanScheduleService creates a new scan schedule service.
func NewDefaultScanScheduleService(repo repository.ScanScheduleRepository) *DefaultScanScheduleService {
	return &DefaultScanScheduleService{repo: repo}
}

// Create creates a new scan schedule.
func (s *DefaultScanScheduleService) Create(ctx context.Context, req CreateScanScheduleRequest) (*models.ScanSchedule, error) {
	if !models.IsValidScanScheduleFrequency(req.Frequency) {
		return nil, fmt.Errorf("invalid schedule frequency: %s", req.Frequency)
	}

	now := time.Now().UTC()
	nextRun := now
	if req.StartAt != nil {
		nextRun = *req.StartAt
	}

	schedule := &models.ScanSchedule{
		ID:        uuid.New(),
		Name:      req.Name,
		Frequency: req.Frequency,
		Limit:     req.Limit,
		Enabled:   req.Enabled,
		NextRunAt: &nextRun,
		CreatedBy: req.CreatedBy,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, schedule); err != nil {
		return nil, fmt.Errorf("create scan schedule: %w", err)
	}
	return schedule, nil
}

// Get returns a scan schedule by ID.
func (s *DefaultScanScheduleService) Get(ctx context.Context, id uuid.UUID) (*models.ScanSchedule, error) {
	schedule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get scan schedule: %w", err)
	}
	return schedule, nil
}

// List returns all scan schedules.
func (s *DefaultScanScheduleService) List(ctx context.Context) ([]models.ScanSchedule, error) {
	schedules, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list scan schedules: %w", err)
	}
	return schedules, nil
}

// Update modifies an existing scan schedule.
func (s *DefaultScanScheduleService) Update(ctx context.Context, id uuid.UUID, req UpdateScanScheduleRequest) (*models.ScanSchedule, error) {
	if !models.IsValidScanScheduleFrequency(req.Frequency) {
		return nil, fmt.Errorf("invalid schedule frequency: %s", req.Frequency)
	}

	schedule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get scan schedule: %w", err)
	}

	now := time.Now().UTC()
	schedule.Name = req.Name
	schedule.Frequency = req.Frequency
	schedule.Limit = req.Limit
	schedule.Enabled = req.Enabled
	schedule.NextRunAt = req.NextRunAt
	schedule.UpdatedAt = now

	if err := s.repo.Update(ctx, schedule); err != nil {
		return nil, fmt.Errorf("update scan schedule: %w", err)
	}
	return schedule, nil
}

// Delete removes a scan schedule.
func (s *DefaultScanScheduleService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete scan schedule: %w", err)
	}
	return nil
}

// ListRuns returns the run history for a schedule.
func (s *DefaultScanScheduleService) ListRuns(ctx context.Context, scheduleID uuid.UUID) ([]models.ScanScheduleRun, error) {
	runs, err := s.repo.ListRunsBySchedule(ctx, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("list scan schedule runs: %w", err)
	}
	return runs, nil
}
