//go:build integration

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ulas-service/internal/config"
	"ulas-service/internal/repository"
	"ulas-service/models"
)

// generateHostsWithPrefix creates n callback payloads with hostnames like "<prefix>-00000.example.com".
func generateHostsWithPrefix(n int, prefix string) []models.CallbackPayload {
	passwd := []map[string]string{
		{"root": "x:0:0:root:/root:/bin/bash"},
		{"bin": "x:1:1:bin:/bin:/sbin/nologin"},
		{"daemon": "x:2:2:daemon:/sbin:/sbin/nologin"},
	}
	group := []map[string]string{
		{"root": "x:0:"},
		{"bin": "x:1:"},
		{"daemon": "x:2:"},
	}

	hosts := make([]models.CallbackPayload, n)
	for i := 0; i < n; i++ {
		hosts[i] = models.CallbackPayload{
			MachineName: fmt.Sprintf("%s-%05d.example.com", prefix, i),
			MachineType: models.OSTypeLinux,
			OSVersion:   "8.10",
			OSName:      "RedHat",
			Environment: "prod",
			Datacenter:  "dc1",
			PasswdFile:  passwd,
			GroupFile:   group,
		}
	}
	return hosts
}

func TestProcessCallbackEnvelope_20kHosts_DB(t *testing.T) {
	const hostCount = 20000

	ctx := context.Background()
	cfg := config.Get()

	gormDB, err := gorm.Open(postgres.Open(cfg.DatabaseDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(80)
	sqlDB.SetMaxIdleConns(20)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	defer sqlDB.Close()

	scanRepo := repository.NewPgScanRepository(sqlDB)
	hostRepo := repository.NewPgHostRepository(sqlDB)
	snapshotRepo := repository.NewPgSnapshotRepository(sqlDB)
	incidentRepo := repository.NewPgIncidentRepository(sqlDB)
	baselineRepo := repository.NewPgBaselineRepository(sqlDB)

	// Use a no-op comparison service so the benchmark measures callback ingestion
	// (host + scan result + snapshot writes) without the extra comparison goroutines
	// hitting the database.
	comparisonSvc := &noopComparisonService{}
	svc := NewDefaultScanService(
		scanRepo,
		hostRepo,
		snapshotRepo,
		incidentRepo,
		baselineRepo,
		nil,
		comparisonSvc,
		"test-template",
		"http://localhost:8080",
		"test",
		nil,
	)

	prefix := fmt.Sprintf("bench-%s", uuid.New().String())
	ansibleJobID := fmt.Sprintf("job-%s", prefix)
	now := time.Now().UTC()
	job := &models.ScanJob{
		ID:            uuid.New(),
		AnsibleJobID:  ansibleJobID,
		JobTemplateID: 1,
		Status:        models.ScanJobStatusRunning,
		InitiatedBy:   "benchmark",
		StartedAt:     &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := scanRepo.CreateScanJob(ctx, job); err != nil {
		t.Fatalf("create scan job: %v", err)
	}

	hosts := generateHostsWithPrefix(hostCount, prefix)

	start := time.Now()
	if err := svc.ProcessCallbackEnvelope(ctx, ansibleJobID, hosts, nil); err != nil {
		t.Fatalf("process callback envelope: %v", err)
	}
	// ProcessCallbackEnvelope returns immediately; wait until the worker pool has
	// persisted all scan results and snapshots before measuring elapsed time.
	waitForIngestion(t, gormDB, job.ID, hostCount, 600*time.Second)
	ingestionElapsed := time.Since(start)

	t.Logf("DB ingestion (%d hosts): %v (%.2f hosts/sec)", hostCount, ingestionElapsed, float64(hostCount)/ingestionElapsed.Seconds())

	var resultCount int64
	if err := gormDB.Table("scan_results").Where("scan_job_id = ?", job.ID).Count(&resultCount).Error; err != nil {
		t.Fatalf("count scan results: %v", err)
	}
	if resultCount != hostCount {
		t.Errorf("expected %d scan results, got %d", hostCount, resultCount)
	}

	var snapshotCount int64
	if err := gormDB.Table("host_file_snapshots").Where("scan_job_id = ?", job.ID).Count(&snapshotCount).Error; err != nil {
		t.Fatalf("count snapshots: %v", err)
	}
	if snapshotCount != hostCount*2 {
		t.Errorf("expected %d snapshots, got %d", hostCount*2, snapshotCount)
	}

	// Clean up now that all ingestion is complete.
	cleanupBenchmarkData(t, gormDB, job.ID, prefix)
}

func waitForIngestion(t *testing.T, db *gorm.DB, jobID uuid.UUID, expected int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var resultCount, snapshotCount int64
		if err := db.Table("scan_results").Where("scan_job_id = ?", jobID).Count(&resultCount).Error; err != nil {
			t.Fatalf("count scan results: %v", err)
		}
		if err := db.Table("host_file_snapshots").Where("scan_job_id = ?", jobID).Count(&snapshotCount).Error; err != nil {
			t.Fatalf("count snapshots: %v", err)
		}
		if resultCount == int64(expected) && snapshotCount == int64(expected*2) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for ingestion (job_id=%s)", jobID)
}

func cleanupBenchmarkData(t *testing.T, db *gorm.DB, jobID uuid.UUID, hostnamePrefix string) {
	pattern := hostnamePrefix + "-%"

	if err := db.Exec("DELETE FROM incidents WHERE scan_result_id IN (SELECT id FROM scan_results WHERE scan_job_id = ?)", jobID).Error; err != nil {
		t.Logf("cleanup incidents: %v", err)
	}
	if err := db.Exec("DELETE FROM host_file_snapshots WHERE scan_job_id = ?", jobID).Error; err != nil {
		t.Logf("cleanup snapshots: %v", err)
	}
	if err := db.Exec("DELETE FROM scan_results WHERE scan_job_id = ?", jobID).Error; err != nil {
		t.Logf("cleanup scan_results: %v", err)
	}
	if err := db.Exec("DELETE FROM hosts WHERE hostname LIKE ?", pattern).Error; err != nil {
		t.Logf("cleanup hosts: %v", err)
	}
	if err := db.Exec("DELETE FROM scan_jobs WHERE id = ?", jobID).Error; err != nil {
		t.Logf("cleanup scan_jobs: %v", err)
	}
}
