package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"ulas-service/internal/config"
	"ulas-service/internal/repository"
	"ulas-service/models"
)

// BaselineService defines business operations for master baselines.
type BaselineService interface {
	Create(ctx context.Context, req CreateBaselineRequest) (*models.MasterBaseline, error)
	Get(ctx context.Context, id uuid.UUID) (*models.MasterBaseline, error)
	List(ctx context.Context, filters repository.BaselineFilters) ([]models.MasterBaseline, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateBaselineRequest) (*models.MasterBaseline, error)
	Delete(ctx context.Context, id uuid.UUID) error

	UploadMasterFile(ctx context.Context, req UploadMasterFileRequest) (int, error)
	ListVersions(ctx context.Context) ([]repository.BaselineVersionSummary, error)
	ListVersionsPaginated(ctx context.Context, page, limit int) (*PaginatedBaselineVersions, error)
	ActivateVersion(ctx context.Context, osType models.OSType, fileType models.FileType, version int) error
	DeactivateScope(ctx context.Context, osType models.OSType, fileType models.FileType, version int) error
	OSVersions() map[string][]int
}

// CreateBaselineRequest is the input for creating a baseline.
type CreateBaselineRequest struct {
	OSType      models.OSType   `json:"os_type" binding:"required"`
	FileType    models.FileType `json:"file_type" binding:"required"`
	EntryKey    string          `json:"entry_key" binding:"required"`
	EntryValue  string          `json:"entry_value" binding:"required"`
	Version     int             `json:"version" binding:"required"`
	Description string          `json:"description"`
	CreatedBy   string          `json:"created_by"`
}

// UpdateBaselineRequest is the input for updating a baseline.
type UpdateBaselineRequest struct {
	OSType       models.OSType   `json:"os_type" binding:"required"`
	FileType     models.FileType `json:"file_type" binding:"required"`
	EntryKey     string          `json:"entry_key" binding:"required"`
	EntryValue   string          `json:"entry_value" binding:"required"`
	Version      int             `json:"version" binding:"required"`
	Description  string          `json:"description"`
	ChangeReason string          `json:"change_reason"`
	ChangedBy    string          `json:"changed_by"`
}

// UploadMasterFileRequest is the input for pasting a full /etc/passwd or /etc/group file.
type UploadMasterFileRequest struct {
	OSType      models.OSType   `json:"os_type" binding:"required"`
	FileType    models.FileType `json:"file_type" binding:"required"`
	Version     int             `json:"version" binding:"required"`
	Content     string          `json:"content" binding:"required"`
	Description string          `json:"description"`
	CreatedBy   string          `json:"created_by"`
}

// DefaultBaselineService is the default implementation of BaselineService.
type DefaultBaselineService struct {
	repo repository.BaselineRepository
	cfg  *config.AppConfig
}

// NewDefaultBaselineService creates a new baseline service.
func NewDefaultBaselineService(repo repository.BaselineRepository, cfg *config.AppConfig) *DefaultBaselineService {
	return &DefaultBaselineService{repo: repo, cfg: cfg}
}

// Create registers a new master baseline.
func (s *DefaultBaselineService) Create(ctx context.Context, req CreateBaselineRequest) (*models.MasterBaseline, error) {
	if err := s.validateOSVersion(req.OSType, req.Version); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	baseline := &models.MasterBaseline{
		ID:          uuid.New(),
		OSType:      req.OSType,
		FileType:    req.FileType,
		EntryKey:    req.EntryKey,
		EntryValue:  req.EntryValue,
		Version:     req.Version,
		IsActive:    true,
		Description: req.Description,
		CreatedBy:   req.CreatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, baseline); err != nil {
		return nil, fmt.Errorf("create baseline: %w", err)
	}
	return baseline, nil
}

// Get retrieves a baseline by ID.
func (s *DefaultBaselineService) Get(ctx context.Context, id uuid.UUID) (*models.MasterBaseline, error) {
	baseline, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get baseline: %w", err)
	}
	return baseline, nil
}

// List returns baselines matching filters.
func (s *DefaultBaselineService) List(ctx context.Context, filters repository.BaselineFilters) ([]models.MasterBaseline, error) {
	baselines, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("list baselines: %w", err)
	}
	return baselines, nil
}

// Update modifies an existing baseline and records a version row.
func (s *DefaultBaselineService) Update(ctx context.Context, id uuid.UUID, req UpdateBaselineRequest) (*models.MasterBaseline, error) {
	if err := s.validateOSVersion(req.OSType, req.Version); err != nil {
		return nil, err
	}

	baseline, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get baseline: %w", err)
	}

	oldValue := baseline.EntryValue
	baseline.OSType = req.OSType
	baseline.FileType = req.FileType
	baseline.EntryKey = req.EntryKey
	baseline.EntryValue = req.EntryValue
	baseline.Version = req.Version
	baseline.Description = req.Description
	baseline.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, baseline); err != nil {
		return nil, fmt.Errorf("update baseline: %w", err)
	}

	version := &models.MasterBaselineVersion{
		ID:           uuid.New(),
		BaselineID:   baseline.ID,
		Version:      baseline.Version,
		EntryValue:   oldValue,
		ChangeReason: req.ChangeReason,
		ChangedBy:    req.ChangedBy,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.repo.CreateVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("record baseline version: %w", err)
	}

	return baseline, nil
}

// Delete removes a baseline.
func (s *DefaultBaselineService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete baseline: %w", err)
	}
	return nil
}

// UploadMasterFile parses a full /etc/passwd or /etc/group file and stores it as a new active version.
func (s *DefaultBaselineService) UploadMasterFile(ctx context.Context, req UploadMasterFileRequest) (int, error) {
	if err := s.validateOSVersion(req.OSType, req.Version); err != nil {
		return 0, err
	}

	entries, err := parseMasterFileContent(req.FileType, req.Content)
	if err != nil {
		return 0, fmt.Errorf("parse master file: %w", err)
	}
	if len(entries) == 0 {
		return 0, fmt.Errorf("no valid entries found in uploaded file")
	}

	if err := s.repo.CreateVersionedEntries(
		ctx,
		req.OSType,
		req.FileType,
		req.Version,
		entries,
		req.CreatedBy,
		req.Description,
	); err != nil {
		return 0, fmt.Errorf("create versioned entries: %w", err)
	}
	return req.Version, nil
}

// PaginatedBaselineVersions is a page of baseline version summaries.
type PaginatedBaselineVersions struct {
	Items []repository.BaselineVersionSummary `json:"items"`
	Total int                                 `json:"total"`
	Page  int                                 `json:"page"`
	Limit int                                 `json:"limit"`
}

// ListVersions returns all versioned master file scopes.
func (s *DefaultBaselineService) ListVersions(ctx context.Context) ([]repository.BaselineVersionSummary, error) {
	versions, err := s.repo.ListVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	return versions, nil
}

// ListVersionsPaginated returns a paginated list of versioned master file scopes.
func (s *DefaultBaselineService) ListVersionsPaginated(ctx context.Context, page, limit int) (*PaginatedBaselineVersions, error) {
	versions, total, err := s.repo.ListVersionsPaginated(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list versions paginated: %w", err)
	}
	return &PaginatedBaselineVersions{
		Items: versions,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

// ActivateVersion makes a specific version the active one for its scope.
func (s *DefaultBaselineService) ActivateVersion(ctx context.Context, osType models.OSType, fileType models.FileType, version int) error {
	if err := s.validateOSVersion(osType, version); err != nil {
		return err
	}
	if err := s.repo.SetActiveVersion(ctx, osType, fileType, version); err != nil {
		return fmt.Errorf("activate version: %w", err)
	}
	return nil
}

// DeactivateScope disables the active version for a scope without activating another.
func (s *DefaultBaselineService) DeactivateScope(ctx context.Context, osType models.OSType, fileType models.FileType, version int) error {
	if err := s.validateOSVersion(osType, version); err != nil {
		return err
	}
	if err := s.repo.DeactivateScope(ctx, osType, fileType, version); err != nil {
		return fmt.Errorf("deactivate scope: %w", err)
	}
	return nil
}

// validateOSVersion ensures the requested major version is allowed for the OS type.
func (s *DefaultBaselineService) validateOSVersion(osType models.OSType, version int) error {
	if s.cfg == nil {
		return nil
	}
	if !s.cfg.IsAllowedOSVersion(string(osType), version) {
		return fmt.Errorf("version %d is not an allowed major version for %s", version, osType)
	}
	return nil
}

// OSVersions returns the configured allowed major versions per OS type.
func (s *DefaultBaselineService) OSVersions() map[string][]int {
	if s.cfg == nil || s.cfg.OSVersions == nil {
		return map[string][]int{}
	}
	versions := make(map[string][]int, len(s.cfg.OSVersions))
	for k, v := range s.cfg.OSVersions {
		versions[k] = make([]int, len(v))
		copy(versions[k], v)
	}
	return versions
}

func parseMasterFileContent(fileType models.FileType, content string) ([]repository.BaselineEntryInput, error) {
	var entries []repository.BaselineEntryInput
	seen := make(map[string]bool)

	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}

		key := parts[0]
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true

		var value string
		switch fileType {
		case models.FileTypePasswd:
			// passwd: username:hash:uid:gid:gecos:home:shell
			// Store everything after the username so direct line comparisons work.
			value = strings.Join(parts[1:], ":")
		case models.FileTypeGroup:
			// group: groupname:hash:gid:member1,member2,...
			// Normalize member ordering so reordering does not produce deviations.
			value = normalizeGroupLine(parts)
		default:
			return nil, fmt.Errorf("unsupported file type: %s", fileType)
		}

		entries = append(entries, repository.BaselineEntryInput{
			EntryKey:   key,
			EntryValue: value,
		})
	}

	return entries, nil
}

func normalizeGroupLine(parts []string) string {
	if len(parts) < 3 {
		return strings.Join(parts[1:], ":")
	}
	// parts[1] = password, parts[2] = gid
	base := parts[1] + ":" + parts[2]
	if len(parts) < 4 || strings.TrimSpace(parts[3]) == "" {
		return base
	}
	members := strings.Split(parts[3], ",")
	for i := range members {
		members[i] = strings.TrimSpace(members[i])
	}
	sort.Strings(members)
	return base + ":" + strings.Join(members, ",")
}
