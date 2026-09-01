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

// ErrScanScheduleNotFound is returned when a scan schedule cannot be located.
var ErrScanScheduleNotFound = errors.New("scan schedule not found")

// ScanScheduleRepository defines storage operations for scan schedules and their run history.
type ScanScheduleRepository interface {
	Create(ctx context.Context, schedule *models.ScanSchedule) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.ScanSchedule, error)
	List(ctx context.Context) ([]models.ScanSchedule, error)
	Update(ctx context.Context, schedule *models.ScanSchedule) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListEnabledDue(ctx context.Context, before time.Time) ([]models.ScanSchedule, error)

	CreateRun(ctx context.Context, run *models.ScanScheduleRun) error
	ListRunsBySchedule(ctx context.Context, scheduleID uuid.UUID) ([]models.ScanScheduleRun, error)
}

// PgScanScheduleRepository is a PostgreSQL implementation of ScanScheduleRepository.
type PgScanScheduleRepository struct {
	db *sql.DB
}

// NewPgScanScheduleRepository creates a new PostgreSQL scan schedule repository.
func NewPgScanScheduleRepository(db *sql.DB) *PgScanScheduleRepository {
	return &PgScanScheduleRepository{db: db}
}

// Create inserts a new scan schedule.
func (r *PgScanScheduleRepository) Create(ctx context.Context, schedule *models.ScanSchedule) error {
	query := `
		INSERT INTO scan_schedules (
			id, name, frequency, "limit", enabled, next_run_at, last_run_at, created_by, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		schedule.ID,
		schedule.Name,
		schedule.Frequency,
		schedule.Limit,
		schedule.Enabled,
		schedule.NextRunAt,
		schedule.LastRunAt,
		schedule.CreatedBy,
		schedule.CreatedAt,
		schedule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create scan schedule: %w", err)
	}
	return nil
}

// GetByID returns a scan schedule by its UUID.
func (r *PgScanScheduleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ScanSchedule, error) {
	query := `
		SELECT id, name, frequency, "limit", enabled, next_run_at, last_run_at, created_by, created_at, updated_at
		FROM scan_schedules
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var schedule models.ScanSchedule
	if err := row.Scan(
		&schedule.ID,
		&schedule.Name,
		&schedule.Frequency,
		&schedule.Limit,
		&schedule.Enabled,
		&schedule.NextRunAt,
		&schedule.LastRunAt,
		&schedule.CreatedBy,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrScanScheduleNotFound
		}
		return nil, fmt.Errorf("get scan schedule: %w", err)
	}
	return &schedule, nil
}

// List returns all scan schedules ordered by creation time descending.
func (r *PgScanScheduleRepository) List(ctx context.Context) ([]models.ScanSchedule, error) {
	query := `
		SELECT id, name, frequency, "limit", enabled, next_run_at, last_run_at, created_by, created_at, updated_at
		FROM scan_schedules
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list scan schedules: %w", err)
	}
	defer rows.Close()

	var schedules []models.ScanSchedule
	for rows.Next() {
		var schedule models.ScanSchedule
		if err := rows.Scan(
			&schedule.ID,
			&schedule.Name,
			&schedule.Frequency,
			&schedule.Limit,
			&schedule.Enabled,
			&schedule.NextRunAt,
			&schedule.LastRunAt,
			&schedule.CreatedBy,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scan schedules: %w", err)
	}
	return schedules, nil
}

// Update modifies an existing scan schedule.
func (r *PgScanScheduleRepository) Update(ctx context.Context, schedule *models.ScanSchedule) error {
	query := `
		UPDATE scan_schedules
		SET name = $2,
		    frequency = $3,
		    "limit" = $4,
		    enabled = $5,
		    next_run_at = $6,
		    last_run_at = $7,
		    updated_at = $8
		WHERE id = $1
	`
	res, err := r.db.ExecContext(ctx, query,
		schedule.ID,
		schedule.Name,
		schedule.Frequency,
		schedule.Limit,
		schedule.Enabled,
		schedule.NextRunAt,
		schedule.LastRunAt,
		schedule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update scan schedule: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrScanScheduleNotFound
	}
	return nil
}

// Delete removes a scan schedule by its UUID.
func (r *PgScanScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM scan_schedules WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete scan schedule: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrScanScheduleNotFound
	}
	return nil
}

// ListEnabledDue returns enabled schedules whose next run is at or before the given time.
func (r *PgScanScheduleRepository) ListEnabledDue(ctx context.Context, before time.Time) ([]models.ScanSchedule, error) {
	query := `
		SELECT id, name, frequency, "limit", enabled, next_run_at, last_run_at, created_by, created_at, updated_at
		FROM scan_schedules
		WHERE enabled = true AND next_run_at IS NOT NULL AND next_run_at <= $1
		ORDER BY next_run_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, before)
	if err != nil {
		return nil, fmt.Errorf("list due scan schedules: %w", err)
	}
	defer rows.Close()

	var schedules []models.ScanSchedule
	for rows.Next() {
		var schedule models.ScanSchedule
		if err := rows.Scan(
			&schedule.ID,
			&schedule.Name,
			&schedule.Frequency,
			&schedule.Limit,
			&schedule.Enabled,
			&schedule.NextRunAt,
			&schedule.LastRunAt,
			&schedule.CreatedBy,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due scan schedules: %w", err)
	}
	return schedules, nil
}

// CreateRun inserts a record of a scheduled scan execution.
func (r *PgScanScheduleRepository) CreateRun(ctx context.Context, run *models.ScanScheduleRun) error {
	query := `
		INSERT INTO scan_schedule_runs (
			id, schedule_id, scan_job_id, status, error_message, started_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		run.ID,
		run.ScheduleID,
		run.ScanJobID,
		run.Status,
		run.ErrorMessage,
		run.StartedAt,
		run.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create scan schedule run: %w", err)
	}
	return nil
}

// ListRunsBySchedule returns the run history for a schedule, newest first.
func (r *PgScanScheduleRepository) ListRunsBySchedule(ctx context.Context, scheduleID uuid.UUID) ([]models.ScanScheduleRun, error) {
	query := `
		SELECT id, schedule_id, scan_job_id, status, error_message, started_at, created_at
		FROM scan_schedule_runs
		WHERE schedule_id = $1
		ORDER BY started_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("list scan schedule runs: %w", err)
	}
	defer rows.Close()

	var runs []models.ScanScheduleRun
	for rows.Next() {
		var run models.ScanScheduleRun
		if err := rows.Scan(
			&run.ID,
			&run.ScheduleID,
			&run.ScanJobID,
			&run.Status,
			&run.ErrorMessage,
			&run.StartedAt,
			&run.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan schedule run: %w", err)
		}
		runs = append(runs, run)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scan schedule runs: %w", err)
	}
	return runs, nil
}
