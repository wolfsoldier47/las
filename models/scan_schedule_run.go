package models

import (
	"time"

	"github.com/google/uuid"
)

// ScanScheduleRunStatus represents the outcome of a scheduled scan run.
type ScanScheduleRunStatus string

const (
	ScanScheduleRunStatusSuccess ScanScheduleRunStatus = "success"
	ScanScheduleRunStatusFailed  ScanScheduleRunStatus = "failed"
)

// ScanScheduleRun records a single execution of a scan schedule.
type ScanScheduleRun struct {
	ID            uuid.UUID             `json:"id" db:"id" gorm:"type:uuid;primary_key"`
	ScheduleID    uuid.UUID             `json:"schedule_id" db:"schedule_id"`
	ScanJobID     *uuid.UUID            `json:"scan_job_id,omitempty" db:"scan_job_id"`
	Status        ScanScheduleRunStatus `json:"status" db:"status"`
	ErrorMessage  string                `json:"error_message,omitempty" db:"error_message"`
	StartedAt     time.Time             `json:"started_at" db:"started_at"`
	CreatedAt     time.Time             `json:"created_at" db:"created_at"`
}
