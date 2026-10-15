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
	ListFileKeyStats(ctx context.Context, scanJobID uuid.UUID, fileType models.FileType, page, limit int) ([]models.FileKeyStat, int, error)
	ListFileKeyHosts(ctx context.Context, scanJobID uuid.UUID, fileType models.FileType, key, search string, page, limit int) ([]models.FileKeyHost, int, error)
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

// ListFileKeyStats aggregates entry keys across every host snapshot of a scan
// job for one file type. Snapshot raw_content stores one "key:value" line per
// entry, so the key is split out per line and counted per host.
func (r *PgSnapshotRepository) ListFileKeyStats(ctx context.Context, scanJobID uuid.UUID, fileType models.FileType, page, limit int) ([]models.FileKeyStat, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	statsQuery := `
		SELECT key, COUNT(DISTINCT host_id) AS host_count, COUNT(*) AS occurrence_count
		FROM (
			SELECT s.host_id, split_part(line, ':', 1) AS key
			FROM host_file_snapshots s,
			     LATERAL unnest(string_to_array(s.raw_content, E'\n')) AS line
			WHERE s.scan_job_id = $1 AND s.file_type = $2 AND btrim(line) <> ''
		) entries
		GROUP BY key
	`

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ("+statsQuery+") t", scanJobID, fileType).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count file key stats: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, statsQuery+" ORDER BY host_count DESC, key ASC LIMIT $3 OFFSET $4",
		scanJobID, fileType, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list file key stats: %w", err)
	}
	defer rows.Close()

	var stats []models.FileKeyStat
	for rows.Next() {
		var stat models.FileKeyStat
		if err := rows.Scan(&stat.Key, &stat.HostCount, &stat.OccurrenceCount); err != nil {
			return nil, 0, fmt.Errorf("scan file key stat: %w", err)
		}
		stats = append(stats, stat)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate file key stats: %w", err)
	}
	return stats, total, nil
}

// ListFileKeyHosts returns the hosts of a scan that carry one entry key in
// one file type, along with the matching raw line. Paginated; total is the
// number of distinct hosts with the key. An optional search filters hostnames
// (case-insensitive substring).
func (r *PgSnapshotRepository) ListFileKeyHosts(ctx context.Context, scanJobID uuid.UUID, fileType models.FileType, key, search string, page, limit int) ([]models.FileKeyHost, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	hostFilter := `
		s.scan_job_id = $1 AND s.file_type = $2
		AND EXISTS (
			SELECT 1
			FROM unnest(string_to_array(s.raw_content, E'\n')) AS line
			WHERE split_part(line, ':', 1) = $3 AND btrim(line) <> ''
		)`
	args := []interface{}{scanJobID, fileType, key}
	if search != "" {
		hostFilter += ` AND h.hostname ILIKE '%' || $4 || '%'`
		args = append(args, search)
	}

	countQuery := `
		SELECT COUNT(*)
		FROM host_file_snapshots s
		JOIN hosts h ON h.id = s.host_id
		WHERE ` + hostFilter
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count file key hosts: %w", err)
	}

	query := `
		SELECT h.hostname,
		       (SELECT line FROM unnest(string_to_array(s.raw_content, E'\n')) AS line
		        WHERE split_part(line, ':', 1) = $3 AND btrim(line) <> ''
		        LIMIT 1)
		FROM host_file_snapshots s
		JOIN hosts h ON h.id = s.host_id
		WHERE ` + hostFilter + `
		ORDER BY h.hostname ASC
		LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)

	rows, err := r.db.QueryContext(ctx, query, append(args, limit, (page-1)*limit)...)
	if err != nil {
		return nil, 0, fmt.Errorf("list file key hosts: %w", err)
	}
	defer rows.Close()

	var hosts []models.FileKeyHost
	for rows.Next() {
		var host models.FileKeyHost
		if err := rows.Scan(&host.Hostname, &host.Value); err != nil {
			return nil, 0, fmt.Errorf("scan file key host: %w", err)
		}
		hosts = append(hosts, host)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate file key hosts: %w", err)
	}
	return hosts, total, nil
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
