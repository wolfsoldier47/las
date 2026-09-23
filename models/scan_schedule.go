package models

import (
	"time"

	"github.com/google/uuid"
)

// ScanScheduleFrequency represents the cadence of a scheduled scan.
type ScanScheduleFrequency string

const (
	ScanScheduleFrequencyDaily   ScanScheduleFrequency = "daily"
	ScanScheduleFrequencyWeekly  ScanScheduleFrequency = "weekly"
	ScanScheduleFrequencyMonthly ScanScheduleFrequency = "monthly"
)

// ScanScheduleFrequencyValues is the list of supported schedule frequencies.
var ScanScheduleFrequencyValues = []ScanScheduleFrequency{
	ScanScheduleFrequencyDaily,
	ScanScheduleFrequencyWeekly,
	ScanScheduleFrequencyMonthly,
}

// IsValidScanScheduleFrequency reports whether a frequency value is supported.
func IsValidScanScheduleFrequency(f ScanScheduleFrequency) bool {
	switch f {
	case ScanScheduleFrequencyDaily, ScanScheduleFrequencyWeekly, ScanScheduleFrequencyMonthly:
		return true
	}
	return false
}

// ScanSchedule stores a recurring scan definition.
type ScanSchedule struct {
	ID         uuid.UUID             `json:"id" db:"id" gorm:"type:uuid;primary_key"`
	Name       string                `json:"name" db:"name"`
	Frequency  ScanScheduleFrequency `json:"frequency" db:"frequency"`
	Limit      string                `json:"limit" db:"limit"`
	TargetOS   OSType                `json:"target_os" db:"target_os" gorm:"column:target_os;type:text;default:linux"`
	Enabled    bool                  `json:"enabled" db:"enabled"`
	NextRunAt  *time.Time            `json:"next_run_at" db:"next_run_at"`
	LastRunAt  *time.Time            `json:"last_run_at" db:"last_run_at"`
	CreatedBy  string                `json:"created_by" db:"created_by"`
	CreatedAt  time.Time             `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time             `json:"updated_at" db:"updated_at"`
}

// IsValidScanScheduleTargetOS reports whether an OS type can be targeted by
// a scheduled scan. Only OS types with a configured AAP instance qualify.
func IsValidScanScheduleTargetOS(os OSType) bool {
	switch os {
	case OSTypeLinux, OSTypeSolaris:
		return true
	}
	return false
}

// ComputeNextRun returns the next scheduled run time based on the frequency.
func (s *ScanSchedule) ComputeNextRun(from time.Time) time.Time {
	switch s.Frequency {
	case ScanScheduleFrequencyDaily:
		return from.AddDate(0, 0, 1)
	case ScanScheduleFrequencyWeekly:
		return from.AddDate(0, 0, 7)
	case ScanScheduleFrequencyMonthly:
		return from.AddDate(0, 1, 0)
	default:
		return from.AddDate(0, 0, 1)
	}
}
