package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"ulas-service/models"
)

// ErrScanJobNotFound is returned when a scan job cannot be located.
var ErrScanJobNotFound = errors.New("scan job not found")

// ScanRepository defines storage operations for scan jobs and results.
type ScanRepository interface {
	CreateScanJob(ctx context.Context, job *models.ScanJob) error
	GetScanJobByID(ctx context.Context, id uuid.UUID) (*models.ScanJob, error)
	GetScanJobByAnsibleJobID(ctx context.Context, ansibleJobID string) (*models.ScanJob, error)
	ListScanJobs(ctx context.Context) ([]models.ScanJob, error)
	ListScanJobsPaginated(ctx context.Context, page, limit int) ([]models.ScanJob, int, error)
	ListScanJobsPaginatedWithDeviationCounts(ctx context.Context, page, limit int, onlyWithDeviations bool, search string, fromDate, toDate *time.Time) ([]models.ScanJobSummary, int, error)
	HasActiveScanJob(ctx context.Context) (bool, error)
	UpdateScanJob(ctx context.Context, job *models.ScanJob) error
	UpdateScanJobStatus(ctx context.Context, job *models.ScanJob) error
	UpdateScanJobLaunch(ctx context.Context, job *models.ScanJob) error
	AppendFailedHosts(ctx context.Context, id uuid.UUID, hostnames []string) error

	CreateScanResult(ctx context.Context, result *models.ScanResult) error
	GetScanResult(ctx context.Context, id uuid.UUID) (*models.ScanResult, error)
	GetScanResultByJobAndHost(ctx context.Context, scanJobID, hostID uuid.UUID) (*models.ScanResult, error)
	ListScanResultsByJobID(ctx context.Context, scanJobID uuid.UUID) ([]models.ScanResult, error)
	ListScanResultsByJobIDPaginated(ctx context.Context, scanJobID uuid.UUID, page, limit int) ([]models.ScanResult, int, error)
	UpdateScanResult(ctx context.Context, result *models.ScanResult) error

	IncrementScanJobCounters(ctx context.Context, id uuid.UUID, callbacks, successful, failed int) error
}

// PgScanRepository is a PostgreSQL implementation of ScanRepository.
type PgScanRepository struct {
	db *sql.DB
}

// NewPgScanRepository creates a new PostgreSQL scan repository.
func NewPgScanRepository(db *sql.DB) *PgScanRepository {
	return &PgScanRepository{db: db}
}

// CreateScanJob inserts a new scan job.
func (r *PgScanRepository) CreateScanJob(ctx context.Context, job *models.ScanJob) error {
	query := `
		INSERT INTO scan_jobs (
			id, ansible_job_id, job_template_id, os_type, "limit", status, initiated_by,
			started_at, completed_at, total_hosts, callbacks_received,
			successful_hosts, failed_hosts, error_message, baseline_snapshot, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	_, err := r.db.ExecContext(ctx, query,
		job.ID,
		job.AnsibleJobID,
		job.JobTemplateID,
		job.OSType,
		job.Limit,
		job.Status,
		job.InitiatedBy,
		job.StartedAt,
		job.CompletedAt,
		job.TotalHosts,
		job.CallbacksReceived,
		job.SuccessfulHosts,
		job.FailedHosts,
		job.ErrorMessage,
		baselineSnapshotValue(job.BaselineSnapshot),
		job.CreatedAt,
		job.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create scan job: %w", err)
	}
	return nil
}

// GetScanJobByID returns a scan job by its UUID.
func (r *PgScanRepository) GetScanJobByID(ctx context.Context, id uuid.UUID) (*models.ScanJob, error) {
	query := `
		SELECT id, ansible_job_id, job_template_id, COALESCE(os_type, 'linux') AS os_type, "limit", status, initiated_by,
		       started_at, completed_at, total_hosts, callbacks_received,
		       successful_hosts, failed_hosts, failed_host_names, error_message, baseline_snapshot, created_at, updated_at
		FROM scan_jobs
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var job models.ScanJob
	if err := row.Scan(
		&job.ID,
		&job.AnsibleJobID,
		&job.JobTemplateID,
		&job.OSType,
		&job.Limit,
		&job.Status,
		&job.InitiatedBy,
		&job.StartedAt,
		&job.CompletedAt,
		&job.TotalHosts,
		&job.CallbacksReceived,
		&job.SuccessfulHosts,
		&job.FailedHosts,
		stringSliceScanner{&job.FailedHostNames},
		&job.ErrorMessage,
		baselineSnapshotScanner{&job.BaselineSnapshot},
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrScanJobNotFound
		}
		return nil, fmt.Errorf("get scan job by id: %w", err)
	}
	return &job, nil
}

// GetScanJobByAnsibleJobID returns a scan job by its AAP job ID.
func (r *PgScanRepository) GetScanJobByAnsibleJobID(ctx context.Context, ansibleJobID string) (*models.ScanJob, error) {
	query := `
		SELECT id, ansible_job_id, job_template_id, COALESCE(os_type, 'linux') AS os_type, "limit", status, initiated_by,
		       started_at, completed_at, total_hosts, callbacks_received,
		       successful_hosts, failed_hosts, failed_host_names, error_message, baseline_snapshot, created_at, updated_at
		FROM scan_jobs
		WHERE ansible_job_id = $1
	`
	row := r.db.QueryRowContext(ctx, query, ansibleJobID)

	var job models.ScanJob
	if err := row.Scan(
		&job.ID,
		&job.AnsibleJobID,
		&job.JobTemplateID,
		&job.OSType,
		&job.Limit,
		&job.Status,
		&job.InitiatedBy,
		&job.StartedAt,
		&job.CompletedAt,
		&job.TotalHosts,
		&job.CallbacksReceived,
		&job.SuccessfulHosts,
		&job.FailedHosts,
		stringSliceScanner{&job.FailedHostNames},
		&job.ErrorMessage,
		baselineSnapshotScanner{&job.BaselineSnapshot},
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrScanJobNotFound
		}
		return nil, fmt.Errorf("get scan job by ansible job id: %w", err)
	}
	return &job, nil
}

// ListScanJobs returns all scan jobs ordered by creation time descending.
func (r *PgScanRepository) ListScanJobs(ctx context.Context) ([]models.ScanJob, error) {
	query := `
		SELECT id, ansible_job_id, job_template_id, COALESCE(os_type, 'linux') AS os_type, "limit", status, initiated_by,
		       started_at, completed_at, total_hosts, callbacks_received,
		       successful_hosts, failed_hosts, failed_host_names, error_message, baseline_snapshot, created_at, updated_at
		FROM scan_jobs
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list scan jobs: %w", err)
	}
	defer rows.Close()

	var jobs []models.ScanJob
	for rows.Next() {
		var job models.ScanJob
		if err := rows.Scan(
			&job.ID,
			&job.AnsibleJobID,
			&job.JobTemplateID,
			&job.OSType,
			&job.Limit,
			&job.Status,
			&job.InitiatedBy,
			&job.StartedAt,
			&job.CompletedAt,
			&job.TotalHosts,
			&job.CallbacksReceived,
			&job.SuccessfulHosts,
			&job.FailedHosts,
			stringSliceScanner{&job.FailedHostNames},
			&job.ErrorMessage,
			baselineSnapshotScanner{&job.BaselineSnapshot},
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan scan job: %w", err)
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scan jobs: %w", err)
	}
	return jobs, nil
}

// ListScanJobsPaginated returns a page of scan jobs plus the total count.
func (r *PgScanRepository) ListScanJobsPaginated(ctx context.Context, page, limit int) ([]models.ScanJob, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scan_jobs`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count scan jobs: %w", err)
	}

	query := `
		SELECT id, ansible_job_id, job_template_id, COALESCE(os_type, 'linux') AS os_type, "limit", status, initiated_by,
		       started_at, completed_at, total_hosts, callbacks_received,
		       successful_hosts, failed_hosts, failed_host_names, error_message, baseline_snapshot, created_at, updated_at
		FROM scan_jobs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list scan jobs paginated: %w", err)
	}
	defer rows.Close()

	var jobs []models.ScanJob
	for rows.Next() {
		var job models.ScanJob
		if err := rows.Scan(
			&job.ID,
			&job.AnsibleJobID,
			&job.JobTemplateID,
			&job.OSType,
			&job.Limit,
			&job.Status,
			&job.InitiatedBy,
			&job.StartedAt,
			&job.CompletedAt,
			&job.TotalHosts,
			&job.CallbacksReceived,
			&job.SuccessfulHosts,
			&job.FailedHosts,
			stringSliceScanner{&job.FailedHostNames},
			&job.ErrorMessage,
			baselineSnapshotScanner{&job.BaselineSnapshot},
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan scan job: %w", err)
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate scan jobs: %w", err)
	}
	return jobs, total, nil
}

// ListScanJobsPaginatedWithDeviationCounts returns scan jobs with their total
// unauthorized and allowed deviation counts. When onlyWithDeviations is true,
// only jobs with at least one deviation (unauthorized or allowed) are returned.
// The optional search term matches id, ansible_job_id, limit, initiated_by, or error_message.
func (r *PgScanRepository) ListScanJobsPaginatedWithDeviationCounts(ctx context.Context, page, limit int, onlyWithDeviations bool, search string, fromDate, toDate *time.Time) ([]models.ScanJobSummary, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	deviationSum := `
		COALESCE((
			SELECT SUM(deviations_found)
			FROM scan_results
			WHERE scan_job_id = j.id
		), 0)
	`
	allowedSum := `
		COALESCE((
			SELECT SUM(COALESCE(JSONB_ARRAY_LENGTH(allowed_deviations), 0))
			FROM scan_results
			WHERE scan_job_id = j.id
		), 0)
	`

	where := " WHERE 1=1"
	var args []interface{}
	if onlyWithDeviations {
		where += fmt.Sprintf(` AND (%s + %s) > 0`, deviationSum, allowedSum)
	}
	if search != "" {
		where += ` AND (j.id::text ILIKE $1 OR j.ansible_job_id ILIKE $1 OR j."limit" ILIKE $1 OR j.initiated_by ILIKE $1 OR j.error_message ILIKE $1)`
		args = append(args, "%"+search+"%")
	}
	if fromDate != nil {
		where += fmt.Sprintf(` AND j.created_at >= $%d`, len(args)+1)
		args = append(args, *fromDate)
	}
	if toDate != nil {
		where += fmt.Sprintf(` AND j.created_at <= $%d`, len(args)+1)
		args = append(args, *toDate)
	}

	countQuery := "SELECT COUNT(*) FROM scan_jobs j" + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count scan jobs with deviations: %w", err)
	}

	query := `
		SELECT j.id, j.ansible_job_id, j.job_template_id, COALESCE(j.os_type, 'linux') AS os_type, j."limit", j.status, j.initiated_by,
		       j.started_at, j.completed_at, j.total_hosts, j.callbacks_received,
		       j.successful_hosts, j.failed_hosts, j.failed_host_names, j.error_message, j.baseline_snapshot, j.created_at, j.updated_at,
		       ` + deviationSum + ` AS total_deviations,
		       ` + allowedSum + ` AS total_allowed_deviations
		FROM scan_jobs j
	` + where + `
		ORDER BY j.created_at DESC
		LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)

	rows, err := r.db.QueryContext(ctx, query, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("list scan jobs with deviation counts: %w", err)
	}
	defer rows.Close()

	var jobs []models.ScanJobSummary
	for rows.Next() {
		var job models.ScanJobSummary
		if err := rows.Scan(
			&job.ID,
			&job.AnsibleJobID,
			&job.JobTemplateID,
			&job.OSType,
			&job.Limit,
			&job.Status,
			&job.InitiatedBy,
			&job.StartedAt,
			&job.CompletedAt,
			&job.TotalHosts,
			&job.CallbacksReceived,
			&job.SuccessfulHosts,
			&job.FailedHosts,
			stringSliceScanner{&job.FailedHostNames},
			&job.ErrorMessage,
			baselineSnapshotScanner{&job.BaselineSnapshot},
			&job.CreatedAt,
			&job.UpdatedAt,
			&job.TotalDeviations,
			&job.TotalAllowedDeviations,
		); err != nil {
			return nil, 0, fmt.Errorf("scan scan job with deviation counts: %w", err)
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate scan jobs with deviation counts: %w", err)
	}
	return jobs, total, nil
}

// HasActiveScanJob returns true if any scan job is currently initiating or running.
func (r *PgScanRepository) HasActiveScanJob(ctx context.Context) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM scan_jobs
			WHERE status IN ($1, $2)
		)
	`
	var exists bool
	if err := r.db.QueryRowContext(
		ctx,
		query,
		models.ScanJobStatusInitiating,
		models.ScanJobStatusRunning,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check active scan job: %w", err)
	}
	return exists, nil
}

// UpdateScanJob modifies an existing scan job.
func (r *PgScanRepository) UpdateScanJob(ctx context.Context, job *models.ScanJob) error {
	query := `
		UPDATE scan_jobs
		SET ansible_job_id = $2,
		    job_template_id = $3,
		    os_type = $4,
		    "limit" = $5,
		    status = $6,
		    initiated_by = $7,
		    started_at = $8,
		    completed_at = $9,
		    total_hosts = $10,
		    callbacks_received = $11,
		    successful_hosts = $12,
		    failed_hosts = $13,
		    error_message = $14,
		    baseline_snapshot = $15,
		    updated_at = $16
		WHERE id = $1
	`
	res, err := r.db.ExecContext(ctx, query,
		job.ID,
		job.AnsibleJobID,
		job.JobTemplateID,
		job.OSType,
		job.Limit,
		job.Status,
		job.InitiatedBy,
		job.StartedAt,
		job.CompletedAt,
		job.TotalHosts,
		job.CallbacksReceived,
		job.SuccessfulHosts,
		job.FailedHosts,
		job.ErrorMessage,
		baselineSnapshotValue(job.BaselineSnapshot),
		job.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update scan job: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrScanJobNotFound
	}
	return nil
}

// CreateScanResult inserts a new scan result.
func (r *PgScanRepository) CreateScanResult(ctx context.Context, result *models.ScanResult) error {
	query := `
		INSERT INTO scan_results (
			id, scan_job_id, host_id, status, error_message, processing_status,
			deviations_found, allowed_deviations, baseline_version_at_scan, no_baseline,
			received_at, processed_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.ExecContext(ctx, query,
		result.ID,
		result.ScanJobID,
		result.HostID,
		result.Status,
		result.ErrorMessage,
		result.ProcessingStatus,
		result.DeviationsFound,
		result.AllowedDeviations,
		result.BaselineVersionAtScan,
		result.NoBaseline,
		result.ReceivedAt,
		result.ProcessedAt,
		result.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create scan result: %w", err)
	}
	return nil
}

// GetScanResult returns a scan result by its UUID.
func (r *PgScanRepository) GetScanResult(ctx context.Context, id uuid.UUID) (*models.ScanResult, error) {
	query := `
		SELECT id, scan_job_id, host_id, status, error_message, processing_status,
		       deviations_found, allowed_deviations, baseline_version_at_scan, no_baseline,
		       received_at, processed_at, created_at
		FROM scan_results
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var result models.ScanResult
	if err := row.Scan(
		&result.ID,
		&result.ScanJobID,
		&result.HostID,
		&result.Status,
		&result.ErrorMessage,
		&result.ProcessingStatus,
		&result.DeviationsFound,
		&result.AllowedDeviations,
		&result.BaselineVersionAtScan,
		&result.NoBaseline,
		&result.ReceivedAt,
		&result.ProcessedAt,
		&result.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("scan result not found")
		}
		return nil, fmt.Errorf("get scan result: %w", err)
	}
	return &result, nil
}

// GetScanResultByJobAndHost returns a scan result for a specific job and host.
func (r *PgScanRepository) GetScanResultByJobAndHost(ctx context.Context, scanJobID, hostID uuid.UUID) (*models.ScanResult, error) {
	query := `
		SELECT id, scan_job_id, host_id, status, error_message, processing_status,
		       deviations_found, allowed_deviations, baseline_version_at_scan, no_baseline,
		       received_at, processed_at, created_at
		FROM scan_results
		WHERE scan_job_id = $1 AND host_id = $2
	`
	row := r.db.QueryRowContext(ctx, query, scanJobID, hostID)

	var result models.ScanResult
	if err := row.Scan(
		&result.ID,
		&result.ScanJobID,
		&result.HostID,
		&result.Status,
		&result.ErrorMessage,
		&result.ProcessingStatus,
		&result.DeviationsFound,
		&result.AllowedDeviations,
		&result.BaselineVersionAtScan,
		&result.NoBaseline,
		&result.ReceivedAt,
		&result.ProcessedAt,
		&result.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("scan result not found")
		}
		return nil, fmt.Errorf("get scan result: %w", err)
	}
	return &result, nil
}

// ListScanResultsByJobID returns all scan results for a scan job.
func (r *PgScanRepository) ListScanResultsByJobID(ctx context.Context, scanJobID uuid.UUID) ([]models.ScanResult, error) {
	query := `
		SELECT id, scan_job_id, host_id, status, error_message, processing_status,
		       deviations_found, allowed_deviations, baseline_version_at_scan, no_baseline,
		       received_at, processed_at, created_at
		FROM scan_results
		WHERE scan_job_id = $1
		ORDER BY created_at
	`
	rows, err := r.db.QueryContext(ctx, query, scanJobID)
	if err != nil {
		return nil, fmt.Errorf("list scan results: %w", err)
	}
	defer rows.Close()

	var results []models.ScanResult
	for rows.Next() {
		var result models.ScanResult
		if err := rows.Scan(
			&result.ID,
			&result.ScanJobID,
			&result.HostID,
			&result.Status,
			&result.ErrorMessage,
			&result.ProcessingStatus,
			&result.DeviationsFound,
			&result.AllowedDeviations,
			&result.BaselineVersionAtScan,
			&result.NoBaseline,
			&result.ReceivedAt,
			&result.ProcessedAt,
			&result.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan scan result: %w", err)
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scan results: %w", err)
	}
	return results, nil
}

// ListScanResultsByJobIDPaginated returns a page of scan results plus the total count.
func (r *PgScanRepository) ListScanResultsByJobIDPaginated(ctx context.Context, scanJobID uuid.UUID, page, limit int) ([]models.ScanResult, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	offset := (page - 1) * limit

	var total int
	countQuery := `SELECT COUNT(*) FROM scan_results WHERE scan_job_id = $1`
	if err := r.db.QueryRowContext(ctx, countQuery, scanJobID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count scan results: %w", err)
	}

	query := `
		SELECT id, scan_job_id, host_id, status, error_message, processing_status,
		       deviations_found, allowed_deviations, baseline_version_at_scan, no_baseline,
		       received_at, processed_at, created_at
		FROM scan_results
		WHERE scan_job_id = $1
		ORDER BY created_at
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, scanJobID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list scan results paginated: %w", err)
	}
	defer rows.Close()

	var results []models.ScanResult
	for rows.Next() {
		var result models.ScanResult
		if err := rows.Scan(
			&result.ID,
			&result.ScanJobID,
			&result.HostID,
			&result.Status,
			&result.ErrorMessage,
			&result.ProcessingStatus,
			&result.DeviationsFound,
			&result.AllowedDeviations,
			&result.BaselineVersionAtScan,
			&result.NoBaseline,
			&result.ReceivedAt,
			&result.ProcessedAt,
			&result.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan scan result: %w", err)
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate scan results: %w", err)
	}
	return results, total, nil
}

// UpdateScanJobLaunch updates fields set after a successful AAP launch, preserving counters.
func (r *PgScanRepository) UpdateScanJobLaunch(ctx context.Context, job *models.ScanJob) error {
	query := `
		UPDATE scan_jobs
		SET ansible_job_id = $2,
		    status = $3,
		    started_at = $4,
		    updated_at = $5
		WHERE id = $1
	`
	res, err := r.db.ExecContext(ctx, query,
		job.ID,
		job.AnsibleJobID,
		job.Status,
		job.StartedAt,
		job.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update scan job launch: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("scan job not found")
	}
	return nil
}

// UpdateScanJobStatus updates only status-related fields, preserving counters.
func (r *PgScanRepository) UpdateScanJobStatus(ctx context.Context, job *models.ScanJob) error {
	query := `
		UPDATE scan_jobs
		SET status = $2,
		    error_message = $3,
		    completed_at = $4,
		    updated_at = $5
		WHERE id = $1
	`
	res, err := r.db.ExecContext(ctx, query,
		job.ID,
		job.Status,
		job.ErrorMessage,
		job.CompletedAt,
		job.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update scan job status: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("scan job not found")
	}
	return nil
}

// AppendFailedHosts appends hostnames to the failed_host_names array and increments failed_hosts.
func (r *PgScanRepository) AppendFailedHosts(ctx context.Context, id uuid.UUID, hostnames []string) error {
	if len(hostnames) == 0 {
		return nil
	}

	data, err := json.Marshal(hostnames)
	if err != nil {
		return fmt.Errorf("marshal failed host names: %w", err)
	}

	query := `
		UPDATE scan_jobs
		SET failed_host_names = COALESCE(failed_host_names, '[]'::jsonb) || $2::jsonb,
		    failed_hosts = failed_hosts + $3,
		    updated_at = $4
		WHERE id = $1
	`
	res, err := r.db.ExecContext(ctx, query, id, data, len(hostnames), time.Now().UTC())
	if err != nil {
		return fmt.Errorf("append failed hosts: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("scan job not found")
	}
	return nil
}

// IncrementScanJobCounters atomically adjusts the callback/success/failure counters.
func (r *PgScanRepository) IncrementScanJobCounters(ctx context.Context, id uuid.UUID, callbacks, successful, failed int) error {
	query := `
		UPDATE scan_jobs
		SET callbacks_received = callbacks_received + $2,
		    successful_hosts = successful_hosts + $3,
		    failed_hosts = failed_hosts + $4,
		    updated_at = $5
		WHERE id = $1
	`
	res, err := r.db.ExecContext(ctx, query, id, callbacks, successful, failed, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("increment scan job counters: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("scan job not found")
	}
	return nil
}

// UpdateScanResult modifies an existing scan result.
func (r *PgScanRepository) UpdateScanResult(ctx context.Context, result *models.ScanResult) error {
	query := `
		UPDATE scan_results
		SET status = $2,
		    error_message = $3,
		    processing_status = $4,
		    deviations_found = $5,
		    allowed_deviations = $6,
		    baseline_version_at_scan = $7,
		    no_baseline = $8,
		    received_at = $9,
		    processed_at = $10
		WHERE id = $1
	`
	res, err := r.db.ExecContext(ctx, query,
		result.ID,
		result.Status,
		result.ErrorMessage,
		result.ProcessingStatus,
		result.DeviationsFound,
		result.AllowedDeviations,
		result.BaselineVersionAtScan,
		result.NoBaseline,
		result.ReceivedAt,
		result.ProcessedAt,
	)
	if err != nil {
		return fmt.Errorf("update scan result: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("scan result not found")
	}
	return nil
}

// baselineSnapshotValue marshals a baseline snapshot for storage as JSONB.
func baselineSnapshotValue(snapshot []models.BaselineVersionSnapshot) interface{} {
	if len(snapshot) == 0 {
		return nil
	}
	b, err := json.Marshal(snapshot)
	if err != nil {
		return nil
	}
	return b
}

// baselineSnapshotScanner scans a JSONB column into a baseline snapshot slice.
type baselineSnapshotScanner struct {
	snapshot *[]models.BaselineVersionSnapshot
}

func (s baselineSnapshotScanner) Scan(value interface{}) error {
	if value == nil {
		*s.snapshot = nil
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan baseline_snapshot: unexpected type %T", value)
	}
	if len(data) == 0 {
		*s.snapshot = nil
		return nil
	}
	return json.Unmarshal(data, s.snapshot)
}

// stringSliceScanner scans a JSONB text array column into a string slice.
type stringSliceScanner struct {
	slice *[]string
}

func (s stringSliceScanner) Scan(value interface{}) error {
	if value == nil {
		*s.slice = nil
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan string slice: unexpected type %T", value)
	}
	if len(data) == 0 {
		*s.slice = nil
		return nil
	}
	return json.Unmarshal(data, s.slice)
}
