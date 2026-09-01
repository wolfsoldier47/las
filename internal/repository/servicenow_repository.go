package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"ulas-service/models"
)

// ErrServiceNowTicketNotFound is returned when a ticket cannot be located.
var ErrServiceNowTicketNotFound = errors.New("service now ticket not found")

// ServiceNowTicketRepository defines storage operations for ServiceNow tickets.
type ServiceNowTicketRepository interface {
	Create(ctx context.Context, ticket *models.ServiceNowTicket) error
	GetByHostAndScanJob(ctx context.Context, hostID, scanJobID uuid.UUID) (*models.ServiceNowTicket, error)
	GetIncidentIDs(ctx context.Context, ticketID uuid.UUID) ([]uuid.UUID, error)
	LinkIncident(ctx context.Context, ticketID, incidentID uuid.UUID) error
	Update(ctx context.Context, ticket *models.ServiceNowTicket) error
}

// PgServiceNowTicketRepository is a PostgreSQL implementation of ServiceNowTicketRepository.
type PgServiceNowTicketRepository struct {
	db *sql.DB
}

// NewPgServiceNowTicketRepository creates a new PostgreSQL ServiceNow ticket repository.
func NewPgServiceNowTicketRepository(db *sql.DB) *PgServiceNowTicketRepository {
	return &PgServiceNowTicketRepository{db: db}
}

// Create inserts a new ServiceNow ticket record and links it to the provided incidents.
func (r *PgServiceNowTicketRepository) Create(ctx context.Context, ticket *models.ServiceNowTicket) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO service_now_tickets (
			id, host_id, scan_job_id, ticket_number, ticket_url,
			ticket_opened, opened_at, error_message, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	if _, err := tx.ExecContext(ctx, query,
		ticket.ID,
		ticket.HostID,
		ticket.ScanJobID,
		ticket.TicketNumber,
		ticket.TicketURL,
		ticket.TicketOpened,
		ticket.OpenedAt,
		ticket.ErrorMessage,
		ticket.CreatedAt,
	); err != nil {
		return fmt.Errorf("create service now ticket: %w", err)
	}

	if len(ticket.IncidentIDs) > 0 {
		if err := r.linkIncidentsTx(ctx, tx, ticket.ID, ticket.IncidentIDs); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create service now ticket: %w", err)
	}
	return nil
}

// GetByHostAndScanJob returns a ServiceNow ticket for a host and scan job, if one exists.
func (r *PgServiceNowTicketRepository) GetByHostAndScanJob(ctx context.Context, hostID, scanJobID uuid.UUID) (*models.ServiceNowTicket, error) {
	query := `
		SELECT id, host_id, scan_job_id, ticket_number, ticket_url,
		       ticket_opened, opened_at, error_message, created_at
		FROM service_now_tickets
		WHERE host_id = $1 AND scan_job_id = $2
	`
	row := r.db.QueryRowContext(ctx, query, hostID, scanJobID)

	var ticket models.ServiceNowTicket
	if err := row.Scan(
		&ticket.ID,
		&ticket.HostID,
		&ticket.ScanJobID,
		&ticket.TicketNumber,
		&ticket.TicketURL,
		&ticket.TicketOpened,
		&ticket.OpenedAt,
		&ticket.ErrorMessage,
		&ticket.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrServiceNowTicketNotFound
		}
		return nil, fmt.Errorf("get service now ticket by host and scan job: %w", err)
	}

	incidentIDs, err := r.GetIncidentIDs(ctx, ticket.ID)
	if err != nil {
		return nil, err
	}
	ticket.IncidentIDs = incidentIDs

	return &ticket, nil
}

// GetIncidentIDs returns the incident IDs linked to a ticket.
func (r *PgServiceNowTicketRepository) GetIncidentIDs(ctx context.Context, ticketID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT incident_id
		FROM service_now_ticket_incidents
		WHERE service_now_ticket_id = $1
		ORDER BY incident_id
	`
	rows, err := r.db.QueryContext(ctx, query, ticketID)
	if err != nil {
		return nil, fmt.Errorf("get linked incidents: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan incident id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate incident ids: %w", err)
	}
	return ids, nil
}

// LinkIncident associates a single incident with a ticket.
func (r *PgServiceNowTicketRepository) LinkIncident(ctx context.Context, ticketID, incidentID uuid.UUID) error {
	return r.linkIncidentsTx(ctx, nil, ticketID, []uuid.UUID{incidentID})
}

func (r *PgServiceNowTicketRepository) linkIncidentsTx(ctx context.Context, tx *sql.Tx, ticketID uuid.UUID, incidentIDs []uuid.UUID) error {
	query := `
		INSERT INTO service_now_ticket_incidents (service_now_ticket_id, incident_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`

	exec := r.db.ExecContext
	if tx != nil {
		exec = tx.ExecContext
	}

	for _, id := range incidentIDs {
		if _, err := exec(ctx, query, ticketID, id); err != nil {
			return fmt.Errorf("link incident %s to ticket %s: %w", id, ticketID, err)
		}
	}
	return nil
}

// Update modifies an existing ServiceNow ticket record.
func (r *PgServiceNowTicketRepository) Update(ctx context.Context, ticket *models.ServiceNowTicket) error {
	query := `
		UPDATE service_now_tickets
		SET ticket_number = $2,
		    ticket_url = $3,
		    ticket_opened = $4,
		    opened_at = $5,
		    error_message = $6
		WHERE id = $1
	`
	res, err := r.db.ExecContext(ctx, query,
		ticket.ID,
		ticket.TicketNumber,
		ticket.TicketURL,
		ticket.TicketOpened,
		ticket.OpenedAt,
		ticket.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("update service now ticket: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrServiceNowTicketNotFound
	}
	return nil
}
