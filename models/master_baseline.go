package models

import (
	"time"

	"github.com/google/uuid"
)

// MasterBaseline represents the admin-defined "should be" state for /etc/passwd or /etc/group entries.
// This is the REGISTERED master file that scan results are compared against.
// Baselines are global per OS type and major version.
// Entries are grouped into versions; only one version per (os_type, file_type) can be active at a time.
type MasterBaseline struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	OSType      OSType     `db:"os_type" json:"os_type" validate:"required"`
	FileType    FileType   `db:"file_type" json:"file_type" validate:"required"`
	EntryKey    string     `db:"entry_key" json:"entry_key" validate:"required"`
	EntryValue  string     `db:"entry_value" json:"entry_value" validate:"required"`
	Version     int        `db:"version" json:"version"`
	IsActive    bool       `db:"is_active" json:"is_active"`
	Description string     `db:"description" json:"description,omitempty"`
	CreatedBy   string     `db:"created_by" json:"created_by,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

// BaselineVersionSnapshot captures an active master file version at a point in time.
// It is stored with a scan job so the portal can show which baselines were used
// for the initial deviation check.
type BaselineVersionSnapshot struct {
	OSType      OSType    `json:"os_type"`
	FileType    FileType  `json:"file_type"`
	Version     int       `json:"version"`
	EntryCount  int       `json:"entry_count"`
	Description string    `json:"description,omitempty"`
	CreatedBy   string    `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// MasterBaselineVersion tracks every change to a master baseline.
// This preserves the history of what the admin considered "correct" over time.
// Version is set automatically by a DB trigger: it increments per baseline_id,
// starting at 1. Do NOT supply it on insert — the DB fills it in.
type MasterBaselineVersion struct {
	ID           uuid.UUID `db:"id" json:"id"`
	BaselineID   uuid.UUID `db:"baseline_id" json:"baseline_id"`
	Version      int       `db:"version" json:"version"` // auto-set by DB trigger, read-only from Go
	EntryValue   string    `db:"entry_value" json:"entry_value"`
	ChangeReason string    `db:"change_reason" json:"change_reason,omitempty"`
	ChangedBy    string    `db:"changed_by" json:"changed_by,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}
