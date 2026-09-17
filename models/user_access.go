package models

import "time"

// Access levels stored in the user_access table.
const (
	// AccessLevelRead allows viewing baselines, exceptions, scan history,
	// reports, and schedules.
	AccessLevelRead = "read"
	// AccessLevelAdmin allows everything, including managing baselines,
	// exceptions, hosts, schedules, and running scans.
	AccessLevelAdmin = "admin"
)

// UserAccess grants a username (comsiid) access to the portal.
// Users absent from this table have no access at all; rows are managed
// manually by inserting the username with an access level of read or admin.
type UserAccess struct {
	Username    string    `gorm:"primaryKey" db:"username" json:"username"`
	AccessLevel string    `db:"access_level" json:"access_level"`
	CreatedBy   string    `db:"created_by" json:"created_by,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// TableName pins the table name so GORM does not pluralize it.
func (UserAccess) TableName() string {
	return "user_access"
}
