package models

import (
	"time"

	"github.com/google/uuid"
)

type HostFileChangeType string

const (
	HostFileChangeTypeAdded    HostFileChangeType = "added"
	HostFileChangeTypeRemoved  HostFileChangeType = "removed"
	HostFileChangeTypeModified HostFileChangeType = "modified"
)

var HostFileChangeTypeValues = []string{
	string(HostFileChangeTypeAdded),
	string(HostFileChangeTypeRemoved),
	string(HostFileChangeTypeModified),
}

// HostFileSnapshot stores the ACTUAL raw content of /etc/passwd or /etc/group
// captured from a host during a scan. One row per file per scan result.
// The full raw file is stored in RawContent so the original formatting is preserved
// and the data can be re-parsed at any time without information loss.
// This table is immutable — rows are never updated after insert.
type HostFileSnapshot struct {
	ID           uuid.UUID `db:"id" json:"id"`
	ScanResultID uuid.UUID `db:"scan_result_id" json:"scan_result_id"`
	HostID       uuid.UUID `db:"host_id" json:"host_id"`
	ScanJobID    uuid.UUID `db:"scan_job_id" json:"scan_job_id"`
	FileType     FileType  `db:"file_type" json:"file_type"`
	// RawContent holds the complete file content as received from the host.
	// For passwd: full /etc/passwd text. For group: full /etc/group text.
	RawContent string    `db:"raw_content" json:"raw_content"`
	LineCount  int       `db:"line_count" json:"line_count"` // number of non-comment, non-empty lines
	SnapshotAt time.Time `db:"snapshot_at" json:"snapshot_at"`
}

// HostFileChange records that a file changed between two consecutive scans for the same host.
// The full before/after content is stored so a diff can be computed or displayed at any time.
// change_type: 'added'    = file did not exist in previous scan, exists now
//
//	'removed'  = file existed in previous scan, missing now
//	'modified' = file exists in both scans but content differs
type HostFileChange struct {
	ID         uuid.UUID          `db:"id" json:"id"`
	HostID     uuid.UUID          `db:"host_id" json:"host_id"`
	FileType   FileType           `db:"file_type" json:"file_type"`
	ChangeType HostFileChangeType `db:"change_type" json:"change_type"`
	// PreviousContent is the full file content from the prior scan (nil if change_type=added).
	PreviousContent *string `db:"previous_content" json:"previous_content,omitempty"`
	// CurrentContent is the full file content from the current scan (nil if change_type=removed).
	CurrentContent    *string    `db:"current_content" json:"current_content,omitempty"`
	PreviousScanJobID *uuid.UUID `db:"previous_scan_job_id" json:"previous_scan_job_id,omitempty"`
	CurrentScanJobID  uuid.UUID  `db:"current_scan_job_id" json:"current_scan_job_id"`
	DetectedAt        time.Time  `db:"detected_at" json:"detected_at"`
}

// HostFileHistoryQuery represents a request to retrieve snapshot history for a host file.
type HostFileHistoryQuery struct {
	HostID    uuid.UUID  `json:"host_id" validate:"required"`
	FileType  FileType   `json:"file_type" validate:"required"`
	AsOfTime  *time.Time `json:"as_of_time,omitempty"`  // If nil, returns the latest snapshot
	ScanJobID *uuid.UUID `json:"scan_job_id,omitempty"` // If set, returns the snapshot for that exact scan
}

// HostFileHistoryResponse represents a single snapshot in historical query results.
type HostFileHistoryResponse struct {
	ScanJobID  uuid.UUID `json:"scan_job_id"`
	SnapshotAt time.Time `json:"snapshot_at"`
	FileType   FileType  `json:"file_type"`
	RawContent string    `json:"raw_content"`
	LineCount  int       `json:"line_count"`
	Changed    bool      `json:"changed,omitempty"` // true if content differs from previous scan
}
