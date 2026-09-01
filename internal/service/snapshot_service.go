package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"ulas-service/internal/repository"
	"ulas-service/models"
)

// SnapshotService defines business operations for snapshots.
type SnapshotService interface {
	GetHistory(ctx context.Context, hostID uuid.UUID, fileType models.FileType) ([]SnapshotHistoryItem, error)
	GetChanges(ctx context.Context, hostID uuid.UUID, fileType models.FileType) ([]models.HostFileChange, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.HostFileSnapshot, error)
}

// SnapshotHistoryItem is a snapshot enriched with change info.
type SnapshotHistoryItem struct {
	models.HostFileSnapshot
	Changed bool `json:"changed"`
}

// DefaultSnapshotService is the default implementation of SnapshotService.
type DefaultSnapshotService struct {
	repo repository.SnapshotRepository
}

// NewDefaultSnapshotService creates a new snapshot service.
func NewDefaultSnapshotService(repo repository.SnapshotRepository) *DefaultSnapshotService {
	return &DefaultSnapshotService{repo: repo}
}

// GetHistory returns the snapshot history for a host/file type, flagging changed snapshots.
func (s *DefaultSnapshotService) GetHistory(ctx context.Context, hostID uuid.UUID, fileType models.FileType) ([]SnapshotHistoryItem, error) {
	snapshots, err := s.repo.ListByHostAndType(ctx, hostID, fileType, 50)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}

	var items []SnapshotHistoryItem
	for i := range snapshots {
		changed := false
		if i < len(snapshots)-1 {
			changed = snapshots[i].RawContent != snapshots[i+1].RawContent
		}
		items = append(items, SnapshotHistoryItem{
			HostFileSnapshot: snapshots[i],
			Changed:          changed,
		})
	}
	return items, nil
}

// GetByID returns a single snapshot by ID.
func (s *DefaultSnapshotService) GetByID(ctx context.Context, id uuid.UUID) (*models.HostFileSnapshot, error) {
	snapshot, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}
	return snapshot, nil
}

// GetChanges returns recorded changes for a host/file type.
func (s *DefaultSnapshotService) GetChanges(ctx context.Context, hostID uuid.UUID, fileType models.FileType) ([]models.HostFileChange, error) {
	// For now, changes are computed on the fly from snapshots.
	// A dedicated change query can be added to the repository later.
	snapshots, err := s.repo.ListByHostAndType(ctx, hostID, fileType, 50)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}

	var changes []models.HostFileChange
	for i := range snapshots {
		if i >= len(snapshots)-1 {
			continue
		}
		current := snapshots[i]
		previous := snapshots[i+1]
		if current.RawContent == previous.RawContent {
			continue
		}

		changeType := models.HostFileChangeTypeModified
		changes = append(changes, models.HostFileChange{
			HostID:            current.HostID,
			FileType:          current.FileType,
			ChangeType:        changeType,
			PreviousContent:   &previous.RawContent,
			CurrentContent:    &current.RawContent,
			PreviousScanJobID: &previous.ScanJobID,
			CurrentScanJobID:  current.ScanJobID,
			DetectedAt:        current.SnapshotAt,
		})
	}
	return changes, nil
}
