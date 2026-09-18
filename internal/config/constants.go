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
