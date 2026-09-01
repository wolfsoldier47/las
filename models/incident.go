package models

import (
	"time"

	"github.com/google/uuid"
)

type IncidentSeverity string

const (
	IncidentSeverityCritical IncidentSeverity = "critical"
	IncidentSeverityHigh     IncidentSeverity = "high"
	IncidentSeverityMedium   IncidentSeverity = "medium"
	IncidentSeverityLow      IncidentSeverity = "low"
)

// IncidentSeverityValues is the canonical list for DB CHECK constraints.
var IncidentSeverityValues = []string{
	string(IncidentSeverityCritical),
	string(IncidentSeverityHigh),
	string(IncidentSeverityMedium),
	string(IncidentSeverityLow),
}

type IncidentStatus string

const (
	IncidentStatusOpen         IncidentStatus = "open"
	IncidentStatusAcknowledged IncidentStatus = "acknowledged"
	IncidentStatusInProgress   IncidentStatus = "in_progress"
	IncidentStatusResolved     IncidentStatus = "resolved"
	IncidentStatusClosed       IncidentStatus = "closed"
)

// IncidentStatusValues is the canonical list for DB CHECK constraints.
var IncidentStatusValues = []string{
	string(IncidentStatusOpen),
	string(IncidentStatusAcknowledged),
	string(IncidentStatusInProgress),
	string(IncidentStatusResolved),
	string(IncidentStatusClosed),
}

// Incident represents an unauthorized deviation detected during scan.
// expected_value comes from master_baselines at the time of scan.
// actual_value comes from host_file_snapshots (the real state found on the host).
// baseline_version_at_scan records which version of the baseline was current,
// so even if the admin later changes the baseline, the incident preserves what was expected at scan time.
type Incident struct {
	ID                     uuid.UUID        `db:"id" json:"id"`
	IncidentNumber         string           `db:"incident_number" json:"incident_number" validate:"required"`
	ScanResultID           uuid.UUID        `db:"scan_result_id" json:"scan_result_id"`
	HostID                 uuid.UUID        `db:"host_id" json:"host_id"`
	FileType               FileType         `db:"file_type" json:"file_type" validate:"required"`
	EntryKey               string           `db:"entry_key" json:"entry_key" validate:"required"`
	ExpectedValue          *string          `db:"expected_value" json:"expected_value,omitempty"`
	ActualValue            string           `db:"actual_value" json:"actual_value" validate:"required"`
	BaselineVersionAtScan  *int             `db:"baseline_version_at_scan" json:"baseline_version_at_scan,omitempty"`
	Severity               IncidentSeverity `db:"severity" json:"severity" validate:"required"`
	Status                 IncidentStatus   `db:"status" json:"status"`
	Notes                  *string          `db:"notes" json:"notes,omitempty"`
	ServiceNowTicketOpened bool             `db:"service_now_ticket_opened" json:"service_now_ticket_opened"`
	Resolution             *string          `db:"resolution" json:"resolution,omitempty"`
	CreatedAt              time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time        `db:"updated_at" json:"updated_at"`
}

// ServiceNowTicket represents a ServiceNow incident ticket linked to one or more
// ulas Incidents for a single host within a single scan job.
// When the user opens tickets, incidents are grouped by (host_id, scan_job_id)
// and one ServiceNow ticket is created per group.
type ServiceNowTicket struct {
	ID           uuid.UUID   `db:"id" json:"id"`
	HostID       uuid.UUID   `db:"host_id" json:"host_id"`
	ScanJobID    uuid.UUID   `db:"scan_job_id" json:"scan_job_id"`
	IncidentIDs  []uuid.UUID `gorm:"-" db:"-" json:"incident_ids,omitempty"`
	TicketNumber *string     `db:"ticket_number" json:"ticket_number,omitempty"`
	TicketURL    *string     `db:"ticket_url" json:"ticket_url,omitempty"`
	// TicketOpened is set to true once the ServiceNow call has been attempted.
	TicketOpened bool       `db:"ticket_opened" json:"ticket_opened"`
	OpenedAt     *time.Time `db:"opened_at" json:"opened_at,omitempty"`
	ErrorMessage *string    `db:"error_message" json:"error_message,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
}

// DeviationResult represents a detected deviation during comparison
type DeviationResult struct {
	HostID          uuid.UUID
	FileType        FileType
	EntryKey        string
	ExpectedValue   string
	ActualValue     string
	IsAllowed       bool
	AllowReason     string
	BaselineVersion int
}
