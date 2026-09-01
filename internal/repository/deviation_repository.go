package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"ulas-service/models"
)

// ErrDeviationNotFound is returned when a deviation cannot be located.
var ErrDeviationNotFound = errors.New("deviation not found")

// ErrDuplicateDeviation is returned when a deviation with the same host, file, and key already exists.
var ErrDuplicateDeviation = errors.New("deviation already exists for this host, file, and key")

// DeviationRepository defines storage operations for allowed deviations.
type DeviationRepository interface {
	Create(ctx context.Context, deviation *models.AllowedDeviation) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.AllowedDeviation, error)
	GetByHostFileKey(ctx context.Context, hostname string, fileType models.FileType, entryKey string) (*models.AllowedDeviation, error)
	List(ctx context.Context, filters DeviationFilters) ([]models.AllowedDeviation, error)
	ListPaginated(ctx context.Context, filters DeviationFilters, page, limit int) ([]models.AllowedDeviation, int, error)
	CountDeviations(ctx context.Context, filters DeviationFilters) (active, inactive int, err error)
	Update(ctx context.Context, deviation *models.AllowedDeviation) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// DeviationFilters contains optional filters for listing deviations.
type DeviationFilters struct {
	Hostname *string
	Search   *string
	FileType *models.FileType
	Active   *bool
}

// PgDeviationRepository is a PostgreSQL implementation of DeviationRepository.
type PgDeviationRepository struct {
	db *sql.DB
}

// NewPgDeviationRepository creates a new PostgreSQL deviation repository.
func NewPgDeviationRepository(db *sql.DB) *PgDeviationRepository {
	return &PgDeviationRepository{db: db}
}

// Create inserts a new allowed deviation.
func (r *PgDeviationRepository) Create(ctx context.Context, deviation *models.AllowedDeviation) error {
	query := `
		INSERT INTO allowed_deviations (
			id, hostname, file_type, entry_key, entry_value, justification,
			approved_by, approved_at, expires_at, is_active, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.ExecContext(ctx, query,
		deviation.ID,
		deviation.Hostname,
		deviation.FileType,
		deviation.EntryKey,
		deviation.EntryValue,
		deviation.Justification,
		deviation.ApprovedBy,
		deviation.ApprovedAt,
		deviation.ExpiresAt,
		deviation.IsActive,
		deviation.CreatedAt,
		deviation.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create deviation: %w", err)
	}
	return nil
}

// GetByHostFileKey returns an active or inactive deviation by hostname, file type, and entry key.
func (r *PgDeviationRepository) GetByHostFileKey(ctx context.Context, hostname string, fileType models.FileType, entryKey string) (*models.AllowedDeviation, error) {
	query := `
		SELECT id, hostname, file_type, entry_key, entry_value, justification,
		       approved_by, approved_at, expires_at, is_active, created_at, updated_at
		FROM allowed_deviations
		WHERE hostname = $1 AND file_type = $2 AND entry_key = $3
	`
	row := r.db.QueryRowContext(ctx, query, hostname, fileType, entryKey)

	var deviation models.AllowedDeviation
	if err := row.Scan(
		&deviation.ID,
		&deviation.Hostname,
		&deviation.FileType,
		&deviation.EntryKey,
		&deviation.EntryValue,
		&deviation.Justification,
		&deviation.ApprovedBy,
		&deviation.ApprovedAt,
		&deviation.ExpiresAt,
		&deviation.IsActive,
		&deviation.CreatedAt,
		&deviation.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviationNotFound
		}
		return nil, fmt.Errorf("get deviation by host file key: %w", err)
	}
	return &deviation, nil
}

// GetByID returns an allowed deviation by its UUID.
func (r *PgDeviationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.AllowedDeviation, error) {
	query := `
		SELECT id, hostname, file_type, entry_key, entry_value, justification,
		       approved_by, approved_at, expires_at, is_active, created_at, updated_at
		FROM allowed_deviations
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var deviation models.AllowedDeviation
	if err := row.Scan(
		&deviation.ID,
		&deviation.Hostname,
		&deviation.FileType,
		&deviation.EntryKey,
		&deviation.EntryValue,
		&deviation.Justification,
		&deviation.ApprovedBy,
		&deviation.ApprovedAt,
		&deviation.ExpiresAt,
		&deviation.IsActive,
		&deviation.CreatedAt,
		&deviation.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviationNotFound
		}
		return nil, fmt.Errorf("get deviation by id: %w", err)
	}
	return &deviation, nil
}

// List returns allowed deviations matching the provided filters.
func (r *PgDeviationRepository) List(ctx context.Context, filters DeviationFilters) ([]models.AllowedDeviation, error) {
	where, args := r.buildDeviationWhere(filters)
	query := `
		SELECT id, hostname, file_type, entry_key, entry_value, justification,
		       approved_by, approved_at, expires_at, is_active, created_at, updated_at
		FROM allowed_deviations
	` + where + `
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list deviations: %w", err)
	}
	defer rows.Close()

	return r.scanDeviations(rows)
}

// ListPaginated returns a paginated list of allowed deviations matching filters.
func (r *PgDeviationRepository) ListPaginated(ctx context.Context, filters DeviationFilters, page, limit int) ([]models.AllowedDeviation, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	where, args := r.buildDeviationWhere(filters)

	countQuery := "SELECT COUNT(*) FROM allowed_deviations" + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count deviations: %w", err)
	}

	query := `
		SELECT id, hostname, file_type, entry_key, entry_value, justification,
		       approved_by, approved_at, expires_at, is_active, created_at, updated_at
		FROM allowed_deviations
	` + where + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
	rows, err := r.db.QueryContext(ctx, query, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("list deviations paginated: %w", err)
	}
	defer rows.Close()

	deviations, err := r.scanDeviations(rows)
	if err != nil {
		return nil, 0, err
	}
	return deviations, total, nil
}

func (r *PgDeviationRepository) buildDeviationWhere(filters DeviationFilters) (string, []interface{}) {
	where := " WHERE 1=1"
	var args []interface{}
	var argCount int

	if filters.Hostname != nil {
		argCount++
		where += fmt.Sprintf(" AND hostname = $%d", argCount)
		args = append(args, *filters.Hostname)
	}
	if filters.Search != nil && *filters.Search != "" {
		argCount++
		where += fmt.Sprintf(" AND hostname ILIKE $%d", argCount)
		args = append(args, "%"+*filters.Search+"%")
	}
	if filters.FileType != nil {
		argCount++
		where += fmt.Sprintf(" AND file_type = $%d", argCount)
		args = append(args, *filters.FileType)
	}
	if filters.Active != nil {
		argCount++
		where += fmt.Sprintf(" AND is_active = $%d", argCount)
		args = append(args, *filters.Active)
	}

	return where, args
}

// CountDeviations returns the number of active and inactive deviations matching filters.
func (r *PgDeviationRepository) CountDeviations(ctx context.Context, filters DeviationFilters) (active, inactive int, err error) {
	// Drop the active filter so we get counts across both states.
	countFilters := filters
	countFilters.Active = nil
	where, args := r.buildDeviationWhere(countFilters)

	query := "SELECT COUNT(*) FILTER (WHERE is_active = true), COUNT(*) FILTER (WHERE is_active = false) FROM allowed_deviations" + where
	row := r.db.QueryRowContext(ctx, query, args...)
	if err := row.Scan(&active, &inactive); err != nil {
		return 0, 0, fmt.Errorf("count deviations: %w", err)
	}
	return active, inactive, nil
}

func (r *PgDeviationRepository) scanDeviations(rows *sql.Rows) ([]models.AllowedDeviation, error) {
	var deviations []models.AllowedDeviation
	for rows.Next() {
		var deviation models.AllowedDeviation
		if err := rows.Scan(
			&deviation.ID,
			&deviation.Hostname,
			&deviation.FileType,
			&deviation.EntryKey,
			&deviation.EntryValue,
			&deviation.Justification,
			&deviation.ApprovedBy,
			&deviation.ApprovedAt,
			&deviation.ExpiresAt,
			&deviation.IsActive,
			&deviation.CreatedAt,
			&deviation.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan deviation: %w", err)
		}
		deviations = append(deviations, deviation)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deviations: %w", err)
	}
	return deviations, nil
}

// Update modifies an existing allowed deviation.
func (r *PgDeviationRepository) Update(ctx context.Context, deviation *models.AllowedDeviation) error {
	query := `
		UPDATE allowed_deviations
		SET hostname = $2,
		    file_type = $3,
		    entry_key = $4,
		    entry_value = $5,
		    justification = $6,
		    approved_by = $7,
		    approved_at = $8,
		    expires_at = $9,
		    is_active = $10,
		    updated_at = $11
		WHERE id = $1
	`
	res, err := r.db.ExecContext(ctx, query,
		deviation.ID,
		deviation.Hostname,
		deviation.FileType,
		deviation.EntryKey,
		deviation.EntryValue,
		deviation.Justification,
		deviation.ApprovedBy,
		deviation.ApprovedAt,
		deviation.ExpiresAt,
		deviation.IsActive,
		deviation.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update deviation: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrDeviationNotFound
	}
	return nil
}

// Delete removes an allowed deviation by its UUID.
func (r *PgDeviationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM allowed_deviations WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete deviation: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrDeviationNotFound
	}
	return nil
}
