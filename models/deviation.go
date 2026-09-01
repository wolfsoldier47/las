package models

import (
	"time"

	"github.com/google/uuid"
)

// AllowedDeviation represents a pre-approved exception for a specific host.
// The hostname is stored as free text so that exceptions can be registered
// before the host is inventoried. Comparison logic matches exceptions by
// hostname against the scanned host's hostname.
type AllowedDeviation struct {
	ID            uuid.UUID  `db:"id" json:"id"`
	Hostname      string     `db:"hostname" json:"hostname" validate:"required"`
	FileType      FileType   `db:"file_type" json:"file_type" validate:"required"`
	EntryKey      string     `db:"entry_key" json:"entry_key" validate:"required"`
	EntryValue    *string    `db:"entry_value" json:"entry_value,omitempty"`
	Justification string     `db:"justification" json:"justification" validate:"required"`
	ApprovedBy    string     `db:"approved_by" json:"approved_by" validate:"required"`
	ApprovedAt    time.Time  `db:"approved_at" json:"approved_at"`
	ExpiresAt     *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	IsActive      bool       `db:"is_active" json:"is_active"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
}
