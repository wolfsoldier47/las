package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"ulas-service/internal/repository"
	"ulas-service/internal/servicenow"
	"ulas-service/models"
)

// ServiceNowService defines operations for opening ServiceNow tickets.
type ServiceNowService interface {
	OpenTicket(ctx context.Context, incidentID uuid.UUID) (*models.ServiceNowTicket, error)
	BulkOpenTickets(ctx context.Context, incidentIDs []uuid.UUID) ([]models.ServiceNowTicket, error)
}

// DefaultServiceNowService is the default implementation of ServiceNowService.
type DefaultServiceNowService struct {
	incidentRepo repository.IncidentRepository
	ticketRepo   repository.ServiceNowTicketRepository
	scanRepo     repository.ScanRepository
	hostRepo     repository.HostRepository
	client       *servicenow.Client
}

// NewDefaultServiceNowService creates a new ServiceNow service.
func NewDefaultServiceNowService(
	incidentRepo repository.IncidentRepository,
	ticketRepo repository.ServiceNowTicketRepository,
	scanRepo repository.ScanRepository,
	hostRepo repository.HostRepository,
	client *servicenow.Client,
) *DefaultServiceNowService {
	return &DefaultServiceNowService{
		incidentRepo: incidentRepo,
		ticketRepo:   ticketRepo,
		scanRepo:     scanRepo,
		hostRepo:     hostRepo,
		client:       client,
	}
}

// OpenTicket opens a ServiceNow incident for the host and scan job that the
// given ulas incident belongs to. All incidents for the same host in the same
// scan job are grouped into a single ServiceNow ticket.
func (s *DefaultServiceNowService) OpenTicket(ctx context.Context, incidentID uuid.UUID) (*models.ServiceNowTicket, error) {
	incident, err := s.incidentRepo.GetByID(ctx, incidentID)
	if err != nil {
		return nil, fmt.Errorf("get incident: %w", err)
	}

	result, err := s.scanRepo.GetScanResult(ctx, incident.ScanResultID)
	if err != nil {
		return nil, fmt.Errorf("get scan result: %w", err)
	}

	ticket, err := s.openTicketForHostScan(ctx, incident.HostID, result.ScanJobID)
	if err != nil {
		return nil, fmt.Errorf("open ticket for host scan: %w", err)
	}

	return ticket, nil
}

// openTicketForHostScan creates or returns an existing ServiceNow ticket for a
// host within a scan job. All unticketed incidents for that host in that scan
// job are grouped into the ticket.
func (s *DefaultServiceNowService) openTicketForHostScan(ctx context.Context, hostID, scanJobID uuid.UUID) (*models.ServiceNowTicket, error) {
	// Return the existing ticket if one has already been successfully opened for this host+scan.
	existing, err := s.ticketRepo.GetByHostAndScanJob(ctx, hostID, scanJobID)
	if err == nil && existing.TicketNumber != nil {
		return existing, nil
	}
	if !errors.Is(err, repository.ErrServiceNowTicketNotFound) {
		return nil, fmt.Errorf("check existing ticket: %w", err)
	}

	scanJob, err := s.scanRepo.GetScanJobByID(ctx, scanJobID)
	if err != nil {
		return nil, fmt.Errorf("get scan job: %w", err)
	}

	host, err := s.hostRepo.GetByID(ctx, hostID)
	if err != nil {
		return nil, fmt.Errorf("get host: %w", err)
	}

	// Fetch all incidents for this host in this scan job.
	incidents, err := s.incidentRepo.List(ctx, repository.IncidentFilters{
		HostID:    &hostID,
		ScanJobID: &scanJobID,
	})
	if err != nil {
		return nil, fmt.Errorf("list incidents: %w", err)
	}

	if len(incidents) == 0 {
		return nil, fmt.Errorf("no open incidents to ticket for host %s in scan job %s", hostID, scanJobID)
	}

	req := buildCreateIncidentRequest(scanJob, host, incidents)

	now := time.Now().UTC()
	newTicket := &models.ServiceNowTicket{
		ID:           uuid.New(),
		HostID:       hostID,
		ScanJobID:    scanJobID,
		IncidentIDs:  incidentIDs(incidents),
		TicketOpened: true,
		OpenedAt:     &now,
		CreatedAt:    now,
	}

	ticketNumber, ticketURL, err := s.client.CreateIncident(ctx, req)
	if err != nil {
		errMsg := err.Error()
		newTicket.ErrorMessage = &errMsg
		if createErr := s.ticketRepo.Create(ctx, newTicket); createErr != nil {
			return nil, fmt.Errorf("record failed ticket: %w", createErr)
		}
		return nil, fmt.Errorf("open service now ticket: %w", err)
	}

	newTicket.TicketNumber = &ticketNumber
	newTicket.TicketURL = &ticketURL

	if err := s.ticketRepo.Create(ctx, newTicket); err != nil {
		return nil, fmt.Errorf("create ticket record: %w", err)
	}

	if err := s.markIncidentsTicketed(ctx, incidents, now); err != nil {
		return nil, fmt.Errorf("mark incidents ticketed: %w", err)
	}

	return newTicket, nil
}

// BulkOpenTickets opens ServiceNow incidents for multiple ulas incidents,
// grouping them by (host_id, scan_job_id) so each host gets at most one ticket
// per scan job.
func (s *DefaultServiceNowService) BulkOpenTickets(ctx context.Context, incidentIDs []uuid.UUID) ([]models.ServiceNowTicket, error) {
	type groupKey struct {
		HostID    uuid.UUID
		ScanJobID uuid.UUID
	}
	groups := make(map[groupKey][]uuid.UUID)

	for _, id := range incidentIDs {
		incident, err := s.incidentRepo.GetByID(ctx, id)
		if err != nil {
			continue
		}
		result, err := s.scanRepo.GetScanResult(ctx, incident.ScanResultID)
		if err != nil {
			continue
		}
		key := groupKey{HostID: incident.HostID, ScanJobID: result.ScanJobID}
		groups[key] = append(groups[key], id)
	}

	var tickets []models.ServiceNowTicket
	for key := range groups {
		ticket, err := s.openTicketForHostScan(ctx, key.HostID, key.ScanJobID)
		if err != nil {
			continue
		}
		tickets = append(tickets, *ticket)
	}
	return tickets, nil
}

func buildCreateIncidentRequest(scanJob *models.ScanJob, host *models.Host, incidents []models.Incident) servicenow.CreateIncidentRequest {
	var b strings.Builder
	fmt.Fprintf(&b, "Ulas deviation report for host: %s\n", host.Hostname)
	fmt.Fprintf(&b, "Scan job ID: %s\n", scanJob.ID)
	if scanJob.AnsibleJobID != "" {
		fmt.Fprintf(&b, "Ansible job ID: %s\n", scanJob.AnsibleJobID)
	}
	fmt.Fprintf(&b, "Total deviations: %d\n\n", len(incidents))

	for i, inc := range incidents {
		fmt.Fprintf(&b, "--- Deviation %d ---\n", i+1)
		fmt.Fprintf(&b, "Incident: %s\n", inc.IncidentNumber)
		fmt.Fprintf(&b, "File: %s\n", inc.FileType)
		fmt.Fprintf(&b, "Entry: %s\n", inc.EntryKey)
		if inc.ExpectedValue != nil {
			fmt.Fprintf(&b, "Expected: %s\n", *inc.ExpectedValue)
		} else {
			fmt.Fprintln(&b, "Expected: (not present in baseline)")
		}
		fmt.Fprintf(&b, "Actual: %s\n", inc.ActualValue)
		fmt.Fprintf(&b, "Severity: %s\n\n", inc.Severity)
	}

	return servicenow.CreateIncidentRequest{
		ShortDescription: fmt.Sprintf("Ulas deviation report: %s", host.Hostname),
		Description:      b.String(),
		Urgency:          2,
		Impact:           2,
	}
}

func incidentIDs(incidents []models.Incident) []uuid.UUID {
	ids := make([]uuid.UUID, len(incidents))
	for i, inc := range incidents {
		ids[i] = inc.ID
	}
	return ids
}

func (s *DefaultServiceNowService) markIncidentsTicketed(ctx context.Context, incidents []models.Incident, now time.Time) error {
	for i := range incidents {
		inc := &incidents[i]
		inc.ServiceNowTicketOpened = true
		inc.UpdatedAt = now
		if err := s.incidentRepo.Update(ctx, inc); err != nil {
			return fmt.Errorf("update incident %s: %w", inc.ID, err)
		}
	}
	return nil
}
