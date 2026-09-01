package models

import (
	"time"

	"github.com/google/uuid"
)

// Host represents a managed Linux, Solaris, or AIX machine
type Host struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Hostname    string    `db:"hostname" json:"hostname" validate:"required"`
	OSType      OSType    `db:"os_type" json:"os_type" validate:"required"`
	OSName      string    `db:"os_name" json:"os_name,omitempty"`
	OSVersion   string    `db:"os_version" json:"os_version,omitempty"`
	Environment string    `db:"environment" json:"environment,omitempty"`
	Datacenter  string    `db:"datacenter" json:"datacenter,omitempty"`
	Description string    `db:"description" json:"description,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}
