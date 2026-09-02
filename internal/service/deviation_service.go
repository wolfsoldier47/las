package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"ulas-service/internal/repository"
	"ulas-service/models"
)

// DeviationService defines business operations for allowed deviations.
type DeviationService interface {
	Create(ctx context.Context, req CreateDeviationRequest) (*models.AllowedDeviation, error)
	Get(ctx context.Context, id uuid.UUID) (*models.AllowedDeviation, error)
	List(ctx context.Context, filters repository.DeviationFilters) ([]models.AllowedDeviation, error)
	ListPaginated(ctx context.Context, filters repository.DeviationFilters, page, limit int) (*PaginatedDeviations, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateDeviationRequest) (*models.AllowedDeviation, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// PaginatedDeviations is a page of allowed deviations.
type PaginatedDeviations struct {
	Items         []models.AllowedDeviation `json:"items"`
	Total         int                       `json:"total"`
	Page          int                       `json:"page"`
	Limit         int                       `json:"limit"`
	ActiveTotal   int                       `json:"active_total"`
	InactiveTotal int                       `json:"inactive_total"`
}

// CreateDeviationRequest is the input for creating an allowed deviation.
type CreateDeviationRequest struct {
	Hostname      string          `json:"hostname" binding:"required"`
	FileType      models.FileType `json:"file_type" binding:"required"`
	EntryLine     string          `json:"entry_line" binding:"required"`
	Justification string          `json:"justification" binding:"required"`
	ApprovedBy    string          `json:"approved_by" binding:"required"`
	ExpiresAt     *time.Time      `json:"expires_at"`
}

// UpdateDeviationRequest is the input for updating an allowed deviation.
type UpdateDeviationRequest struct {
	Hostname      string          `json:"hostname" binding:"required"`
	FileType      models.FileType `json:"file_type" binding:"required"`
	EntryLine     string          `json:"entry_line" binding:"required"`
	Justification string          `json:"justification" binding:"required"`
	ApprovedBy    string          `json:"approved_by" binding:"required"`
	ExpiresAt     *time.Time      `json:"expires_at"`
	IsActive      bool            `json:"is_active"`
}

// DefaultDeviationService is the default implementation of DeviationService.
type DefaultDeviationService struct {
	repo repository.DeviationRepository
}

// NewDefaultDeviationService creates a new deviation service.
func NewDefaultDeviationService(repo repository.DeviationRepository) *DefaultDeviationService {
	return &DefaultDeviationService{repo: repo}
}

// Create registers a new allowed deviation. The hostname is stored as free text
// and is not required to exist in the host inventory. The entry line is parsed
// into key and value in the backend (e.g. /etc/passwd or /etc/group line).
func (s *DefaultDeviationService) Create(ctx context.Context, req CreateDeviationRequest) (*models.AllowedDeviation, error) {
	entryKey, entryValue, err := parseEntryLine(req.FileType, req.EntryLine)
	if err != nil {
		return nil, err
	}

	if _, err := s.repo.GetByHostFileKey(ctx, req.Hostname, req.FileType, entryKey); err == nil {
		return nil, repository.ErrDuplicateDeviation
	} else if !errors.Is(err, repository.ErrDeviationNotFound) {
		return nil, fmt.Errorf("check duplicate deviation: %w", err)
	}

	now := time.Now().UTC()
	deviation := &models.AllowedDeviation{
		ID:            uuid.New(),
		Hostname:      req.Hostname,
		FileType:      req.FileType,
		EntryKey:      entryKey,
		EntryValue:    entryValue,
		Justification: req.Justification,
		ApprovedBy:    req.ApprovedBy,
		ApprovedAt:    now,
		ExpiresAt:     req.ExpiresAt,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.Create(ctx, deviation); err != nil {
		return nil, fmt.Errorf("create deviation: %w", err)
	}
	return deviation, nil
}

// Get retrieves an allowed deviation by ID.
func (s *DefaultDeviationService) Get(ctx context.Context, id uuid.UUID) (*models.AllowedDeviation, error) {
	deviation, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get deviation: %w", err)
	}
	return deviation, nil
}

// List returns allowed deviations matching filters.
func (s *DefaultDeviationService) List(ctx context.Context, filters repository.DeviationFilters) ([]models.AllowedDeviation, error) {
	deviations, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("list deviations: %w", err)
	}
	return deviations, nil
}

// ListPaginated returns a paginated list of allowed deviations matching filters.
func (s *DefaultDeviationService) ListPaginated(ctx context.Context, filters repository.DeviationFilters, page, limit int) (*PaginatedDeviations, error) {
	deviations, total, err := s.repo.ListPaginated(ctx, filters, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list deviations paginated: %w", err)
	}

	activeTotal, inactiveTotal, err := s.repo.CountDeviations(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("count deviations: %w", err)
	}

	return &PaginatedDeviations{
		Items:         deviations,
		Total:         total,
		Page:          page,
		Limit:         limit,
		ActiveTotal:   activeTotal,
		InactiveTotal: inactiveTotal,
	}, nil
}

// Update modifies an existing allowed deviation.
func (s *DefaultDeviationService) Update(ctx context.Context, id uuid.UUID, req UpdateDeviationRequest) (*models.AllowedDeviation, error) {
	deviation, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get deviation: %w", err)
	}

	entryKey, entryValue, err := parseEntryLine(req.FileType, req.EntryLine)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByHostFileKey(ctx, req.Hostname, req.FileType, entryKey)
	if err == nil && existing.ID != id {
		return nil, repository.ErrDuplicateDeviation
	} else if err != nil && !errors.Is(err, repository.ErrDeviationNotFound) {
		return nil, fmt.Errorf("check duplicate deviation: %w", err)
	}

	deviation.Hostname = req.Hostname
	deviation.FileType = req.FileType
	deviation.EntryKey = entryKey
	deviation.EntryValue = entryValue
	deviation.Justification = req.Justification
	deviation.ApprovedBy = req.ApprovedBy
	deviation.ExpiresAt = req.ExpiresAt
	deviation.IsActive = req.IsActive
	deviation.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, deviation); err != nil {
		return nil, fmt.Errorf("update deviation: %w", err)
	}
	return deviation, nil
}

// Delete removes an allowed deviation.
func (s *DefaultDeviationService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete deviation: %w", err)
	}
	return nil
}

// parseEntryLine splits a passwd/group style line into key and value.
// key is the first field; value is everything after the first colon.
// For group entries, the value is normalized (member list sorted, trailing
// colon stripped) so it matches the scanned snapshot format.
// A missing or empty value is stored as nil (wildcard).
func parseEntryLine(fileType models.FileType, line string) (string, *string, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", nil, fmt.Errorf("entry line is required")
	}

	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("entry line must contain at least one colon separating key and value")
	}

	key := strings.TrimSpace(parts[0])
	if key == "" {
		return "", nil, fmt.Errorf("entry key cannot be empty")
	}

	value := strings.TrimSpace(parts[1])
	if value == "" {
		return key, nil, nil
	}

	if fileType == models.FileTypeGroup {
		value = normalizeGroupSnapshotValue(value)
	}

	return key, &value, nil
}
