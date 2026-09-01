package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"ulas-service/internal/aap"
	"ulas-service/internal/config"
	"ulas-service/internal/repository"
	"ulas-service/models"
)

// ErrScanAlreadyRunning is returned when a scan is attempted while another is active.
var ErrScanAlreadyRunning = errors.New("a scan is already running")

// ScanService defines business operations for scans.
type ScanService interface {
	InitiateScan(ctx context.Context, limit, initiatedBy string) (*models.ScanJob, error)
	ProcessCallback(ctx context.Context, payload models.CallbackPayload) error
	ProcessCallbackEnvelope(ctx context.Context, ansibleJobID string, hosts []models.CallbackPayload, failedHosts []string) error
	RecordFailedHosts(ctx context.Context, ansibleJobID string, hostnames []string) error
	ListScanJobs(ctx context.Context) ([]models.ScanJob, error)
	ListScanJobsPaginated(ctx context.Context, page, limit int, onlyWithDeviations bool, search string, fromDate, toDate *time.Time) (*PaginatedScanJobs, error)
	GetScanDetail(ctx context.Context, scanJobID uuid.UUID, includeIncidents bool) (*ScanDetail, error)
	GetScanDetailPaginated(ctx context.Context, scanJobID uuid.UUID, page, limit int, includeIncidents bool) (*PaginatedScanDetail, error)
	GetHostResult(ctx context.Context, scanJobID, hostID uuid.UUID) (*HostScanDetail, error)
	GetScanJobByAnsibleJobID(ctx context.Context, ansibleJobID string) (*models.ScanJob, error)
	PollActiveScans(ctx context.Context) error
}

// ScanDetail aggregates a scan job with its per-host results and incidents.
type ScanDetail struct {
	Job     *models.ScanJob  `json:"job"`
	Results []HostScanDetail `json:"results"`
}

// PaginatedScanJobs is a page of scan jobs.
type PaginatedScanJobs struct {
	Items []models.ScanJobSummary `json:"items"`
	Total int                     `json:"total"`
	Page  int                     `json:"page"`
	Limit int                     `json:"limit"`
}

// PaginatedScanDetail is a scan job with a page of per-host results and incidents.
type PaginatedScanDetail struct {
	Job     *models.ScanJob  `json:"job"`
	Results []HostScanDetail `json:"results"`
	Total   int              `json:"total"`
	Page    int              `json:"page"`
	Limit   int              `json:"limit"`
}

// HostScanDetail is a scan result enriched with host and incident info.
type HostScanDetail struct {
	models.ScanResult
	HostID      uuid.UUID         `json:"host_id"`
	Hostname    string            `json:"hostname"`
	OSType      string            `json:"os_type,omitempty"`
	OSVersion   string            `json:"os_version,omitempty"`
	OSName      string            `json:"os_name,omitempty"`
	Environment string            `json:"environment,omitempty"`
	Datacenter  string            `json:"datacenter,omitempty"`
	Incidents   []models.Incident `json:"incidents"`
}

// callbackJob is a unit of work queued by ProcessCallbackEnvelope for the
// shared background worker pool.
type callbackJob struct {
	payload models.CallbackPayload
}

// DefaultScanService is the default implementation of ScanService.
type DefaultScanService struct {
	scanRepo          repository.ScanRepository
	hostRepo          repository.HostRepository
	snapshotRepo      repository.SnapshotRepository
	incidentRepo      repository.IncidentRepository
	baselineRepo      repository.BaselineRepository
	aapClient         *aap.Client
	comparisonService ComparisonService
	jobTemplateName   string
	backendBaseURL    string
	appStage          string
	cfg               *config.AppConfig
	scanLock          sync.Mutex
	scanLocked        atomic.Bool
	callbackQueue     chan callbackJob
}

// NewDefaultScanService creates a new scan service.
func NewDefaultScanService(
	scanRepo repository.ScanRepository,
	hostRepo repository.HostRepository,
	snapshotRepo repository.SnapshotRepository,
	incidentRepo repository.IncidentRepository,
	baselineRepo repository.BaselineRepository,
	aapClient *aap.Client,
	comparisonService ComparisonService,
	jobTemplateName string,
	backendBaseURL string,
	appStage string,
	cfg *config.AppConfig,
) *DefaultScanService {
	const callbackWorkerCount = 20
	const callbackQueueSize = 100000

	svc := &DefaultScanService{
		scanRepo:          scanRepo,
		hostRepo:          hostRepo,
		snapshotRepo:      snapshotRepo,
		incidentRepo:      incidentRepo,
		baselineRepo:      baselineRepo,
		aapClient:         aapClient,
		comparisonService: comparisonService,
		jobTemplateName:   jobTemplateName,
		backendBaseURL:    backendBaseURL,
		appStage:          appStage,
		cfg:               cfg,
		callbackQueue:     make(chan callbackJob, callbackQueueSize),
	}

	for i := 0; i < callbackWorkerCount; i++ {
		go svc.callbackWorker()
	}

	return svc
}

// callbackWorker drains the callback queue and processes each host.
// A fixed-size pool prevents a burst of callbacks from saturating the database.
func (s *DefaultScanService) callbackWorker() {
	for job := range s.callbackQueue {
		if err := s.ProcessCallback(context.Background(), job.payload); err != nil {
			slog.Error("failed to process queued host callback",
				"ansible_job_id", job.payload.JobID,
				"hostname", job.payload.MachineName,
				"error", err,
			)
		}
	}
}

// releaseScanLock releases the single-scan lock if it is currently held.
func (s *DefaultScanService) releaseScanLock() {
	if s.scanLocked.CompareAndSwap(true, false) {
		s.scanLock.Unlock()
	}
}

// InitiateScan resolves the configured AAP template, launches the job, and creates a ScanJob record.
// Only one scan can be active at a time; if another scan is running, an error is returned.
func (s *DefaultScanService) InitiateScan(ctx context.Context, limit, initiatedBy string) (*models.ScanJob, error) {
	if !s.scanLock.TryLock() {
		return nil, ErrScanAlreadyRunning
	}
	s.scanLocked.Store(true)

	// Check the database for an active scan so a restart does not lose the lock.
	active, err := s.scanRepo.HasActiveScanJob(ctx)
	if err != nil {
		s.releaseScanLock()
		return nil, fmt.Errorf("check active scan: %w", err)
	}
	if active {
		s.releaseScanLock()
		return nil, ErrScanAlreadyRunning
	}

	templateID, err := s.aapClient.FindJobTemplateID(ctx, s.jobTemplateName)
	if err != nil {
		s.releaseScanLock()
		fmt.Println(err)
		return nil, fmt.Errorf("resolve template %q: %w", s.jobTemplateName, err)
	}

	now := time.Now().UTC()
	job := &models.ScanJob{
		ID:            uuid.New(),
		JobTemplateID: templateID,
		Limit:         limit,
		Status:        models.ScanJobStatusInitiating,
		InitiatedBy:   initiatedBy,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.scanRepo.CreateScanJob(ctx, job); err != nil {
		s.releaseScanLock()
		return nil, fmt.Errorf("create scan job: %w", err)
	}

	// Capture the currently active baseline versions so the portal can display
	// which /etc/passwd and /etc/group master files will be used for deviation checks.
	if err := s.captureBaselineSnapshot(ctx, job); err != nil {
		s.releaseScanLock()
		return nil, fmt.Errorf("capture baseline snapshot: %w", err)
	}

	extraVars, err := json.Marshal(map[string]string{
		"env":               s.appStage,
		"back_end_base_url": s.backendBaseURL,
	})
	if err != nil {
		s.releaseScanLock()
		return nil, fmt.Errorf("marshal extra vars: %w", err)
	}

	launchReq := aap.LaunchRequest{
		Limit:     limit,
		ExtraVars: string(extraVars),
	}
	aapJobID, err := s.aapClient.LaunchJobTemplate(ctx, templateID, launchReq)
	if err != nil {
		job.Status = models.ScanJobStatusFailed
		job.ErrorMessage = err.Error()
		job.UpdatedAt = time.Now().UTC()
		_ = s.scanRepo.UpdateScanJob(ctx, job)
		s.releaseScanLock()
		return nil, fmt.Errorf("launch scan: %w", err)
	}

	job.AnsibleJobID = fmt.Sprintf("%d", aapJobID)
	job.Status = models.ScanJobStatusRunning
	job.StartedAt = &now
	job.UpdatedAt = now
	if err := s.scanRepo.UpdateScanJobLaunch(ctx, job); err != nil {
		s.releaseScanLock()
		return nil, fmt.Errorf("update scan job launch: %w", err)
	}

	return job, nil
}

// ProcessCallback handles the JSON payload posted by AAP after a host is scanned.
func (s *DefaultScanService) ProcessCallback(ctx context.Context, payload models.CallbackPayload) error {
	fmt.Printf("[CALLBACK] job=%s hostname=%s\n", payload.JobID, payload.MachineName)

	scanJob, err := s.scanRepo.GetScanJobByAnsibleJobID(ctx, payload.JobID)
	if err != nil {
		fmt.Printf("[CALLBACK ERROR] find scan job: job=%s hostname=%s err=%v\n", payload.JobID, payload.MachineName, err)
		slog.Error("failed to find scan job for callback",
			"ansible_job_id", payload.JobID,
			"hostname", payload.MachineName,
			"error", err,
		)
		return fmt.Errorf("find scan job for callback: %w", err)
	}

	host, err := s.resolveOrCreateHost(ctx, payload)
	if err != nil {
		fmt.Printf("[CALLBACK ERROR] resolve host: job=%s hostname=%s err=%v\n", payload.JobID, payload.MachineName, err)
		slog.Error("failed to resolve host for callback",
			"ansible_job_id", payload.JobID,
			"hostname", payload.MachineName,
			"error", err,
		)
		return fmt.Errorf("resolve host: %w", err)
	}

	now := time.Now().UTC()
	result := &models.ScanResult{
		ID:               uuid.New(),
		ScanJobID:        scanJob.ID,
		HostID:           host.ID,
		Status:           models.ScanResultStatusPending,
		ProcessingStatus: models.ScanProcessingStatusPending,
		ReceivedAt:       &now,
		CreatedAt:        now,
	}
	if err := s.scanRepo.CreateScanResult(ctx, result); err != nil {
		fmt.Printf("[CALLBACK ERROR] create scan result: job=%s host=%s err=%v\n", scanJob.ID, host.Hostname, err)
		slog.Error("failed to create scan result",
			"scan_job_id", scanJob.ID,
			"host_id", host.ID,
			"hostname", host.Hostname,
			"error", err,
		)
		return fmt.Errorf("create scan result: %w", err)
	}

	if err := s.storeSnapshots(ctx, scanJob.ID, host.ID, result.ID, payload); err != nil {
		fmt.Printf("[CALLBACK ERROR] store snapshots: job=%s host=%s err=%v\n", scanJob.ID, host.Hostname, err)
		slog.Error("failed to store snapshots",
			"scan_job_id", scanJob.ID,
			"host_id", host.ID,
			"hostname", host.Hostname,
			"error", err,
		)
		return fmt.Errorf("store snapshots: %w", err)
	}

	if err := s.scanRepo.IncrementScanJobCounters(ctx, scanJob.ID, 1, 1, 0); err != nil {
		fmt.Printf("[CALLBACK ERROR] update counters: job=%s host=%s err=%v\n", scanJob.ID, host.Hostname, err)
		slog.Error("failed to update scan job counters",
			"scan_job_id", scanJob.ID,
			"host_id", host.ID,
			"hostname", host.Hostname,
			"error", err,
		)
		return fmt.Errorf("update scan job counters: %w", err)
	}

	fmt.Printf("[CALLBACK OK] job=%s host=%s result=%s\n", scanJob.ID, host.Hostname, result.ID)

	// Run comparison asynchronously so the callback returns quickly.
	go func() {
		ctx := context.Background()
		if err := s.comparisonService.CompareScanResult(ctx, result.ID); err != nil {
			slog.Error("comparison failed",
				"scan_result_id", result.ID,
				"host_id", host.ID,
				"hostname", host.Hostname,
				"error", err,
			)
		}
	}()

	return nil
}

// ProcessCallbackEnvelope accepts a batch of host callbacks and queues them for the
// shared background worker pool. The HTTP handler returns immediately, so AAP does not
// time out while the database catches up.
func (s *DefaultScanService) ProcessCallbackEnvelope(ctx context.Context, ansibleJobID string, hosts []models.CallbackPayload, failedHosts []string) error {
	// Drop empty/blank failed host entries so an empty failed_hosts array (or one
	// containing blank strings) does not create phantom failed-host rows.
	failedHosts = filterNonEmpty(failedHosts)

	fmt.Printf("[ENVELOPE] job=%s hosts=%d failed=%d queue_depth=%d\n", ansibleJobID, len(hosts), len(failedHosts), len(s.callbackQueue))

	// Validate the scan job once per envelope so the HTTP handler can return a
	// clear 404 for unknown jobs without queueing useless work.
	if _, err := s.scanRepo.GetScanJobByAnsibleJobID(ctx, ansibleJobID); err != nil {
		if errors.Is(err, repository.ErrScanJobNotFound) {
			return repository.ErrScanJobNotFound
		}
		return fmt.Errorf("find scan job for envelope: %w", err)
	}

	// Record failed hosts asynchronously so the HTTP response is not blocked.
	if len(failedHosts) > 0 {
		go func() {
			if err := s.RecordFailedHosts(context.Background(), ansibleJobID, failedHosts); err != nil {
				slog.Error("failed to record failed hosts",
					"ansible_job_id", ansibleJobID,
					"error", err,
				)
			}
		}()
	}

	for _, h := range hosts {
		// Skip callbacks with no hostname to avoid creating empty inventory rows.
		if strings.TrimSpace(h.MachineName) == "" {
			continue
		}
		h.JobID = ansibleJobID
		s.callbackQueue <- callbackJob{payload: h}
	}

	return nil
}

// RecordFailedHosts stores the list of hostnames that AAP could not reach during a scan.
func (s *DefaultScanService) RecordFailedHosts(ctx context.Context, ansibleJobID string, hostnames []string) error {
	if len(hostnames) == 0 {
		return nil
	}

	scanJob, err := s.scanRepo.GetScanJobByAnsibleJobID(ctx, ansibleJobID)
	if err != nil {
		return fmt.Errorf("find scan job for failed hosts: %w", err)
	}

	if err := s.scanRepo.AppendFailedHosts(ctx, scanJob.ID, hostnames); err != nil {
		return fmt.Errorf("append failed hosts: %w", err)
	}
	return nil
}

// GetScanJobByAnsibleJobID returns a scan job by its AAP job ID.
func (s *DefaultScanService) GetScanJobByAnsibleJobID(ctx context.Context, ansibleJobID string) (*models.ScanJob, error) {
	job, err := s.scanRepo.GetScanJobByAnsibleJobID(ctx, ansibleJobID)
	if err != nil {
		return nil, fmt.Errorf("get scan job by ansible job id: %w", err)
	}
	return job, nil
}

// ListScanJobs returns all scan jobs.
func (s *DefaultScanService) ListScanJobs(ctx context.Context) ([]models.ScanJob, error) {
	jobs, err := s.scanRepo.ListScanJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list scan jobs: %w", err)
	}
	return jobs, nil
}

// ListScanJobsPaginated returns a page of scan jobs.
func (s *DefaultScanService) ListScanJobsPaginated(ctx context.Context, page, limit int, onlyWithDeviations bool, search string, fromDate, toDate *time.Time) (*PaginatedScanJobs, error) {
	jobs, total, err := s.scanRepo.ListScanJobsPaginatedWithDeviationCounts(ctx, page, limit, onlyWithDeviations, search, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("list scan jobs paginated: %w", err)
	}
	return &PaginatedScanJobs{
		Items: jobs,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

// GetScanDetail returns a scan job with its per-host results. Incidents are only
// included when includeIncidents is true; otherwise use GetHostResult to load
// incidents for a specific host on demand.
func (s *DefaultScanService) GetScanDetail(ctx context.Context, scanJobID uuid.UUID, includeIncidents bool) (*ScanDetail, error) {
	job, err := s.scanRepo.GetScanJobByID(ctx, scanJobID)
	if err != nil {
		return nil, fmt.Errorf("get scan job: %w", err)
	}

	results, err := s.scanRepo.ListScanResultsByJobID(ctx, scanJobID)
	if err != nil {
		return nil, fmt.Errorf("list scan results: %w", err)
	}

	details, err := s.buildHostScanDetails(ctx, results, includeIncidents)
	if err != nil {
		return nil, err
	}

	return &ScanDetail{
		Job:     job,
		Results: details,
	}, nil
}

// GetScanDetailPaginated returns a paginated scan job with its per-host results.
// Incidents are only included when includeIncidents is true.
func (s *DefaultScanService) GetScanDetailPaginated(ctx context.Context, scanJobID uuid.UUID, page, limit int, includeIncidents bool) (*PaginatedScanDetail, error) {
	job, err := s.scanRepo.GetScanJobByID(ctx, scanJobID)
	if err != nil {
		return nil, fmt.Errorf("get scan job: %w", err)
	}

	results, total, err := s.scanRepo.ListScanResultsByJobIDPaginated(ctx, scanJobID, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list scan results paginated: %w", err)
	}

	details, err := s.buildHostScanDetails(ctx, results, includeIncidents)
	if err != nil {
		return nil, err
	}

	return &PaginatedScanDetail{
		Job:     job,
		Results: details,
		Total:   total,
		Page:    page,
		Limit:   limit,
	}, nil
}

// GetHostResult returns the full details for a single host within a scan job,
// including incidents.
func (s *DefaultScanService) GetHostResult(ctx context.Context, scanJobID, hostID uuid.UUID) (*HostScanDetail, error) {
	result, err := s.scanRepo.GetScanResultByJobAndHost(ctx, scanJobID, hostID)
	if err != nil {
		return nil, fmt.Errorf("get scan result: %w", err)
	}

	host, err := s.hostRepo.GetByID(ctx, result.HostID)
	if err != nil {
		host = &models.Host{Hostname: "unknown"}
	}

	incidents, err := s.incidentRepo.List(ctx, repository.IncidentFilters{ScanResultID: &result.ID})
	if err != nil {
		return nil, fmt.Errorf("list incidents: %w", err)
	}
	if incidents == nil {
		incidents = []models.Incident{}
	}

	if result.AllowedDeviations == nil {
		result.AllowedDeviations = models.AllowedDeviations{}
	}

	return &HostScanDetail{
		ScanResult:  *result,
		HostID:      result.HostID,
		Hostname:    host.Hostname,
		OSType:      string(host.OSType),
		OSVersion:   host.OSVersion,
		Environment: host.Environment,
		Datacenter:  host.Datacenter,
		Incidents:   incidents,
	}, nil
}

// buildHostScanDetails builds host results, optionally loading incidents per host.
func (s *DefaultScanService) buildHostScanDetails(ctx context.Context, results []models.ScanResult, includeIncidents bool) ([]HostScanDetail, error) {
	hostCache := make(map[uuid.UUID]*models.Host)
	var details []HostScanDetail
	for i := range results {
		result := &results[i]
		host, ok := hostCache[result.HostID]
		if !ok {
			var err error
			host, err = s.hostRepo.GetByID(ctx, result.HostID)
			if err != nil {
				host = &models.Host{Hostname: "unknown"}
			}
			hostCache[result.HostID] = host
		}

		var incidents []models.Incident
		if includeIncidents {
			var err error
			incidents, err = s.incidentRepo.List(ctx, repository.IncidentFilters{ScanResultID: &result.ID})
			if err != nil {
				return nil, fmt.Errorf("list incidents: %w", err)
			}
		}
		if incidents == nil {
			incidents = []models.Incident{}
		}

		details = append(details, HostScanDetail{
			ScanResult:  *result,
			HostID:      result.HostID,
			Hostname:    host.Hostname,
			OSType:      string(host.OSType),
			OSVersion:   host.OSVersion,
			Environment: host.Environment,
			Datacenter:  host.Datacenter,
			Incidents:   incidents,
		})
	}
	return details, nil
}

// PollActiveScans checks AAP status for running scan jobs and updates them.
// Jobs that stay in a non-terminal state for too long (AAP unreachable, no callbacks,
// or stuck initiating) are marked as failed so they do not block new scans forever.
func (s *DefaultScanService) PollActiveScans(ctx context.Context) error {
	jobs, err := s.scanRepo.ListScanJobs(ctx)
	if err != nil {
		return fmt.Errorf("list scan jobs: %w", err)
	}

	staleTimeout := 48 * time.Hour
	if s.cfg != nil {
		staleTimeout = s.cfg.StaleScanTimeoutDuration()
	}
	now := time.Now().UTC()

	for i := range jobs {
		job := &jobs[i]
		if job.Status != models.ScanJobStatusRunning && job.Status != models.ScanJobStatusInitiating {
			continue
		}

		// Jobs stuck initiating without ever getting an AAP job ID are stale.
		if job.AnsibleJobID == "" {
			if now.Sub(job.CreatedAt) > staleTimeout {
				job.Status = models.ScanJobStatusFailed
				job.ErrorMessage = "scan job stuck initiating; marked as stale"
				job.CompletedAt = &now
				job.UpdatedAt = now
				if err := s.scanRepo.UpdateScanJobStatus(ctx, job); err != nil {
					slog.Error("failed to mark stale scan job", "scan_job_id", job.ID, "error", err)
				}
				s.releaseScanLock()
			}
			continue
		}

		aapJobID, err := parseAAPJobID(job.AnsibleJobID)
		if err == nil {
			aapJob, err := s.aapClient.GetJob(ctx, aapJobID)
			if err != nil {
				if errors.Is(err, aap.ErrJobNotFound) {
					job.Status = models.ScanJobStatusFailed
					job.ErrorMessage = "AAP job not found; marked as stale"
					job.CompletedAt = &now
					job.UpdatedAt = now
					if err := s.scanRepo.UpdateScanJobStatus(ctx, job); err != nil {
						slog.Error("failed to mark stale scan job", "scan_job_id", job.ID, "error", err)
					}
					// Release the single-scan lock so a new scan can be started.
					s.releaseScanLock()
				}
				// If AAP is unreachable, fall through to the generic staleness check below.
			} else {
				newStatus := mapAAPStatus(aapJob.Status)
				if newStatus != job.Status {
					job.Status = newStatus
					job.UpdatedAt = now
					if newStatus == models.ScanJobStatusCompleted || newStatus == models.ScanJobStatusFailed || newStatus == models.ScanJobStatusCancelled {
						job.CompletedAt = &now
						// Release the single-scan lock when the job reaches a terminal state.
						s.releaseScanLock()
					}
					if aapJob.Failed && job.ErrorMessage == "" {
						job.ErrorMessage = "AAP job failed"
					}
					if err := s.scanRepo.UpdateScanJobStatus(ctx, job); err != nil {
						slog.Error("failed to update scan job status", "scan_job_id", job.ID, "error", err)
					}
				}
			}
		}
		// If parseAAPJobID failed (non-integer ansible_job_id), we also fall through
		// to the generic staleness check so seed/test jobs do not stay stuck forever.

		// Mark jobs that have not reached a terminal state within the timeout as stale.
		// This covers AAP unreachable, lost callbacks, unparseable AAP job IDs, or any other hang.
		if job.Status == models.ScanJobStatusRunning || job.Status == models.ScanJobStatusInitiating {
			ref := job.CreatedAt
			if job.StartedAt != nil {
				ref = *job.StartedAt
			}
			if now.Sub(ref) > staleTimeout {
				job.Status = models.ScanJobStatusFailed
				job.ErrorMessage = "scan job did not reach a terminal status within the stale timeout"
				job.CompletedAt = &now
				job.UpdatedAt = now
				if err := s.scanRepo.UpdateScanJobStatus(ctx, job); err != nil {
					slog.Error("failed to mark stale scan job", "scan_job_id", job.ID, "error", err)
				}
				s.releaseScanLock()
			}
		}
	}

	return nil
}

func parseAAPJobID(id string) (int, error) {
	var jobID int
	_, err := fmt.Sscanf(id, "%d", &jobID)
	return jobID, err
}

func mapAAPStatus(status string) models.ScanJobStatus {
	switch status {
	case string(models.AAPJobStatusSuccessful):
		return models.ScanJobStatusCompleted
	case string(models.AAPJobStatusFailed), string(models.AAPJobStatusError):
		return models.ScanJobStatusFailed
	case string(models.AAPJobStatusCanceled):
		return models.ScanJobStatusCancelled
	case string(models.AAPJobStatusPending), string(models.AAPJobStatusWaiting), string(models.AAPJobStatusRunning):
		return models.ScanJobStatusRunning
	default:
		return models.ScanJobStatusRunning
	}
}

func (s *DefaultScanService) resolveOrCreateHost(ctx context.Context, payload models.CallbackPayload) (*models.Host, error) {
	if strings.TrimSpace(payload.MachineName) == "" {
		return nil, fmt.Errorf("machine_name is empty")
	}

	host, err := s.hostRepo.GetByHostname(ctx, payload.MachineName)
	if err != nil && err != repository.ErrHostNotFound {
		return nil, err
	}

	now := time.Now().UTC()
	if host != nil {
		// Refresh host metadata from the latest callback.
		host.OSType = payload.MachineType
		host.OSName = payload.OSName
		host.OSVersion = payload.OSVersion
		host.Environment = payload.Environment
		host.Datacenter = payload.Datacenter
		host.UpdatedAt = now
		if err := s.hostRepo.Update(ctx, host); err != nil {
			return nil, err
		}
		return host, nil
	}

	newHost := &models.Host{
		ID:          uuid.New(),
		Hostname:    payload.MachineName,
		OSType:      payload.MachineType,
		OSName:      payload.OSName,
		OSVersion:   payload.OSVersion,
		Environment: payload.Environment,
		Datacenter:  payload.Datacenter,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.hostRepo.Create(ctx, newHost); err != nil {
		return nil, err
	}
	return newHost, nil
}

func (s *DefaultScanService) storeSnapshots(ctx context.Context, scanJobID, hostID, scanResultID uuid.UUID, payload models.CallbackPayload) error {
	now := time.Now().UTC()

	if err := s.storeSnapshot(ctx, scanJobID, hostID, scanResultID, models.FileTypePasswd, payload.PasswdFile, now); err != nil {
		return err
	}
	if err := s.storeSnapshot(ctx, scanJobID, hostID, scanResultID, models.FileTypeGroup, payload.GroupFile, now); err != nil {
		return err
	}

	return nil
}

func (s *DefaultScanService) storeSnapshot(ctx context.Context, scanJobID, hostID, scanResultID uuid.UUID, fileType models.FileType, entries []map[string]string, now time.Time) error {
	rawContent := entriesToRawContent(entries)
	snapshot := &models.HostFileSnapshot{
		ID:           uuid.New(),
		ScanResultID: scanResultID,
		HostID:       hostID,
		ScanJobID:    scanJobID,
		FileType:     fileType,
		RawContent:   rawContent,
		LineCount:    len(entries),
		SnapshotAt:   now,
	}
	if err := s.snapshotRepo.Create(ctx, snapshot); err != nil {
		return fmt.Errorf("create %s snapshot: %w", fileType, err)
	}

	previous, err := s.snapshotRepo.GetLatestByHostAndType(ctx, hostID, fileType)
	if err != nil {
		// No previous snapshot means this file is newly seen.
		change := &models.HostFileChange{
			ID:               uuid.New(),
			HostID:           hostID,
			FileType:         fileType,
			ChangeType:       models.HostFileChangeTypeAdded,
			CurrentContent:   &rawContent,
			CurrentScanJobID: scanJobID,
			DetectedAt:       now,
		}
		if err := s.snapshotRepo.CreateChange(ctx, change); err != nil {
			return fmt.Errorf("record added change: %w", err)
		}
		return nil
	}

	if previous.RawContent == rawContent {
		return nil
	}

	change := &models.HostFileChange{
		ID:                uuid.New(),
		HostID:            hostID,
		FileType:          fileType,
		ChangeType:        models.HostFileChangeTypeModified,
		PreviousContent:   &previous.RawContent,
		CurrentContent:    &rawContent,
		PreviousScanJobID: &previous.ScanJobID,
		CurrentScanJobID:  scanJobID,
		DetectedAt:        now,
	}
	if err := s.snapshotRepo.CreateChange(ctx, change); err != nil {
		return fmt.Errorf("record modified change: %w", err)
	}
	return nil
}

func entriesToRawContent(entries []map[string]string) string {
	var lines []string
	for _, entry := range entries {
		for key, value := range entry {
			lines = append(lines, fmt.Sprintf("%s:%s", key, value))
		}
	}
	return strings.Join(lines, "\n")
}

// filterNonEmpty returns a new slice with only non-empty, non-blank strings.
func filterNonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}

// captureBaselineSnapshot records the active master file versions for a scan job.
func (s *DefaultScanService) captureBaselineSnapshot(ctx context.Context, job *models.ScanJob) error {
	versions, err := s.baselineRepo.ListVersions(ctx)
	if err != nil {
		return fmt.Errorf("list baseline versions: %w", err)
	}

	var snapshot []models.BaselineVersionSnapshot
	for _, v := range versions {
		if v.IsActive {
			snapshot = append(snapshot, v.ToSnapshot())
		}
	}

	job.BaselineSnapshot = snapshot
	if err := s.scanRepo.UpdateScanJob(ctx, job); err != nil {
		return fmt.Errorf("save baseline snapshot: %w", err)
	}
	return nil
}
