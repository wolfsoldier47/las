package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"ulas-service/models"
)

// ErrBaselineNotFound is returned when a baseline cannot be located.
var ErrBaselineNotFound = errors.New("baseline not found")

// ErrBaselineVersionNotFound is returned when no baseline rows exist for a versioned scope.
var ErrBaselineVersionNotFound = errors.New("baseline version not found")

// BaselineEntryInput is a single parsed entry used for bulk versioned inserts.
type BaselineEntryInput struct {
	EntryKey   string
	EntryValue string
	CheckIDs   bool // privilege list: compare uid/gid for this entry
}

// BaselineVersionSummary represents a versioned master file scope.
type BaselineVersionSummary struct {
	OSType         models.OSType   `json:"os_type"`
	FileType       models.FileType `json:"file_type"`
	Version        int             `json:"version"`
	IsActive       bool            `json:"is_active"`
	EntryCount     int             `json:"entry_count"`
	Description    string          `json:"description,omitempty"`
	CreatedBy      string          `json:"created_by,omitempty"`
	ApprovedBy     string          `json:"approved_by,omitempty"`
	ApprovalStatus string          `json:"approval_status"`
	CreatedAt      time.Time       `json:"created_at"`
}

// ToSnapshot converts the summary to a model snapshot.
func (s BaselineVersionSummary) ToSnapshot() models.BaselineVersionSnapshot {
	return models.BaselineVersionSnapshot{
		OSType:      s.OSType,
		FileType:    s.FileType,
		Version:     s.Version,
		EntryCount:  s.EntryCount,
		Description: s.Description,
		CreatedBy:   s.CreatedBy,
		CreatedAt:   s.CreatedAt,
	}
}

// BaselineRepository defines storage operations for master baselines.
type BaselineRepository interface {
	Create(ctx context.Context, baseline *models.MasterBaseline) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.MasterBaseline, error)
	List(ctx context.Context, filters BaselineFilters) ([]models.MasterBaseline, error)
	Update(ctx context.Context, baseline *models.MasterBaseline) error
	Delete(ctx context.Context, id uuid.UUID) error
	CreateVersion(ctx context.Context, version *models.MasterBaselineVersion) error

	// Versioned bulk operations.
	CreateVersionedEntries(ctx context.Context, osType models.OSType, fileType models.FileType, version int, entries []BaselineEntryInput, createdBy, description string, active bool) error
	SetActiveVersion(ctx context.Context, osType models.OSType, fileType models.FileType, version int) error
	DeactivateScope(ctx context.Context, osType models.OSType, fileType models.FileType, version int) error
	ListVersions(ctx context.Context) ([]BaselineVersionSummary, error)
	ListVersionsPaginated(ctx context.Context, page, limit int) ([]BaselineVersionSummary, int, error)
	ListPendingVersions(ctx context.Context) ([]BaselineVersionSummary, error)

	// Approval workflow.
	ApproveVersion(ctx context.Context, osType models.OSType, fileType models.FileType, version int, approver string) (int64, error)
	GetVersionCreator(ctx context.Context, osType models.OSType, fileType models.FileType, version int) (string, error)
}

// BaselineFilters contains optional filters for listing baselines.
type BaselineFilters struct {
	OSType   *models.OSType
	FileType *models.FileType
	Version  *int
	IsActive *bool
}

// PgBaselineRepository is a PostgreSQL implementation of BaselineRepository.
type PgBaselineRepository struct {
	db *sql.DB
}

// NewPgBaselineRepository creates a new PostgreSQL baseline repository.
func NewPgBaselineRepository(db *sql.DB) *PgBaselineRepository {
	return &PgBaselineRepository{db: db}
}

// Create inserts a new master baseline.
func (r *PgBaselineRepository) Create(ctx context.Context, baseline *models.MasterBaseline) error {
	query := `
		INSERT INTO master_baselines (
			id, os_type, file_type, entry_key, entry_value,
			version, is_active, check_ids, description, created_by,
			approval_status, approved_by, approved_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := r.db.ExecContext(ctx, query,
		baseline.ID,
		baseline.OSType,
		baseline.FileType,
		baseline.EntryKey,
		baseline.EntryValue,
		baseline.Version,
		baseline.IsActive,
		baseline.CheckIDs,
		baseline.Description,
		baseline.CreatedBy,
		baseline.ApprovalStatus,
		baseline.ApprovedBy,
		baseline.ApprovedAt,
		baseline.CreatedAt,
		baseline.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create baseline: %w", err)
	}
	return nil
}

// GetByID returns a baseline by its UUID.
func (r *PgBaselineRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.MasterBaseline, error) {
	query := `
		SELECT id, os_type, file_type, entry_key, entry_value,
		       version, is_active, check_ids, description, created_by,
		       approval_status, approved_by, approved_at, created_at, updated_at
		FROM master_baselines
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var baseline models.MasterBaseline
	if err := row.Scan(
		&baseline.ID,
		&baseline.OSType,
		&baseline.FileType,
		&baseline.EntryKey,
		&baseline.EntryValue,
		&baseline.Version,
		&baseline.IsActive,
		&baseline.CheckIDs,
		&baseline.Description,
		&baseline.CreatedBy,
		&baseline.ApprovalStatus,
		&baseline.ApprovedBy,
		&baseline.ApprovedAt,
		&baseline.CreatedAt,
		&baseline.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBaselineNotFound
		}
		return nil, fmt.Errorf("get baseline by id: %w", err)
	}
	return &baseline, nil
}

// List returns baselines matching the provided filters.
func (r *PgBaselineRepository) List(ctx context.Context, filters BaselineFilters) ([]models.MasterBaseline, error) {
	query := `
		SELECT id, os_type, file_type, entry_key, entry_value,
		       version, is_active, check_ids, description, created_by,
		       approval_status, approved_by, approved_at, created_at, updated_at
		FROM master_baselines
		WHERE 1=1
	`
	var args []interface{}
	var argCount int

	if filters.OSType != nil {
		argCount++
		query += fmt.Sprintf(" AND os_type = $%d", argCount)
		args = append(args, *filters.OSType)
	}
	if filters.FileType != nil {
		argCount++
		query += fmt.Sprintf(" AND file_type = $%d", argCount)
		args = append(args, *filters.FileType)
	}
	if filters.Version != nil {
		argCount++
		query += fmt.Sprintf(" AND version = $%d", argCount)
		args = append(args, *filters.Version)
	}
	if filters.IsActive != nil {
		argCount++
		query += fmt.Sprintf(" AND is_active = $%d", argCount)
		args = append(args, *filters.IsActive)
	}

	query += " ORDER BY os_type, file_type, entry_key"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list baselines: %w", err)
	}
	defer rows.Close()

	var baselines []models.MasterBaseline
	for rows.Next() {
		var baseline models.MasterBaseline
		if err := rows.Scan(
			&baseline.ID,
			&baseline.OSType,
			&baseline.FileType,
			&baseline.EntryKey,
			&baseline.EntryValue,
			&baseline.Version,
			&baseline.IsActive,
			&baseline.CheckIDs,
			&baseline.Description,
			&baseline.CreatedBy,
			&baseline.ApprovalStatus,
			&baseline.ApprovedBy,
			&baseline.ApprovedAt,
			&baseline.CreatedAt,
			&baseline.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan baseline: %w", err)
		}
		baselines = append(baselines, baseline)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate baselines: %w", err)
	}
	return baselines, nil
}

// Update modifies an existing baseline and records a new version row.
func (r *PgBaselineRepository) Update(ctx context.Context, baseline *models.MasterBaseline) error {
	query := `
		UPDATE master_baselines
		SET os_type = $2,
		    file_type = $3,
		    entry_key = $4,
		    entry_value = $5,
		    version = $6,
		    is_active = $7,
		    check_ids = $8,
		    description = $9,
		    created_by = $10,
		    approval_status = $11,
		    approved_by = $12,
		    approved_at = $13,
		    updated_at = $14
		WHERE id = $1
	`
	res, err := r.db.ExecContext(ctx, query,
		baseline.ID,
		baseline.OSType,
		baseline.FileType,
		baseline.EntryKey,
		baseline.EntryValue,
		baseline.Version,
		baseline.IsActive,
		baseline.CheckIDs,
		baseline.Description,
		baseline.CreatedBy,
		baseline.ApprovalStatus,
		baseline.ApprovedBy,
		baseline.ApprovedAt,
		baseline.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update baseline: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrBaselineNotFound
	}
	return nil
}

// Delete removes a baseline by its UUID.
func (r *PgBaselineRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM master_baselines WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete baseline: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrBaselineNotFound
	}
	return nil
}

// CreateVersion inserts a baseline version record.
func (r *PgBaselineRepository) CreateVersion(ctx context.Context, version *models.MasterBaselineVersion) error {
	query := `
		INSERT INTO master_baseline_versions (id, baseline_id, version, entry_value, change_reason, changed_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		version.ID,
		version.BaselineID,
		version.Version,
		version.EntryValue,
		version.ChangeReason,
		version.ChangedBy,
		version.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create baseline version: %w", err)
	}
	return nil
}

// CreateVersionedEntries parses a master file into entries, removes any existing rows for the
// same scope and OS major version, and inserts the entries with the requested activation state.
// Inactive entries are inserted as pending approval; approving a version activates it.
func (r *PgBaselineRepository) CreateVersionedEntries(
	ctx context.Context,
	osType models.OSType,
	fileType models.FileType,
	version int,
	entries []BaselineEntryInput,
	createdBy, description string,
	active bool,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := r.deleteScopeVersionTx(ctx, tx, osType, fileType, version); err != nil {
		return err
	}

	approvalStatus := "pending"
	if active {
		approvalStatus = "approved"
	}

	now := time.Now().UTC()
	insertQuery := `
		INSERT INTO master_baselines (
			id, os_type, file_type, entry_key, entry_value,
			version, is_active, check_ids, description, created_by,
			approval_status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	for _, entry := range entries {
		_, err := tx.ExecContext(ctx, insertQuery,
			uuid.New(), osType, fileType,
			entry.EntryKey, entry.EntryValue,
			version, active, entry.CheckIDs, description, createdBy,
			approvalStatus, now, now,
		)
		if err != nil {
			return fmt.Errorf("insert baseline entry: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// SetActiveVersion marks the specified version as active.
func (r *PgBaselineRepository) SetActiveVersion(ctx context.Context, osType models.OSType, fileType models.FileType, version int) error {
	query := `
		UPDATE master_baselines
		SET is_active = true
		WHERE os_type = $1
		  AND file_type = $2
		  AND version = $3
	`
	_, err := r.db.ExecContext(ctx, query, osType, fileType, version)
	if err != nil {
		return fmt.Errorf("activate version: %w", err)
	}
	return nil
}

// DeactivateScope sets is_active = false for the requested scope and OS major version.
func (r *PgBaselineRepository) DeactivateScope(ctx context.Context, osType models.OSType, fileType models.FileType, version int) error {
	query := `
		UPDATE master_baselines
		SET is_active = false
		WHERE os_type = $1
		  AND file_type = $2
		  AND version = $3
	`
	_, err := r.db.ExecContext(ctx, query, osType, fileType, version)
	if err != nil {
		return fmt.Errorf("deactivate scope: %w", err)
	}
	return nil
}

// ListVersions returns a summary of every versioned master file scope.
func (r *PgBaselineRepository) ListVersions(ctx context.Context) ([]BaselineVersionSummary, error) {
	query := `
		SELECT
			b.os_type,
			b.file_type,
			b.version,
			bool_or(b.is_active) AS is_active,
			COUNT(*) AS entry_count,
			MAX(b.description) AS description,
			MAX(b.created_by) AS created_by,
			MAX(b.approved_by) AS approved_by,
			MAX(b.approval_status) AS approval_status,
			MIN(b.created_at) AS created_at
		FROM master_baselines b
		GROUP BY b.os_type, b.file_type, b.version
		ORDER BY b.os_type, b.file_type, b.version DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()

	return r.scanVersionSummaries(rows)
}

// ListVersionsPaginated returns a paginated summary of versioned master file scopes.
func (r *PgBaselineRepository) ListVersionsPaginated(ctx context.Context, page, limit int) ([]BaselineVersionSummary, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	countQuery := `
		SELECT COUNT(*) FROM (
			SELECT b.os_type, b.file_type, b.version
			FROM master_baselines b
			GROUP BY b.os_type, b.file_type, b.version
		) AS scopes
	`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count baseline versions: %w", err)
	}

	query := `
		SELECT
			b.os_type,
			b.file_type,
			b.version,
			bool_or(b.is_active) AS is_active,
			COUNT(*) AS entry_count,
			MAX(b.description) AS description,
			MAX(b.created_by) AS created_by,
			MAX(b.approved_by) AS approved_by,
			MAX(b.approval_status) AS approval_status,
			MIN(b.created_at) AS created_at
		FROM master_baselines b
		GROUP BY b.os_type, b.file_type, b.version
		ORDER BY b.os_type, b.file_type, b.version DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list versions paginated: %w", err)
	}
	defer rows.Close()

	summaries, err := r.scanVersionSummaries(rows)
	if err != nil {
		return nil, 0, err
	}
	return summaries, total, nil
}

func (r *PgBaselineRepository) scanVersionSummaries(rows *sql.Rows) ([]BaselineVersionSummary, error) {
	var summaries []BaselineVersionSummary
	for rows.Next() {
		var s BaselineVersionSummary
		if err := rows.Scan(
			&s.OSType,
			&s.FileType,
			&s.Version,
			&s.IsActive,
			&s.EntryCount,
			&s.Description,
			&s.CreatedBy,
			&s.ApprovedBy,
			&s.ApprovalStatus,
			&s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan version summary: %w", err)
		}
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate versions: %w", err)
	}
	return summaries, nil
}

// ListPendingVersions returns the distinct versioned scopes whose rows are all
// still awaiting approval, newest first.
func (r *PgBaselineRepository) ListPendingVersions(ctx context.Context) ([]BaselineVersionSummary, error) {
	query := `
		SELECT
			b.os_type,
			b.file_type,
			b.version,
			b.created_by,
			b.created_at,
			(
				SELECT COUNT(*)
				FROM master_baselines c
				WHERE c.os_type = b.os_type AND c.file_type = b.file_type AND c.version = b.version
			) AS entry_count
		FROM master_baselines b
		WHERE b.approval_status = 'pending'
		GROUP BY b.os_type, b.file_type, b.version, b.created_by, b.created_at
		ORDER BY b.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list pending versions: %w", err)
	}
	defer rows.Close()

	var summaries = make([]BaselineVersionSummary, 0)
	for rows.Next() {
		var s BaselineVersionSummary
		if err := rows.Scan(
			&s.OSType,
			&s.FileType,
			&s.Version,
			&s.CreatedBy,
			&s.CreatedAt,
			&s.EntryCount,
		); err != nil {
			return nil, fmt.Errorf("scan pending version: %w", err)
		}
		s.ApprovalStatus = "pending"
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending versions: %w", err)
	}
	return summaries, nil
}

// ApproveVersion activates a versioned scope, marks it approved, and records the
// approver. It returns the number of rows updated.
func (r *PgBaselineRepository) ApproveVersion(ctx context.Context, osType models.OSType, fileType models.FileType, version int, approver string) (int64, error) {
	query := `
		UPDATE master_baselines
		SET is_active = true,
		    approval_status = 'approved',
		    approved_by = $4,
		    approved_at = NOW(),
		    updated_at = NOW()
		WHERE os_type = $1
		  AND file_type = $2
		  AND version = $3
	`
	res, err := r.db.ExecContext(ctx, query, osType, fileType, version, approver)
	if err != nil {
		return 0, fmt.Errorf("approve version: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return rowsAffected, nil
}

// GetVersionCreator returns the created_by value of the first row in a versioned scope.
func (r *PgBaselineRepository) GetVersionCreator(ctx context.Context, osType models.OSType, fileType models.FileType, version int) (string, error) {
	query := `
		SELECT created_by
		FROM master_baselines
		WHERE os_type = $1
		  AND file_type = $2
		  AND version = $3
		LIMIT 1
	`
	var creator string
	if err := r.db.QueryRowContext(ctx, query, osType, fileType, version).Scan(&creator); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrBaselineVersionNotFound
		}
		return "", fmt.Errorf("get version creator: %w", err)
	}
	return creator, nil
}

func (r *PgBaselineRepository) deleteScopeVersionTx(
	ctx context.Context,
	tx *sql.Tx,
	osType models.OSType,
	fileType models.FileType,
	version int,
) error {
	query := `
		DELETE FROM master_baselines
		WHERE os_type = $1
		  AND file_type = $2
		  AND version = $3
	`
	_, err := tx.ExecContext(ctx, query, osType, fileType, version)
	if err != nil {
		return fmt.Errorf("delete existing scope version: %w", err)
	}
	return nil
}
