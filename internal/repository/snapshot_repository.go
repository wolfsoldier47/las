package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"ulas-service/models"
)

// ErrSnapshotNotFound is returned when a snapshot cannot be located.
var ErrSnapshotNotFound = errors.New("snapshot not found")

// SnapshotRepository defines storage operations for host file snapshots.
type SnapshotRepository interface {
	Create(ctx context.Context, snapshot *models.HostFileSnapshot) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.HostFileSnapshot, error)
	GetByScanResultAndType(ctx context.Context, scanResultID uuid.UUID, fileType models.FileType) (*models.HostFileSnapshot, error)
	GetLatestByHostAndType(ctx context.Context, hostID uuid.UUID, fileType models.FileType) (*models.HostFileSnapshot, error)
	ListByHostAndType(ctx context.Context, hostID uuid.UUID, fileType models.FileType, limit int) ([]models.HostFileSnapshot, error)
	CreateChange(ctx context.Context, change *models.HostFileChange) error
}

// PgSnapshotRepository is a PostgreSQL implementation of SnapshotRepository.
type PgSnapshotRepository struct {
	db *sql.DB
}

// NewPgSnapshotRepository creates a new PostgreSQL snapshot repository.
func NewPgSnapshotRepository(db *sql.DB) *PgSnapshotRepository {
	return &PgSnapshotRepository{db: db}
}

// Create inserts a new host file snapshot.
func (r *PgSnapshotRepository) Create(ctx context.Context, snapshot *models.HostFileSnapshot) error {
	query := `
		INSERT INTO host_file_snapshots (
			id, scan_result_id, host_id, scan_job_id, file_type, raw_content, line_count, snapshot_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		snapshot.ID,
		snapshot.ScanResultID,
		snapshot.HostID,
		snapshot.ScanJobID,
		snapshot.FileType,
		snapshot.RawContent,
		snapshot.LineCount,
		snapshot.SnapshotAt,
	)
	if err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}
	return nil
}

// GetByID returns a snapshot by its UUID.
func (r *PgSnapshotRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.HostFileSnapshot, error) {
	query := `
		SELECT id, scan_result_id, host_id, scan_job_id, file_type, raw_content, line_count, snapshot_at
		FROM host_file_snapshots
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var snapshot models.HostFileSnapshot
	if err := row.Scan(
		&snapshot.ID,
		&snapshot.ScanResultID,
		&snapshot.HostID,
		&snapshot.ScanJobID,
		&snapshot.FileType,
		&snapshot.RawContent,
		&snapshot.LineCount,
		&snapshot.SnapshotAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSnapshotNotFound
		}
		return nil, fmt.Errorf("get snapshot by id: %w", err)
	}
	return &snapshot, nil
}

// GetByScanResultAndType returns a snapshot for a specific scan result and file type.
func (r *PgSnapshotRepository) GetByScanResultAndType(ctx context.Context, scanResultID uuid.UUID, fileType models.FileType) (*models.HostFileSnapshot, error) {
	query := `
		SELECT id, scan_result_id, host_id, scan_job_id, file_type, raw_content, line_count, snapshot_at
		FROM host_file_snapshots
		WHERE scan_result_id = $1 AND file_type = $2
	`
	row := r.db.QueryRowContext(ctx, query, scanResultID, fileType)

	var snapshot models.HostFileSnapshot
	if err := row.Scan(
		&snapshot.ID,
		&snapshot.ScanResultID,
		&snapshot.HostID,
		&snapshot.ScanJobID,
		&snapshot.FileType,
		&snapshot.RawContent,
		&snapshot.LineCount,
		&snapshot.SnapshotAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("snapshot not found")
		}
		return nil, fmt.Errorf("get snapshot by scan result: %w", err)
	}
	return &snapshot, nil
}

// GetLatestByHostAndType returns the most recent snapshot for a host and file type.
func (r *PgSnapshotRepository) GetLatestByHostAndType(ctx context.Context, hostID uuid.UUID, fileType models.FileType) (*models.HostFileSnapshot, error) {
	query := `
		SELECT id, scan_result_id, host_id, scan_job_id, file_type, raw_content, line_count, snapshot_at
		FROM host_file_snapshots
		WHERE host_id = $1 AND file_type = $2
		ORDER BY snapshot_at DESC
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, hostID, fileType)

	var snapshot models.HostFileSnapshot
	if err := row.Scan(
		&snapshot.ID,
		&snapshot.ScanResultID,
		&snapshot.HostID,
		&snapshot.ScanJobID,
		&snapshot.FileType,
		&snapshot.RawContent,
		&snapshot.LineCount,
		&snapshot.SnapshotAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("snapshot not found")
		}
		return nil, fmt.Errorf("get latest snapshot: %w", err)
	}
	return &snapshot, nil
}

// ListByHostAndType returns recent snapshots for a host and file type.
func (r *PgSnapshotRepository) ListByHostAndType(ctx context.Context, hostID uuid.UUID, fileType models.FileType, limit int) ([]models.HostFileSnapshot, error) {
	if limit <= 0 {
		limit = 10
	}
	query := `
		SELECT id, scan_result_id, host_id, scan_job_id, file_type, raw_content, line_count, snapshot_at
		FROM host_file_snapshots
		WHERE host_id = $1 AND file_type = $2
		ORDER BY snapshot_at DESC
		LIMIT $3
	`
	rows, err := r.db.QueryContext(ctx, query, hostID, fileType, limit)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []models.HostFileSnapshot
	for rows.Next() {
		var snapshot models.HostFileSnapshot
		if err := rows.Scan(
			&snapshot.ID,
			&snapshot.ScanResultID,
			&snapshot.HostID,
			&snapshot.ScanJobID,
			&snapshot.FileType,
			&snapshot.RawContent,
			&snapshot.LineCount,
			&snapshot.SnapshotAt,
		); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshots: %w", err)
	}
	return snapshots, nil
}

// CreateChange inserts a host file change record.
func (r *PgSnapshotRepository) CreateChange(ctx context.Context, change *models.HostFileChange) error {
	query := `
		INSERT INTO host_file_changes (
			id, host_id, file_type, change_type, previous_content, current_content,
			previous_scan_job_id, current_scan_job_id, detected_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		change.ID,
		change.HostID,
		change.FileType,
		change.ChangeType,
		change.PreviousContent,
		change.CurrentContent,
		change.PreviousScanJobID,
		change.CurrentScanJobID,
		change.DetectedAt,
	)
	if err != nil {
		return fmt.Errorf("create change: %w", err)
	}
	return nil
}
