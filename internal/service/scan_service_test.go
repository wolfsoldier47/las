package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"ulas-service/internal/aap"
	"ulas-service/internal/config"
	"ulas-service/internal/repository"
	"ulas-service/models"
)

// testScanRepo is an in-memory scan repository that can return a pre-registered job.
type testScanRepo struct {
	mu              sync.Mutex
	jobs            map[string]*models.ScanJob
	results         map[uuid.UUID]*models.ScanResult
	listJobs        []models.ScanJob
	updatedStatuses map[uuid.UUID]*models.ScanJob
}

func newTestScanRepo() *testScanRepo {
	return &testScanRepo{
		jobs:            make(map[string]*models.ScanJob),
		results:         make(map[uuid.UUID]*models.ScanResult),
		updatedStatuses: make(map[uuid.UUID]*models.ScanJob),
	}
}

func (r *testScanRepo) registerJob(job *models.ScanJob) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs[job.AnsibleJobID] = job
}

func (r *testScanRepo) setListJobs(jobs []models.ScanJob) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listJobs = jobs
}

func (r *testScanRepo) getUpdatedJob(id uuid.UUID) *models.ScanJob {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.updatedStatuses[id]
}

func (r *testScanRepo) getResultCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.results)
}

func (r *testScanRepo) CreateScanJob(ctx context.Context, job *models.ScanJob) error { return nil }
func (r *testScanRepo) GetScanJobByID(ctx context.Context, id uuid.UUID) (*models.ScanJob, error) {
	return nil, repository.ErrScanJobNotFound
}
func (r *testScanRepo) GetScanJobByAnsibleJobID(ctx context.Context, ansibleJobID string) (*models.ScanJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.jobs[ansibleJobID]
	if !ok {
		return nil, repository.ErrScanJobNotFound
	}
	return job, nil
}
func (r *testScanRepo) ListScanJobs(ctx context.Context) ([]models.ScanJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.listJobs, nil
}
func (r *testScanRepo) ListScanJobsPaginated(ctx context.Context, page, limit int) ([]models.ScanJob, int, error) {
	return nil, 0, nil
}
func (r *testScanRepo) ListScanJobsPaginatedWithDeviationCounts(ctx context.Context, page, limit int, onlyWithDeviations bool, search string, fromDate, toDate *time.Time) ([]models.ScanJobSummary, int, error) {
	return nil, 0, nil
}
func (r *testScanRepo) HasActiveScanJob(ctx context.Context) (bool, error) { return false, nil }
func (r *testScanRepo) UpdateScanJob(ctx context.Context, job *models.ScanJob) error { return nil }
func (r *testScanRepo) UpdateScanJobStatus(ctx context.Context, job *models.ScanJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updatedStatuses[job.ID] = job
	return nil
}
func (r *testScanRepo) UpdateScanJobLaunch(ctx context.Context, job *models.ScanJob) error { return nil }
func (r *testScanRepo) CreateScanResult(ctx context.Context, result *models.ScanResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.results[result.ID] = result
	return nil
}
func (r *testScanRepo) GetScanResult(ctx context.Context, id uuid.UUID) (*models.ScanResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.results[id]
	if !ok {
		return nil, repository.ErrScanJobNotFound
	}
	return res, nil
}
func (r *testScanRepo) GetScanResultByJobAndHost(ctx context.Context, scanJobID, hostID uuid.UUID) (*models.ScanResult, error) {
	return nil, nil
}
func (r *testScanRepo) ListScanResultsByJobID(ctx context.Context, scanJobID uuid.UUID) ([]models.ScanResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]models.ScanResult, 0, len(r.results))
	for _, res := range r.results {
		if res.ScanJobID == scanJobID {
			out = append(out, *res)
		}
	}
	return out, nil
}
func (r *testScanRepo) ListScanResultsByJobIDPaginated(ctx context.Context, scanJobID uuid.UUID, page, limit int) ([]models.ScanResult, int, error) {
	return nil, 0, nil
}
func (r *testScanRepo) UpdateScanResult(ctx context.Context, result *models.ScanResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.results[result.ID] = result
	return nil
}
func (r *testScanRepo) IncrementScanJobCounters(ctx context.Context, id uuid.UUID, callbacks, successful, failed int) error {
	return nil
}
func (r *testScanRepo) AppendFailedHosts(ctx context.Context, id uuid.UUID, hostnames []string) error { return nil }

// noopComparisonService skips deviation comparison so the benchmark measures callback ingestion only.
type noopComparisonService struct{}

func (s *noopComparisonService) CompareScanResult(ctx context.Context, scanResultID uuid.UUID) error { return nil }

// generateHosts creates n identical-looking callback payloads with unique hostnames.
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

func TestProcessCallbackEnvelope_20kHosts(t *testing.T) {
	const hostCount = 20_000
	const ansibleJobID = "247024"

	ctx := context.Background()

	scanRepo := newTestScanRepo()
	scanRepo.registerJob(&models.ScanJob{
		ID:           uuid.New(),
		AnsibleJobID: ansibleJobID,
		Status:       models.ScanJobStatusRunning,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	})

	hostRepo := newMemHostRepo(&models.Host{ID: uuid.New(), Hostname: "existing.example.com"})
	snapshotRepo := newMemSnapshotRepo()
	baselineRepo := &memBaselineRepo{}
	incidentRepo := &memIncidentRepo{}

	svc := NewDefaultScanService(
		scanRepo,
		hostRepo,
		snapshotRepo,
		incidentRepo,
		baselineRepo,
		nil,
		nil,
		&noopComparisonService{},
		"test-template",
		"",
		"http://localhost:8080",
		"test",
		nil,
	)

	hosts := generateHosts(hostCount)

	start := time.Now()
	if err := svc.ProcessCallbackEnvelope(ctx, ansibleJobID, hosts, nil); err != nil {
		t.Fatalf("process callback envelope: %v", err)
	}

	// Wait for the worker pool to finish processing all hosts.
	done := make(chan struct{})
	go func() {
		for {
			if scanRepo.getResultCount() == hostCount {
				close(done)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	select {
	case <-done:
	case <-time.After(60 * time.Second):
		t.Fatalf("timed out waiting for %d scan results, got %d", hostCount, scanRepo.getResultCount())
	}

	elapsed := time.Since(start)
	t.Logf("Processed %d hosts in %v (%.2f hosts/sec)", hostCount, elapsed, float64(hostCount)/elapsed.Seconds())

	if scanRepo.getResultCount() != hostCount {
		t.Errorf("expected %d scan results, got %d", hostCount, scanRepo.getResultCount())
	}
}

// TestProcessCallbackEnvelope_100NonCompliantHosts processes 100 hosts in a single scan
// where every host has an extra /etc/passwd entry compared to the active baseline.
// It verifies that each host gets a deviation_found result and one incident is created per host.
func TestProcessCallbackEnvelope_100NonCompliantHosts(t *testing.T) {
	const hostCount = 100
	const ansibleJobID = "job-100-noncompliant"

	ctx := context.Background()

	scanRepo := newTestScanRepo()
	scanJobID := uuid.New()
	scanRepo.registerJob(&models.ScanJob{
		ID:           scanJobID,
		AnsibleJobID: ansibleJobID,
		Status:       models.ScanJobStatusRunning,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	})

	// Active baselines for RHEL 8. The actual files from generateHosts include
	// an extra "daemon" entry in /etc/passwd that is not in the baseline.
	baselineRepo := &memBaselineRepo{
		baselines: []models.MasterBaseline{
			{ID: uuid.New(), OSType: models.OSTypeLinux, FileType: models.FileTypePasswd, EntryKey: "root", EntryValue: "x:0:0:root:/root:/bin/bash", Version: 8, IsActive: true},
			{ID: uuid.New(), OSType: models.OSTypeLinux, FileType: models.FileTypePasswd, EntryKey: "bin", EntryValue: "x:1:1:bin:/bin:/sbin/nologin", Version: 8, IsActive: true},
			{ID: uuid.New(), OSType: models.OSTypeLinux, FileType: models.FileTypeGroup, EntryKey: "root", EntryValue: "x:0", Version: 8, IsActive: true},
			{ID: uuid.New(), OSType: models.OSTypeLinux, FileType: models.FileTypeGroup, EntryKey: "bin", EntryValue: "x:1", Version: 8, IsActive: true},
			{ID: uuid.New(), OSType: models.OSTypeLinux, FileType: models.FileTypeGroup, EntryKey: "daemon", EntryValue: "x:2", Version: 8, IsActive: true},
		},
	}

	hostRepo := newMemHostRepo(&models.Host{ID: uuid.New(), Hostname: "existing.example.com"})
	snapshotRepo := newMemSnapshotRepo()
	incidentRepo := &memIncidentRepo{}
	deviationRepo := &memDeviationRepo{}

	comparisonService := NewDefaultComparisonService(
		scanRepo,
		snapshotRepo,
		hostRepo,
		baselineRepo,
		deviationRepo,
		incidentRepo,
	)

	svc := NewDefaultScanService(
		scanRepo,
		hostRepo,
		snapshotRepo,
		incidentRepo,
		baselineRepo,
		nil,
		nil,
		comparisonService,
		"test-template",
		"",
		"http://localhost:8080",
		"test",
		nil,
	)

	hosts := generateHosts(hostCount)

	start := time.Now()
	if err := svc.ProcessCallbackEnvelope(ctx, ansibleJobID, hosts, nil); err != nil {
		t.Fatalf("process callback envelope: %v", err)
	}

	// Wait for all callbacks to be ingested.
	resultsDone := make(chan struct{})
	go func() {
		for {
			if scanRepo.getResultCount() == hostCount {
				close(resultsDone)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	select {
	case <-resultsDone:
	case <-time.After(60 * time.Second):
		t.Fatalf("timed out waiting for %d scan results, got %d", hostCount, scanRepo.getResultCount())
	}

	// Wait for the asynchronous comparison goroutines to finish.
	incidentsDone := make(chan struct{})
	go func() {
		for {
			if incidentRepo.Count() == hostCount {
				close(incidentsDone)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	select {
	case <-incidentsDone:
	case <-time.After(60 * time.Second):
		t.Fatalf("timed out waiting for %d incidents, got %d", hostCount, incidentRepo.Count())
	}

	elapsed := time.Since(start)
	t.Logf("Processed %d non-compliant hosts in %v (%.2f hosts/sec)", hostCount, elapsed, float64(hostCount)/elapsed.Seconds())

	results, err := scanRepo.ListScanResultsByJobID(ctx, scanJobID)
	if err != nil {
		t.Fatalf("list scan results: %v", err)
	}
	if len(results) != hostCount {
		t.Fatalf("expected %d scan results, got %d", hostCount, len(results))
	}

	deviationCount := 0
	for _, res := range results {
		if res.Status != models.ScanResultStatusDeviationFound {
			t.Errorf("expected status deviation_found for host %s, got %s", res.HostID, res.Status)
		}
		if res.NoBaseline {
			t.Errorf("did not expect no_baseline flag for host %s", res.HostID)
		}
		if res.BaselineVersionAtScan == nil || *res.BaselineVersionAtScan != 8 {
			t.Errorf("expected baseline_version_at_scan=8 for host %s, got %v", res.HostID, res.BaselineVersionAtScan)
		}
		if res.DeviationsFound > 0 {
			deviationCount++
		}
	}
	if deviationCount != hostCount {
		t.Errorf("expected all %d hosts to have deviations, got %d", hostCount, deviationCount)
	}

	if incidentRepo.Count() != hostCount {
		t.Errorf("expected %d incidents, got %d", hostCount, incidentRepo.Count())
	}
}

func TestPollActiveScans_MarksStaleRunningJob(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/controller/v2/jobs/123/" {
			_ = json.NewEncoder(w).Encode(aap.JobResponse{ID: 123, Status: "running", Failed: false})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := aap.NewClient(server.URL+"/api/controller/v2/", "user", "pass")

	scanRepo := newTestScanRepo()
	hostRepo := newMemHostRepo(&models.Host{ID: uuid.New(), Hostname: "h1.example.com"})
	snapshotRepo := newMemSnapshotRepo()
	baselineRepo := &memBaselineRepo{}
	incidentRepo := &memIncidentRepo{}

	cfg := &config.AppConfig{StaleScanTimeout: 1}

	svc := NewDefaultScanService(
		scanRepo,
		hostRepo,
		snapshotRepo,
		incidentRepo,
		baselineRepo,
		client,
		nil,
		&noopComparisonService{},
		"test-template",
		"",
		"http://localhost:8080",
		"test",
		cfg,
	)

	startedAt := time.Now().UTC().Add(-2 * time.Minute)
	jobID := uuid.New()
	job := models.ScanJob{
		ID:            jobID,
		AnsibleJobID:  "123",
		JobTemplateID: 1,
		Status:        models.ScanJobStatusRunning,
		StartedAt:     &startedAt,
		CreatedAt:     startedAt,
		UpdatedAt:     startedAt,
	}
	scanRepo.setListJobs([]models.ScanJob{job})

	if err := svc.PollActiveScans(ctx); err != nil {
		t.Fatalf("PollActiveScans failed: %v", err)
	}

	updated := scanRepo.getUpdatedJob(jobID)
	if updated == nil {
		t.Fatalf("expected scan job to be updated")
	}
	if updated.Status != models.ScanJobStatusFailed {
		t.Errorf("expected status %q, got %q", models.ScanJobStatusFailed, updated.Status)
	}
	if updated.ErrorMessage == "" {
		t.Errorf("expected error message to be set")
	}
	if updated.CompletedAt == nil {
		t.Errorf("expected completed_at to be set")
	}
}

func TestPollActiveScans_MarksStaleJobWithNonIntegerAAPID(t *testing.T) {
	ctx := context.Background()

	scanRepo := newTestScanRepo()
	hostRepo := newMemHostRepo(&models.Host{ID: uuid.New(), Hostname: "h1.example.com"})
	snapshotRepo := newMemSnapshotRepo()
	baselineRepo := &memBaselineRepo{}
	incidentRepo := &memIncidentRepo{}

	cfg := &config.AppConfig{StaleScanTimeout: 1}

	svc := NewDefaultScanService(
		scanRepo,
		hostRepo,
		snapshotRepo,
		incidentRepo,
		baselineRepo,
		nil, // no AAP client needed because the ID cannot be parsed
		nil,
		&noopComparisonService{},
		"test-template",
		"",
		"http://localhost:8080",
		"test",
		cfg,
	)

	createdAt := time.Now().UTC().Add(-2 * time.Minute)
	jobID := uuid.New()
	job := models.ScanJob{
		ID:            jobID,
		AnsibleJobID:  "seed-100-noncompliant",
		JobTemplateID: 1,
		Status:        models.ScanJobStatusRunning,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}
	scanRepo.setListJobs([]models.ScanJob{job})

	if err := svc.PollActiveScans(ctx); err != nil {
		t.Fatalf("PollActiveScans failed: %v", err)
	}

	updated := scanRepo.getUpdatedJob(jobID)
	if updated == nil {
		t.Fatalf("expected scan job to be updated")
	}
	if updated.Status != models.ScanJobStatusFailed {
		t.Errorf("expected status %q, got %q", models.ScanJobStatusFailed, updated.Status)
	}
	if updated.CompletedAt == nil {
		t.Errorf("expected completed_at to be set")
	}
}
