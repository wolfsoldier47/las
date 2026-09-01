package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"ulas-service/models"
)

// ErrIncidentNotFound is returned when an incident cannot be located.
var ErrIncidentNotFound = errors.New("incident not found")

// IncidentRepository defines storage operations for incidents.
type IncidentRepository interface {
	Create(ctx context.Context, incident *models.Incident) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Incident, error)
	List(ctx context.Context, filters IncidentFilters) ([]models.Incident, error)
	Update(ctx context.Context, incident *models.Incident) error
}

// IncidentFilters contains optional filters for listing incidents.
type IncidentFilters struct {
	HostID       *uuid.UUID
	Status       *models.IncidentStatus
	ScanResultID *uuid.UUID
	ScanJobID    *uuid.UUID
}

// PgIncidentRepository is a PostgreSQL implementation of IncidentRepository.
type PgIncidentRepository struct {
	db *sql.DB
}

// NewPgIncidentRepository creates a new PostgreSQL incident repository.
func NewPgIncidentRepository(db *sql.DB) *PgIncidentRepository {
	return &PgIncidentRepository{db: db}
}

// Create inserts a new incident.
func (r *PgIncidentRepository) Create(ctx context.Context, incident *models.Incident) error {
	query := `
		INSERT INTO incidents (
			id, incident_number, scan_result_id, host_id, file_type, entry_key, expected_value,
			actual_value, baseline_version_at_scan, severity, status, notes,
			service_now_ticket_opened, resolution, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`
	_, err := r.db.ExecContext(ctx, query,
		incident.ID,
		incident.IncidentNumber,
		incident.ScanResultID,
		incident.HostID,
		incident.FileType,
		incident.EntryKey,
		incident.ExpectedValue,
		incident.ActualValue,
		incident.BaselineVersionAtScan,
		incident.Severity,
		incident.Status,
		incident.Notes,
		incident.ServiceNowTicketOpened,
		incident.Resolution,
		incident.CreatedAt,
		incident.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create incident: %w", err)
	}
	return nil
}

// GetByID returns an incident by its UUID.
func (r *PgIncidentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Incident, error) {
	query := `
		SELECT id, incident_number, scan_result_id, host_id, file_type, entry_key, expected_value,
		       actual_value, baseline_version_at_scan, severity, status, notes,
		       service_now_ticket_opened, resolution, created_at, updated_at
		FROM incidents
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var incident models.Incident
	if err := row.Scan(
		&incident.ID,
		&incident.IncidentNumber,
		&incident.ScanResultID,
		&incident.HostID,
		&incident.FileType,
		&incident.EntryKey,
		&incident.ExpectedValue,
		&incident.ActualValue,
		&incident.BaselineVersionAtScan,
		&incident.Severity,
		&incident.Status,
		&incident.Notes,
		&incident.ServiceNowTicketOpened,
		&incident.Resolution,
		&incident.CreatedAt,
		&incident.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrIncidentNotFound
		}
		return nil, fmt.Errorf("get incident by id: %w", err)
	}
	return &incident, nil
}

// List returns incidents matching the provided filters.
func (r *PgIncidentRepository) List(ctx context.Context, filters IncidentFilters) ([]models.Incident, error) {
	query := `
		SELECT i.id, i.incident_number, i.scan_result_id, i.host_id, i.file_type, i.entry_key, i.expected_value,
		       i.actual_value, i.baseline_version_at_scan, i.severity, i.status, i.notes,
		       i.service_now_ticket_opened, i.resolution, i.created_at, i.updated_at
		FROM incidents i
		WHERE 1=1
	`
	var args []interface{}
	var argCount int

	if filters.HostID != nil {
		argCount++
		query += fmt.Sprintf(" AND i.host_id = $%d", argCount)
		args = append(args, *filters.HostID)
	}
	if filters.Status != nil {
		argCount++
		query += fmt.Sprintf(" AND i.status = $%d", argCount)
		args = append(args, *filters.Status)
	}
	if filters.ScanResultID != nil {
		argCount++
		query += fmt.Sprintf(" AND i.scan_result_id = $%d", argCount)
		args = append(args, *filters.ScanResultID)
	}
	if filters.ScanJobID != nil {
		argCount++
		query += fmt.Sprintf(" AND i.scan_result_id IN (SELECT id FROM scan_results WHERE scan_job_id = $%d)", argCount)
		args = append(args, *filters.ScanJobID)
	}

	query += " ORDER BY i.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list incidents: %w", err)
	}
	defer rows.Close()

	var incidents []models.Incident
	for rows.Next() {
		var incident models.Incident
		if err := rows.Scan(
			&incident.ID,
			&incident.IncidentNumber,
			&incident.ScanResultID,
			&incident.HostID,
			&incident.FileType,
			&incident.EntryKey,
			&incident.ExpectedValue,
			&incident.ActualValue,
			&incident.BaselineVersionAtScan,
			&incident.Severity,
			&incident.Status,
			&incident.Notes,
			&incident.ServiceNowTicketOpened,
			&incident.Resolution,
			&incident.CreatedAt,
			&incident.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan incident: %w", err)
		}
		incidents = append(incidents, incident)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate incidents: %w", err)
	}
	return incidents, nil
}

// Update modifies an existing incident.
func (r *PgIncidentRepository) Update(ctx context.Context, incident *models.Incident) error {
	query := `
		UPDATE incidents
		SET incident_number = $2,
		    scan_result_id = $3,
		    host_id = $4,
		    file_type = $5,
		    entry_key = $6,
		    expected_value = $7,
		    actual_value = $8,
		    baseline_version_at_scan = $9,
		    severity = $10,
		    status = $11,
		    notes = $12,
		    service_now_ticket_opened = $13,
		    resolution = $14,
		    updated_at = $15
		WHERE id = $1
	`
	res, err := r.db.ExecContext(ctx, query,
		incident.ID,
		incident.IncidentNumber,
		incident.ScanResultID,
		incident.HostID,
		incident.FileType,
		incident.EntryKey,
		incident.ExpectedValue,
		incident.ActualValue,
		incident.BaselineVersionAtScan,
		incident.Severity,
		incident.Status,
		incident.Notes,
		incident.ServiceNowTicketOpened,
		incident.Resolution,
		incident.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update incident: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrIncidentNotFound
	}
	return nil
}
