package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"ulas-service/internal/repository"
	"ulas-service/models"
)

// in-memory test doubles

type memScanRepo struct {
	results map[uuid.UUID]*models.ScanResult
}

func newMemScanRepo() *memScanRepo {
	return &memScanRepo{results: make(map[uuid.UUID]*models.ScanResult)}
}

func (r *memScanRepo) CreateScanJob(ctx context.Context, job *models.ScanJob) error { return nil }
func (r *memScanRepo) GetScanJobByID(ctx context.Context, id uuid.UUID) (*models.ScanJob, error) {
	return nil, repository.ErrScanJobNotFound
}
func (r *memScanRepo) GetScanJobByAnsibleJobID(ctx context.Context, ansibleJobID string) (*models.ScanJob, error) {
	return nil, repository.ErrScanJobNotFound
}
func (r *memScanRepo) ListScanJobs(ctx context.Context) ([]models.ScanJob, error)   { return nil, nil }
func (r *memScanRepo) ListScanJobsPaginated(ctx context.Context, page, limit int) ([]models.ScanJob, int, error) {
	return nil, 0, nil
}
func (r *memScanRepo) ListScanJobsPaginatedWithDeviationCounts(ctx context.Context, page, limit int, onlyWithDeviations bool, search string, fromDate, toDate *time.Time) ([]models.ScanJobSummary, int, error) {
	return nil, 0, nil
}
func (r *memScanRepo) HasActiveScanJob(ctx context.Context) (bool, error) { return false, nil }
func (r *memScanRepo) UpdateScanJob(ctx context.Context, job *models.ScanJob) error { return nil }
func (r *memScanRepo) CreateScanResult(ctx context.Context, result *models.ScanResult) error {
	r.results[result.ID] = result
	return nil
}
func (r *memScanRepo) GetScanResult(ctx context.Context, id uuid.UUID) (*models.ScanResult, error) {
	res, ok := r.results[id]
	if !ok {
		return nil, repository.ErrScanJobNotFound
	}
	return res, nil
}
func (r *memScanRepo) GetScanResultByJobAndHost(ctx context.Context, scanJobID, hostID uuid.UUID) (*models.ScanResult, error) {
	return nil, nil
}
func (r *memScanRepo) ListScanResultsByJobID(ctx context.Context, scanJobID uuid.UUID) ([]models.ScanResult, error) {
	return nil, nil
}
func (r *memScanRepo) ListScanResultsByJobIDPaginated(ctx context.Context, scanJobID uuid.UUID, page, limit int) ([]models.ScanResult, int, error) {
	return nil, 0, nil
}
func (r *memScanRepo) UpdateScanResult(ctx context.Context, result *models.ScanResult) error {
	r.results[result.ID] = result
	return nil
}
func (r *memScanRepo) IncrementScanJobCounters(ctx context.Context, id uuid.UUID, callbacks, successful, failed int) error {
	return nil
}
func (r *memScanRepo) UpdateScanJobStatus(ctx context.Context, job *models.ScanJob) error { return nil }
func (r *memScanRepo) UpdateScanJobLaunch(ctx context.Context, job *models.ScanJob) error { return nil }
func (r *memScanRepo) AppendFailedHosts(ctx context.Context, id uuid.UUID, hostnames []string) error {
	return nil
}

type memSnapshotRepo struct {
	mu        sync.RWMutex
	snapshots map[uuid.UUID]*models.HostFileSnapshot
}

func newMemSnapshotRepo() *memSnapshotRepo {
	return &memSnapshotRepo{snapshots: make(map[uuid.UUID]*models.HostFileSnapshot)}
}

func (r *memSnapshotRepo) Create(ctx context.Context, snapshot *models.HostFileSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshots[snapshot.ID] = snapshot
	return nil
}
func (r *memSnapshotRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.HostFileSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.snapshots[id]
	if !ok {
		return nil, repository.ErrSnapshotNotFound
	}
	return s, nil
}
func (r *memSnapshotRepo) GetByScanResultAndType(ctx context.Context, scanResultID uuid.UUID, fileType models.FileType) (*models.HostFileSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.snapshots {
		if s.ScanResultID == scanResultID && s.FileType == fileType {
			return s, nil
		}
	}
	return nil, fmt.Errorf("snapshot not found")
}
func (r *memSnapshotRepo) GetLatestByHostAndType(ctx context.Context, hostID uuid.UUID, fileType models.FileType) (*models.HostFileSnapshot, error) {
	return nil, repository.ErrScanJobNotFound
}
func (r *memSnapshotRepo) ListByHostAndType(ctx context.Context, hostID uuid.UUID, fileType models.FileType, limit int) ([]models.HostFileSnapshot, error) {
	return nil, nil
}
func (r *memSnapshotRepo) CreateChange(ctx context.Context, change *models.HostFileChange) error {
	return nil
}

type memHostRepo struct {
	mu    sync.RWMutex
	hosts map[uuid.UUID]*models.Host
}

func newMemHostRepo(host *models.Host) *memHostRepo {
	return &memHostRepo{hosts: map[uuid.UUID]*models.Host{host.ID: host}}
}

func (r *memHostRepo) Create(ctx context.Context, host *models.Host) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hosts[host.ID] = host
	return nil
}
func (r *memHostRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Host, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.hosts[id]
	if !ok {
		return nil, repository.ErrHostNotFound
	}
	return h, nil
}
func (r *memHostRepo) GetByHostname(ctx context.Context, hostname string) (*models.Host, error) {
	return nil, repository.ErrHostNotFound
}
func (r *memHostRepo) List(ctx context.Context) ([]models.Host, error)     { return nil, nil }
func (r *memHostRepo) ListPaginated(ctx context.Context, filters repository.HostFilters, page, limit int) ([]models.Host, int, error) {
	return nil, 0, nil
}
func (r *memHostRepo) Update(ctx context.Context, host *models.Host) error { return nil }
func (r *memHostRepo) Delete(ctx context.Context, id uuid.UUID) error      { return nil }

type memBaselineRepo struct {
	baselines []models.MasterBaseline
}

func (r *memBaselineRepo) Create(ctx context.Context, baseline *models.MasterBaseline) error {
	return nil
}
func (r *memBaselineRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.MasterBaseline, error) {
	return nil, repository.ErrBaselineNotFound
}
func (r *memBaselineRepo) List(ctx context.Context, filters repository.BaselineFilters) ([]models.MasterBaseline, error) {
	var out []models.MasterBaseline
	for _, b := range r.baselines {
		if filters.OSType != nil && b.OSType != *filters.OSType {
			continue
		}
		if filters.FileType != nil && b.FileType != *filters.FileType {
			continue
		}
		if filters.Version != nil && b.Version != *filters.Version {
			continue
		}
		if filters.IsActive != nil && b.IsActive != *filters.IsActive {
			continue
		}
		out = append(out, b)
	}
	return out, nil
}
func (r *memBaselineRepo) Update(ctx context.Context, baseline *models.MasterBaseline) error {
	return nil
}
func (r *memBaselineRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (r *memBaselineRepo) CreateVersion(ctx context.Context, version *models.MasterBaselineVersion) error {
	return nil
}
func (r *memBaselineRepo) CreateVersionedEntries(ctx context.Context, osType models.OSType, fileType models.FileType, version int, entries []repository.BaselineEntryInput, createdBy, description string) error {
	return nil
}
func (r *memBaselineRepo) SetActiveVersion(ctx context.Context, osType models.OSType, fileType models.FileType, version int) error {
	return nil
}
func (r *memBaselineRepo) DeactivateScope(ctx context.Context, osType models.OSType, fileType models.FileType, version int) error {
	return nil
}
func (r *memBaselineRepo) ListVersions(ctx context.Context) ([]repository.BaselineVersionSummary, error) {
	return nil, nil
}
func (r *memBaselineRepo) ListVersionsPaginated(ctx context.Context, page, limit int) ([]repository.BaselineVersionSummary, int, error) {
	return nil, 0, nil
}

type memDeviationRepo struct {
	deviations []models.AllowedDeviation
}

func (r *memDeviationRepo) Create(ctx context.Context, deviation *models.AllowedDeviation) error {
	return nil
}
func (r *memDeviationRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.AllowedDeviation, error) {
	return nil, repository.ErrDeviationNotFound
}
func (r *memDeviationRepo) GetByHostFileKey(ctx context.Context, hostname string, fileType models.FileType, entryKey string) (*models.AllowedDeviation, error) {
	for i := range r.deviations {
		d := &r.deviations[i]
		if d.Hostname == hostname && d.FileType == fileType && d.EntryKey == entryKey {
			return d, nil
		}
	}
	return nil, repository.ErrDeviationNotFound
}
func (r *memDeviationRepo) List(ctx context.Context, filters repository.DeviationFilters) ([]models.AllowedDeviation, error) {
	return r.deviations, nil
}
func (r *memDeviationRepo) ListPaginated(ctx context.Context, filters repository.DeviationFilters, page, limit int) ([]models.AllowedDeviation, int, error) {
	return r.deviations, len(r.deviations), nil
}
func (r *memDeviationRepo) CountDeviations(ctx context.Context, filters repository.DeviationFilters) (active, inactive int, err error) {
	for _, d := range r.deviations {
		if d.IsActive {
			active++
		} else {
			inactive++
		}
	}
	return active, inactive, nil
}
func (r *memDeviationRepo) Update(ctx context.Context, deviation *models.AllowedDeviation) error {
	return nil
}
func (r *memDeviationRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

type memIncidentRepo struct {
	mu        sync.Mutex
	incidents []models.Incident
}

func (r *memIncidentRepo) Create(ctx context.Context, incident *models.Incident) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.incidents = append(r.incidents, *incident)
	return nil
}

func (r *memIncidentRepo) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.incidents)
}
func (r *memIncidentRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Incident, error) {
	return nil, repository.ErrIncidentNotFound
}
func (r *memIncidentRepo) List(ctx context.Context, filters repository.IncidentFilters) ([]models.Incident, error) {
	return r.incidents, nil
}
func (r *memIncidentRepo) Update(ctx context.Context, incident *models.Incident) error { return nil }

func TestCompareScanResult_CreatesIncidentForDeviation(t *testing.T) {
	ctx := context.Background()
	hostID := uuid.New()
	scanResultID := uuid.New()
	scanJobID := uuid.New()

	host := &models.Host{
		ID:        hostID,
		Hostname:  "host001.example.com",
		OSType:    models.OSTypeLinux,
		OSVersion: "7.1",
	}

	scanRepo := newMemScanRepo()
	scanRepo.CreateScanResult(ctx, &models.ScanResult{
		ID:        scanResultID,
		ScanJobID: scanJobID,
		HostID:    hostID,
	})

	snapshotRepo := newMemSnapshotRepo()
	snapshotRepo.Create(ctx, &models.HostFileSnapshot{
		ID:           uuid.New(),
		ScanResultID: scanResultID,
		HostID:       hostID,
		ScanJobID:    scanJobID,
		FileType:     models.FileTypePasswd,
		RawContent:   "root:x:0:0:root:/root:/bin/bash\nadmin:x:1000:1000:admin:/home/admin:/bin/bash",
	})

	baselineRepo := &memBaselineRepo{
		baselines: []models.MasterBaseline{
			{
				ID:         uuid.New(),
				OSType:     models.OSTypeLinux,
				FileType:   models.FileTypePasswd,
				EntryKey:   "root",
				EntryValue: "x:0:0:root:/root:/bin/bash",
				Version:    7,
				IsActive:   true,

			},
		},
	}

	incidentRepo := &memIncidentRepo{}

	comparison := NewDefaultComparisonService(
		scanRepo,
		snapshotRepo,
		newMemHostRepo(host),
		baselineRepo,
		&memDeviationRepo{},
		incidentRepo,
	)

	if err := comparison.CompareScanResult(ctx, scanResultID); err != nil {
		t.Fatalf("CompareScanResult failed: %v", err)
	}

	if len(incidentRepo.incidents) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(incidentRepo.incidents))
	}

	incident := incidentRepo.incidents[0]
	if incident.EntryKey != "admin" {
		t.Errorf("expected incident for admin, got %s", incident.EntryKey)
	}
	if incident.Status != models.IncidentStatusOpen {
		t.Errorf("expected incident status open, got %s", incident.Status)
	}

	result, _ := scanRepo.GetScanResult(ctx, scanResultID)
	if result.Status != models.ScanResultStatusDeviationFound {
		t.Errorf("expected scan result status deviation_found, got %s", result.Status)
	}
	if result.ProcessingStatus != models.ScanProcessingStatusProcessed {
		t.Errorf("expected processing status processed, got %s", result.ProcessingStatus)
	}
}

func TestCompareScanResult_AllowedDeviationDoesNotCreateIncident(t *testing.T) {
	ctx := context.Background()
	hostID := uuid.New()
	scanResultID := uuid.New()
	scanJobID := uuid.New()

	host := &models.Host{
		ID:        hostID,
		Hostname:  "host001.example.com",
		OSType:    models.OSTypeLinux,
		OSVersion: "7.1",
	}

	scanRepo := newMemScanRepo()
	scanRepo.CreateScanResult(ctx, &models.ScanResult{
		ID:        scanResultID,
		ScanJobID: scanJobID,
		HostID:    hostID,
	})

	entryValue := "x:0:0:admin:/home/admin:/bin/bash"
	snapshotRepo := newMemSnapshotRepo()
	snapshotRepo.Create(ctx, &models.HostFileSnapshot{
		ID:           uuid.New(),
		ScanResultID: scanResultID,
		HostID:       hostID,
		ScanJobID:    scanJobID,
		FileType:     models.FileTypePasswd,
		RawContent:   "root:x:0:0:root:/root:/bin/bash\nadmin:" + entryValue,
	})

	baselineRepo := &memBaselineRepo{
		baselines: []models.MasterBaseline{
			{
				ID:         uuid.New(),
				OSType:     models.OSTypeLinux,
				FileType:   models.FileTypePasswd,
				EntryKey:   "root",
				EntryValue: "x:0:0:root:/root:/bin/bash",
				Version:    7,
				IsActive:   true,

			},
		},
	}

	deviationRepo := &memDeviationRepo{
		deviations: []models.AllowedDeviation{
			{
				ID:            uuid.New(),
				Hostname:      "host001.example.com",
				FileType:      models.FileTypePasswd,
				EntryKey:      "admin",
				EntryValue:    &entryValue,
				Justification: "service account",
				ApprovedBy:    "admin",
				ApprovedAt:    time.Now().UTC(),
				IsActive:      true,
			},
		},
	}

	incidentRepo := &memIncidentRepo{}

	comparison := NewDefaultComparisonService(
		scanRepo,
		snapshotRepo,
		newMemHostRepo(host),
		baselineRepo,
		deviationRepo,
		incidentRepo,
	)

	if err := comparison.CompareScanResult(ctx, scanResultID); err != nil {
		t.Fatalf("CompareScanResult failed: %v", err)
	}

	if len(incidentRepo.incidents) != 0 {
		t.Fatalf("expected 0 incidents, got %d", len(incidentRepo.incidents))
	}

	result, _ := scanRepo.GetScanResult(ctx, scanResultID)
	if result.Status != models.ScanResultStatusSuccess {
		t.Errorf("expected scan result status success, got %s", result.Status)
	}
}
func TestCompareScanResult_NoBaselineForMajorVersion(t *testing.T) {
	ctx := context.Background()
	hostID := uuid.New()
	scanResultID := uuid.New()
	scanJobID := uuid.New()

	host := &models.Host{
		ID:        hostID,
		Hostname:  "host001.example.com",
		OSType:    models.OSTypeLinux,
		OSVersion: "8.10",
	}

	scanRepo := newMemScanRepo()
	scanRepo.CreateScanResult(ctx, &models.ScanResult{
		ID:        scanResultID,
		ScanJobID: scanJobID,
		HostID:    hostID,
	})

	snapshotRepo := newMemSnapshotRepo()
	snapshotRepo.Create(ctx, &models.HostFileSnapshot{
		ID:           uuid.New(),
		ScanResultID: scanResultID,
		HostID:       hostID,
		ScanJobID:    scanJobID,
		FileType:     models.FileTypePasswd,
		RawContent:   "root:x:0:0:root:/root:/bin/bash\nadmin:x:1000:1000:admin:/home/admin:/bin/bash",
	})

	// Active baseline exists only for major version 7, host is version 8.
	baselineRepo := &memBaselineRepo{
		baselines: []models.MasterBaseline{
			{
				ID:         uuid.New(),
				OSType:     models.OSTypeLinux,
				FileType:   models.FileTypePasswd,
				EntryKey:   "root",
				EntryValue: "x:0:0:root:/root:/bin/bash",
				Version:    7,
				IsActive:   true,

			},
		},
	}

	incidentRepo := &memIncidentRepo{}
	comparison := NewDefaultComparisonService(
		scanRepo,
		snapshotRepo,
		newMemHostRepo(host),
		baselineRepo,
		&memDeviationRepo{},
		incidentRepo,
	)

	if err := comparison.CompareScanResult(ctx, scanResultID); err != nil {
		t.Fatalf("CompareScanResult failed: %v", err)
	}

	if len(incidentRepo.incidents) != 0 {
		t.Fatalf("expected 0 incidents when no baseline for major version, got %d", len(incidentRepo.incidents))
	}

	result, _ := scanRepo.GetScanResult(ctx, scanResultID)
	if result.Status != models.ScanResultStatusSuccess {
		t.Errorf("expected scan result status success, got %s", result.Status)
	}
	if !result.NoBaseline {
		t.Errorf("expected no_baseline flag to be true")
	}
	if result.BaselineVersionAtScan == nil || *result.BaselineVersionAtScan != 8 {
		t.Errorf("expected baseline_version_at_scan = 8, got %v", result.BaselineVersionAtScan)
	}
}
