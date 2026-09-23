package config

import (
	"sync"
	"ulas-service/models"
)

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

func (c *AppConfig) SolarisHCVAppRole() models.HCVAppRole {
	return models.HCVAppRole{
		RoleID:    c.SolarisRoleID,
		SecretID:  c.SolarisSecretID,
		HCVPath:   c.SolarisHCVPath,
		Namespace: c.SolarisVaultNameSpace,
	}
}

func vaultAddrForStage(stage Stage) string {
	switch stage {
	case StageProd, StageTud:
		return "test.com"
	case StageEntw, StageTuc:
		return "testprod.com"
	default:
		return "testprod.com"
	}
}
