package config

import "sync"

type Stage string

const (
	StageProd Stage = "prd"
	StageEntw Stage = "entw"
	StageTuc  Stage = "tuc"
	StageTud  Stage = "tud"
)

var (
	instance *AppConfig
	once     sync.Once
)

// Reload re-reads the configuration from the environment, replacing the
// current singleton instance. Used at startup after bootstrap steps (e.g.
// the Solaris vault credential fetch) set new environment variables.
func Reload() {
	instance = nil
	once = sync.Once{}
	Load()
}
