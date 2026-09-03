package models

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ScanJobStatus string

const (
	ScanJobStatusInitiating ScanJobStatus = "initiating"
	ScanJobStatusRunning    ScanJobStatus = "running"
	ScanJobStatusCompleted  ScanJobStatus = "completed"
	ScanJobStatusFailed     ScanJobStatus = "failed"
	ScanJobStatusTimeout    ScanJobStatus = "timeout"
	ScanJobStatusCancelled  ScanJobStatus = "cancelled"
)

// AAPJobStatus represents the status strings returned by the Ansible Automation Platform API.
type AAPJobStatus string

const (
	AAPJobStatusSuccessful AAPJobStatus = "successful"
	AAPJobStatusFailed     AAPJobStatus = "failed"
	AAPJobStatusError      AAPJobStatus = "error"
	AAPJobStatusCanceled   AAPJobStatus = "canceled"
	AAPJobStatusPending    AAPJobStatus = "pending"
	AAPJobStatusWaiting    AAPJobStatus = "waiting"
	AAPJobStatusRunning    AAPJobStatus = "running"
)

var ScanJobStatusValues = []string{
	string(ScanJobStatusInitiating),
	string(ScanJobStatusRunning),
	string(ScanJobStatusCompleted),
	string(ScanJobStatusFailed),
	string(ScanJobStatusTimeout),
	string(ScanJobStatusCancelled),
}

var ActiveScanJobStatusValues = []string{
	string(ScanJobStatusInitiating),
	string(ScanJobStatusRunning),
}

type ScanResultStatus string

const (
	ScanResultStatusPending          ScanResultStatus = "pending"
	ScanResultStatusSuccess          ScanResultStatus = "success"
	ScanResultStatusFailed           ScanResultStatus = "failed"
	ScanResultStatusDeviationFound   ScanResultStatus = "deviation_found"
	ScanResultStatusAllowedDeviation ScanResultStatus = "allowed_deviation"
)

var ScanResultStatusValues = []string{
	string(ScanResultStatusPending),
	string(ScanResultStatusSuccess),
	string(ScanResultStatusFailed),
	string(ScanResultStatusDeviationFound),
	string(ScanResultStatusAllowedDeviation),
}

type ScanProcessingStatus string

const (
	ScanProcessingStatusPending    ScanProcessingStatus = "pending"
	ScanProcessingStatusProcessing ScanProcessingStatus = "processing"
	ScanProcessingStatusProcessed  ScanProcessingStatus = "processed"
	ScanProcessingStatusFailed     ScanProcessingStatus = "failed"
)

var ScanProcessingStatusValues = []string{
	string(ScanProcessingStatusPending),
	string(ScanProcessingStatusProcessing),
	string(ScanProcessingStatusProcessed),
	string(ScanProcessingStatusFailed),
}

// ScanJob represents an initiated Ansible scan job
type ScanJob struct {
	ID            uuid.UUID     `db:"id" json:"id"`
	AnsibleJobID  string        `db:"ansible_job_id" json:"ansible_job_id"`   // job ID returned by Tower on launch
	JobTemplateID int           `db:"job_template_id" json:"job_template_id"` // Tower Job Template ID
	OSType        OSType        `db:"os_type" json:"os_type"`                 // target OS type (linux, solaris, aix)
	Limit         string        `db:"limit" json:"limit,omitempty"`           // host pattern passed to Tower
	Status        ScanJobStatus `db:"status" json:"status"`
	InitiatedBy   string        `db:"initiated_by" json:"initiated_by,omitempty"`
	StartedAt     *time.Time    `db:"started_at" json:"started_at,omitempty"`

	CompletedAt       *time.Time                 `db:"completed_at" json:"completed_at,omitempty"`
	TotalHosts        int                        `db:"total_hosts" json:"total_hosts"`
	CallbacksReceived int                        `db:"callbacks_received" json:"callbacks_received"`
	SuccessfulHosts   int                        `db:"successful_hosts" json:"successful_hosts"`
	FailedHosts       int                        `db:"failed_hosts" json:"failed_hosts"`
	FailedHostNames   []string                   `gorm:"type:jsonb;serializer:json" db:"failed_host_names" json:"failed_host_names,omitempty"`
	ErrorMessage      string                     `db:"error_message" json:"error_message,omitempty"`
	BaselineSnapshot  []BaselineVersionSnapshot  `gorm:"type:jsonb;serializer:json" db:"baseline_snapshot" json:"baseline_snapshot,omitempty"`
	CreatedAt         time.Time                  `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time                  `db:"updated_at" json:"updated_at"`
}

// ScanJobSummary extends ScanJob with aggregated deviation counts.
type ScanJobSummary struct {
	ScanJob
	TotalDeviations        int `json:"total_deviations"`
	TotalAllowedDeviations int `json:"total_allowed_deviations"`
}

// AllowedDeviationFound records a baseline deviation that is covered by an exception.
type AllowedDeviationFound struct {
	FileType      FileType `json:"file_type"`
	EntryKey      string   `json:"entry_key"`
	ExpectedValue string   `json:"expected_value,omitempty"`
	ActualValue   string   `json:"actual_value"`
}

// AllowedDeviations is a JSONB-backed slice for storage in PostgreSQL.
type AllowedDeviations []AllowedDeviationFound

// Value marshals the slice for storage as JSONB.
func (d AllowedDeviations) Value() (driver.Value, error) {
	if len(d) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Scan unmarshals a JSONB column into the slice.
func (d *AllowedDeviations) Scan(value interface{}) error {
	if value == nil {
		*d = nil
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan allowed_deviations: unexpected type %T", value)
	}
	if len(data) == 0 {
		*d = nil
		return nil
	}
	return json.Unmarshal(data, d)
}

// ScanResult represents per-host summary per scan job.
// The actual file contents are stored in host_file_snapshots (one row per entry).
type ScanResult struct {
	ID                    uuid.UUID            `db:"id" json:"id"`
	ScanJobID             uuid.UUID            `db:"scan_job_id" json:"scan_job_id"`
	HostID                uuid.UUID            `db:"host_id" json:"host_id"`
	Status                ScanResultStatus     `db:"status" json:"status"`
	ErrorMessage          string               `db:"error_message" json:"error_message,omitempty"`
	ProcessingStatus      ScanProcessingStatus `db:"processing_status" json:"processing_status"`
	DeviationsFound       int                  `db:"deviations_found" json:"deviations_found"`
	AllowedDeviations     AllowedDeviations    `gorm:"type:jsonb;serializer:json" db:"allowed_deviations" json:"allowed_deviations,omitempty"`
	BaselineVersionAtScan *int                 `db:"baseline_version_at_scan" json:"baseline_version_at_scan,omitempty"`
	NoBaseline            bool                 `db:"no_baseline" json:"no_baseline"`
	ReceivedAt            *time.Time           `db:"received_at" json:"received_at,omitempty"`
	ProcessedAt           *time.Time           `db:"processed_at" json:"processed_at,omitempty"`
	CreatedAt             time.Time            `db:"created_at" json:"created_at"`
}

// CallbackPayload represents the JSON received from Ansible callback API.
// The raw file contents are parsed and stored as individual HostFileSnapshot rows.
type CallbackPayload struct {
	JobID        string                 `json:"job_id,omitempty"`
	MachineName  string                 `json:"machine_name" validate:"required"`
	MachineType  OSType                 `json:"machine_type" validate:"required"`
	OSVersion    string                 `json:"os_version,omitempty"`
	OSName       string                 `json:"os_name,omitempty"`
	Environment  string                 `json:"environment,omitempty"`
	Datacenter   string                 `json:"DataCenter,omitempty"`
	PasswdFile   []map[string]string    `json:"passwd_file"`
	GroupFile    []map[string]string    `json:"group_file"`
	Timestamp    time.Time              `json:"timestamp"`
	AnsibleFacts map[string]interface{} `json:"ansible_facts,omitempty"`
}

// FlexibleStringSlice is a JSON array that accepts both string and numeric
// elements and stores them as strings. AAP may send failed_hosts as host
// IDs (integers) or hostnames (strings), so both shapes are normalized.
type FlexibleStringSlice []string

// UnmarshalJSON accepts a JSON array of strings, numbers, or a mix and stores
// every element as a string.
func (s *FlexibleStringSlice) UnmarshalJSON(data []byte) error {
	var raw []interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*s = make([]string, 0, len(raw))
	for _, v := range raw {
		switch id := v.(type) {
		case string:
			*s = append(*s, id)
		case float64:
			*s = append(*s, fmt.Sprintf("%.0f", id))
		case int:
			*s = append(*s, fmt.Sprintf("%d", id))
		case int64:
			*s = append(*s, fmt.Sprintf("%d", id))
		default:
			*s = append(*s, fmt.Sprintf("%v", id))
		}
	}
	return nil
}

// CallbackEnvelope is the new AAP callback format: a top-level ansible_job_id
// with a nested hosts array. JobID is no longer required inside each host object.
// FailedHosts lists hostnames (or host IDs) that AAP could not reach.
type CallbackEnvelope struct {
	AnsibleJobID interface{}         `json:"ansible_job_id" validate:"required"`
	JobID        interface{}         `json:"job_id,omitempty"` // some AAP versions send job_id instead of ansible_job_id on the failed_hosts summary
	Hosts        []CallbackPayload   `json:"hosts" validate:"required"`
	FailedHosts  FlexibleStringSlice `json:"failed_hosts,omitempty"`
}

// UnmarshalJSON accepts both "hosts" and the singular "host" for the host list
// so payloads from different AAP callback formats are handled.
func (e *CallbackEnvelope) UnmarshalJSON(data []byte) error {
	type alias CallbackEnvelope
	var aux struct {
		*alias
		Host []CallbackPayload `json:"host"`
	}
	aux.alias = (*alias)(e)

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(e.Hosts) == 0 && len(aux.Host) > 0 {
		e.Hosts = aux.Host
	}
	return nil
}

// UnmarshalJSON normalizes the various field names Ansible may send.
//   - machine_type / os_type
//   - environment / stage
//   - DataCenter / datacenter / datacentre
// It also accepts passwd_file and group_file as either JSON arrays or string-encoded JSON arrays.
func (p *CallbackPayload) UnmarshalJSON(data []byte) error {
	type Alias CallbackPayload
	aux := &struct {
		MachineType string              `json:"machine_type"`
		Ostype      string              `json:"os_type"`
		OsVersion   string              `json:"os_verion"`
		Environment string              `json:"environment"`
		Stage       string              `json:"stage"`
		DataCenter  string              `json:"DataCenter"`
		Datacenter  string              `json:"datacenter"`
		Datacentre  string              `json:"datacentre"`
		PasswdFile  json.RawMessage     `json:"passwd_file"`
		GroupFile   json.RawMessage     `json:"group_file"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Normalize machine_type / os_type.
	if aux.MachineType != "" {
		p.MachineType = parseOSType(aux.MachineType)
	}
	if aux.Ostype != "" {
		p.MachineType = parseOSType(aux.Ostype)
	}

	// Normalize environment / stage.
	if aux.Environment != "" {
		p.Environment = aux.Environment
	}
	if aux.Stage != "" {
		p.Environment = aux.Stage
	}

	// Normalize os_version (also accepts the misspelled "os_verion" field).
	if aux.OsVersion != "" {
		p.OSVersion = aux.OsVersion
	}

	// Normalize DataCenter / datacenter / datacentre.
	if aux.DataCenter != "" {
		p.Datacenter = aux.DataCenter
	}
	if aux.Datacenter != "" {
		p.Datacenter = aux.Datacenter
	}
	if aux.Datacentre != "" {
		p.Datacenter = aux.Datacentre
	}

	// Parse passwd_file / group_file. They may be JSON arrays or string-encoded JSON arrays.
	var err error
	if p.PasswdFile, err = normalizeFileEntries(aux.PasswdFile); err != nil {
		return fmt.Errorf("parse passwd_file: %w", err)
	}
	if p.GroupFile, err = normalizeFileEntries(aux.GroupFile); err != nil {
		return fmt.Errorf("parse group_file: %w", err)
	}

	return nil
}

// normalizeFileEntries accepts a JSON array or a string containing a JSON array
// and returns a slice of map[string]string entries.
func normalizeFileEntries(raw json.RawMessage) ([]map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var entries []map[string]string
	if err := json.Unmarshal(raw, &entries); err == nil {
		return entries, nil
	}

	var str string
	if err := json.Unmarshal(raw, &str); err != nil {
		return nil, err
	}

	str = strings.TrimSpace(str)
	if str == "" {
		return nil, nil
	}
	if err := json.Unmarshal([]byte(str), &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// parseOSType reuses the OSType unmarshaler so that values such as "rhel" are normalized to "linux".
func parseOSType(s string) OSType {
	var o OSType
	_ = json.Unmarshal([]byte(strconv.Quote(s)), &o)
	return o
}

// ScanResultResponse represents the aggregated result of a completed scan
type ScanResultResponse struct {
	JobID             string           `json:"job_id"`
	Status            string           `json:"status"`
	TotalHosts        int              `json:"total_hosts"`
	CallbacksReceived int              `json:"callbacks_received"`
	SuccessfulHosts   int              `json:"successful_hosts"`
	FailedHosts       int              `json:"failed_hosts"`
	StartedAt         *time.Time       `json:"started_at,omitempty"`
	CompletedAt       *time.Time       `json:"completed_at,omitempty"`
	HostResults       []HostScanResult `json:"host_results"`
	IncidentsOpened   int              `json:"incidents_opened"`
	ErrorMessage      string           `json:"error_message,omitempty"`
}

// FileEntry represents a single entry from passwd/group file
type FileEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// HostScanResult represents a single host's scan result
type HostScanResult struct {
	Hostname      string            `json:"hostname"`
	MachineType   OSType            `json:"machine_type"`
	Status        string            `json:"status"`
	PasswdEntries []FileEntry       `json:"passwd_entries,omitempty"`
	GroupEntries  []FileEntry       `json:"group_entries,omitempty"`
	Deviations    []DeviationDetail `json:"deviations,omitempty"`
	ErrorMessage  string            `json:"error_message,omitempty"`
}

// DeviationDetail represents a detected deviation for response
type DeviationDetail struct {
	FileType        FileType `json:"file_type"`
	EntryKey        string   `json:"entry_key"`
	ExpectedValue   string   `json:"expected_value,omitempty"`
	ActualValue     string   `json:"actual_value"`
	IsAllowed       bool     `json:"is_allowed"`
	IncidentID      string   `json:"incident_id,omitempty"`
	BaselineVersion int      `json:"baseline_version,omitempty"`
}

type ScanInitiationWithAAPService interface {
	InitiateScan(ctx context.Context, req *AapExtraVars, initiatedBy string, workflowID string) (*ScanResultResponse, error)
}
