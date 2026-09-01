package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"ulas-service/internal/aap"
	"ulas-service/internal/config"
	"ulas-service/internal/database"
	"ulas-service/internal/repository"
	"ulas-service/internal/service"
	"ulas-service/models"
)

func main() {
	logHandler := config.LoggerInit()
	defer logHandler.Close()

	config.Load()
	cfg := config.Get()

	if err := database.Initialize(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	db := database.SQLDB()
	defer db.Close()

	ctx := context.Background()

	hostRepo := repository.NewPgHostRepository(db)
	baselineRepo := repository.NewPgBaselineRepository(db)
	scanRepo := repository.NewPgScanRepository(db)
	snapshotRepo := repository.NewPgSnapshotRepository(db)
	incidentRepo := repository.NewPgIncidentRepository(db)
	deviationRepo := repository.NewPgDeviationRepository(db)

	if err := seedBaselines(ctx, baselineRepo); err != nil {
		fmt.Fprintf(os.Stderr, "failed to seed baselines: %v\n", err)
		os.Exit(1)
	}

	comparisonService := service.NewDefaultComparisonService(
		scanRepo,
		snapshotRepo,
		hostRepo,
		baselineRepo,
		deviationRepo,
		incidentRepo,
	)

	aapClient := aap.NewClient(cfg.AAPURL+cfg.AAPRESTVERSION, cfg.AAPUsername, cfg.AAPPassword)
	scanService := service.NewDefaultScanService(
		scanRepo,
		hostRepo,
		snapshotRepo,
		incidentRepo,
		baselineRepo,
		aapClient,
		comparisonService,
		cfg.AAPJobTemplateName,
		cfg.BackEndBaseUrl,
		cfg.AppStage,
		cfg,
	)

	const hostCount = 100
	const ansibleJobID = "seed-100-noncompliant"

	jobID := uuid.New()
	now := time.Now().UTC()
	job := &models.ScanJob{
		ID:           jobID,
		AnsibleJobID: ansibleJobID,
		Status:       models.ScanJobStatusRunning,
		InitiatedBy:  "seed",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := scanRepo.CreateScanJob(ctx, job); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create scan job: %v\n", err)
		os.Exit(1)
	}

	hosts := generateHosts(hostCount)
	if err := scanService.ProcessCallbackEnvelope(ctx, ansibleJobID, hosts, nil); err != nil {
		fmt.Fprintf(os.Stderr, "failed to process callback envelope: %v\n", err)
		os.Exit(1)
	}

	// Wait for the worker pool to ingest all callbacks.
	if err := waitForResults(ctx, scanRepo, jobID, hostCount); err != nil {
		fmt.Fprintf(os.Stderr, "failed waiting for scan results: %v\n", err)
		os.Exit(1)
	}

	// Wait for asynchronous comparison to finish creating incidents.
	if err := waitForIncidents(ctx, incidentRepo, jobID, hostCount); err != nil {
		fmt.Fprintf(os.Stderr, "failed waiting for incidents: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Seeded %d non-compliant hosts into scan job %s\n", hostCount, jobID)
}

func seedBaselines(ctx context.Context, baselineRepo repository.BaselineRepository) error {
	entries := []models.MasterBaseline{
		// /etc/passwd baseline for RHEL 8.
		{ID: uuid.New(), OSType: models.OSTypeLinux, FileType: models.FileTypePasswd, EntryKey: "root", EntryValue: "x:0:0:root:/root:/bin/bash", Version: 8, IsActive: true, CreatedBy: "seed", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		{ID: uuid.New(), OSType: models.OSTypeLinux, FileType: models.FileTypePasswd, EntryKey: "bin", EntryValue: "x:1:1:bin:/bin:/sbin/nologin", Version: 8, IsActive: true, CreatedBy: "seed", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		// /etc/group baseline for RHEL 8. Values are stored without trailing colons
		// because the comparison parser normalizes group snapshots that way.
		{ID: uuid.New(), OSType: models.OSTypeLinux, FileType: models.FileTypeGroup, EntryKey: "root", EntryValue: "x:0", Version: 8, IsActive: true, CreatedBy: "seed", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		{ID: uuid.New(), OSType: models.OSTypeLinux, FileType: models.FileTypeGroup, EntryKey: "bin", EntryValue: "x:1", Version: 8, IsActive: true, CreatedBy: "seed", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		{ID: uuid.New(), OSType: models.OSTypeLinux, FileType: models.FileTypeGroup, EntryKey: "daemon", EntryValue: "x:2", Version: 8, IsActive: true, CreatedBy: "seed", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	}

	for i := range entries {
		if err := baselineRepo.Create(ctx, &entries[i]); err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				continue
			}
			return err
		}
	}
	return nil
}

func generateHosts(n int) []models.CallbackPayload {
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
			MachineName: fmt.Sprintf("host-%05d.example.com", i),
			MachineType: models.OSTypeLinux,
			OSVersion:   "8.10",
			OSName:      "RedHat",
			Environment: "prod",
			Datacenter:  "ffm",
			PasswdFile:  passwd,
			GroupFile:   group,
		}
	}
	return hosts
}

func waitForResults(ctx context.Context, scanRepo repository.ScanRepository, jobID uuid.UUID, expected int) error {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		results, err := scanRepo.ListScanResultsByJobID(ctx, jobID)
		if err != nil {
			return err
		}
		if len(results) >= expected {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %d scan results", expected)
}

func waitForIncidents(ctx context.Context, incidentRepo repository.IncidentRepository, jobID uuid.UUID, expected int) error {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		incidents, err := incidentRepo.List(ctx, repository.IncidentFilters{ScanJobID: &jobID})
		if err != nil {
			return err
		}
		if len(incidents) >= expected {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %d incidents", expected)
}
