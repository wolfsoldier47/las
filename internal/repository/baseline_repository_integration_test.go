//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ulas-service/internal/config"
	"ulas-service/models"
)

// TestBaselineCheckIDsRoundTrip_DB verifies the check_ids (privilege list)
// flag survives the repository insert/read paths against a live database.
func TestBaselineCheckIDsRoundTrip_DB(t *testing.T) {
	ctx := context.Background()

	gormDB, err := gorm.Open(postgres.Open(config.Get().DatabaseDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	defer sqlDB.Close()

	repo := NewPgBaselineRepository(sqlDB)
	now := time.Now().UTC()

	privileged := &models.MasterBaseline{
		ID:         uuid.New(),
		OSType:     models.OSTypeLinux,
		FileType:   models.FileTypePasswd,
		EntryKey:   "e2e-priv-" + uuid.New().String()[:8],
		EntryValue: "x:0:0:root:/root:/bin/bash",
		Version:    7,
		IsActive:   true,
		CheckIDs:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	plain := &models.MasterBaseline{
		ID:         uuid.New(),
		OSType:     models.OSTypeLinux,
		FileType:   models.FileTypePasswd,
		EntryKey:   "e2e-plain-" + uuid.New().String()[:8],
		EntryValue: "x:1:1:daemon:/sbin:/sbin/nologin",
		Version:    7,
		IsActive:   true,
		CheckIDs:   false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	defer func() {
		sqlDB.ExecContext(ctx, `DELETE FROM master_baselines WHERE id = $1`, privileged.ID)
		sqlDB.ExecContext(ctx, `DELETE FROM master_baselines WHERE id = $1`, plain.ID)
	}()

	for _, b := range []*models.MasterBaseline{privileged, plain} {
		if err := repo.Create(ctx, b); err != nil {
			t.Fatalf("create baseline: %v", err)
		}
	}

	got, err := repo.GetByID(ctx, privileged.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if !got.CheckIDs {
		t.Errorf("expected CheckIDs true after round trip")
	}

	gotPlain, err := repo.GetByID(ctx, plain.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if gotPlain.CheckIDs {
		t.Errorf("expected CheckIDs false after round trip")
	}

	active := true
	linux := models.OSTypeLinux
	passwd := models.FileTypePasswd
	listed, err := repo.List(ctx, BaselineFilters{OSType: &linux, FileType: &passwd, IsActive: &active})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := map[string]bool{}
	for _, b := range listed {
		if b.ID == privileged.ID && b.CheckIDs {
			found["privileged"] = true
		}
		if b.ID == plain.ID && !b.CheckIDs {
			found["plain"] = true
		}
	}
	if !found["privileged"] || !found["plain"] {
		t.Errorf("list did not return expected check_ids values: %v", found)
	}
}
