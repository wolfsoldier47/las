package models

import (
	"encoding/json"
	"strings"
)

type OSType string

const (
	OSTypeLinux   OSType = "linux"
	OSTypeSolaris OSType = "solaris"
	OSTypeAIX     OSType = "aix"
)

// OSTypeValues is the canonical list for DB CHECK constraints.
var OSTypeValues = []string{
	string(OSTypeLinux),
	string(OSTypeSolaris),
	string(OSTypeAIX),
}

// UnmarshalJSON accepts OS type values case-insensitively and normalizes to lowercase.
// This keeps the DB canonical while forgiving inputs such as "AIX" from Ansible callbacks.
func (o *OSType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch strings.ToLower(s) {
	case "linux", "rhel", "ubuntu", "centos", "debian", "fedora", "rocky", "almalinux", "sles", "opensuse":
		*o = OSTypeLinux
	case "solaris":
		*o = OSTypeSolaris
	case "aix":
		*o = OSTypeAIX
	default:
		// Preserve unknown values but always store them in lowercase so the DB
		// CHECK constraint can match them case-insensitively.
		*o = OSType(strings.ToLower(s))
	}
	return nil
}
