package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"ulas-service/internal/repository"
	"ulas-service/models"
)

// ComparisonService compares host file snapshots against baselines and deviations.
type ComparisonService interface {
	CompareScanResult(ctx context.Context, scanResultID uuid.UUID) error
}

// DefaultComparisonService is the default implementation of ComparisonService.
type DefaultComparisonService struct {
	scanRepo      repository.ScanRepository
	snapshotRepo  repository.SnapshotRepository
	hostRepo      repository.HostRepository
	baselineRepo  repository.BaselineRepository
	deviationRepo repository.DeviationRepository
	incidentRepo  repository.IncidentRepository
}

// NewDefaultComparisonService creates a new comparison service.
func NewDefaultComparisonService(
	scanRepo repository.ScanRepository,
	snapshotRepo repository.SnapshotRepository,
	hostRepo repository.HostRepository,
	baselineRepo repository.BaselineRepository,
	deviationRepo repository.DeviationRepository,
	incidentRepo repository.IncidentRepository,
) *DefaultComparisonService {
	return &DefaultComparisonService{
		scanRepo:      scanRepo,
		snapshotRepo:  snapshotRepo,
		hostRepo:      hostRepo,
		baselineRepo:  baselineRepo,
		deviationRepo: deviationRepo,
		incidentRepo:  incidentRepo,
	}
}

// CompareScanResult evaluates a scan result and creates incidents for unauthorized deviations.
func (s *DefaultComparisonService) CompareScanResult(ctx context.Context, scanResultID uuid.UUID) error {
	result, err := s.scanRepo.GetScanResult(ctx, scanResultID)
	if err != nil {
		return fmt.Errorf("get scan result: %w", err)
	}

	host, err := s.hostRepo.GetByID(ctx, result.HostID)
	if err != nil {
		return fmt.Errorf("get host: %w", err)
	}

	now := time.Now().UTC()
	result.ProcessingStatus = models.ScanProcessingStatusProcessing
	result.ProcessedAt = &now

	passwdSnapshot, err := s.snapshotRepo.GetByScanResultAndType(ctx, scanResultID, models.FileTypePasswd)
	if err != nil && !strings.Contains(err.Error(), "snapshot not found") {
		return fmt.Errorf("get passwd snapshot: %w", err)
	}

	groupSnapshot, err := s.snapshotRepo.GetByScanResultAndType(ctx, scanResultID, models.FileTypeGroup)
	if err != nil && !strings.Contains(err.Error(), "snapshot not found") {
		return fmt.Errorf("get group snapshot: %w", err)
	}

	deviationsFound := 0

	if passwdSnapshot != nil {
		actual := parseSnapshotContent(models.FileTypePasswd, passwdSnapshot.RawContent)
		count, err := s.compareFile(ctx, result, host, models.FileTypePasswd, actual)
		if err != nil {
			return fmt.Errorf("compare passwd: %w", err)
		}
		deviationsFound += count
	}

	if groupSnapshot != nil {
		actual := parseSnapshotContent(models.FileTypeGroup, groupSnapshot.RawContent)
		count, err := s.compareFile(ctx, result, host, models.FileTypeGroup, actual)
		if err != nil {
			return fmt.Errorf("compare group: %w", err)
		}
		deviationsFound += count
	}

	result.DeviationsFound = deviationsFound
	if deviationsFound > 0 {
		result.Status = models.ScanResultStatusDeviationFound
	} else {
		result.Status = models.ScanResultStatusSuccess
	}
	result.ProcessingStatus = models.ScanProcessingStatusProcessed

	if err := s.scanRepo.UpdateScanResult(ctx, result); err != nil {
		return fmt.Errorf("update scan result: %w", err)
	}

	return nil
}

func (s *DefaultComparisonService) compareFile(
	ctx context.Context,
	result *models.ScanResult,
	host *models.Host,
	fileType models.FileType,
	actual map[string]string,
) (int, error) {
	majorVersion := parseMajorVersion(host.OSVersion)
	if majorVersion == 0 {
		result.NoBaseline = true
		result.BaselineVersionAtScan = nil
		return 0, nil
	}

	active := true
	baselines, err := s.baselineRepo.List(ctx, repository.BaselineFilters{
		OSType:   &host.OSType,
		FileType: &fileType,
		Version:  &majorVersion,
		IsActive: &active,
	})
	if err != nil {
		return 0, fmt.Errorf("list baselines: %w", err)
	}

	slog.Debug("loaded active baselines for comparison",
		"host_id", host.ID,
		"hostname", host.Hostname,
		"os_type", host.OSType,
		"os_major_version", majorVersion,
		"file_type", fileType,
		"baseline_entries", len(baselines),
	)

	// Record the version we attempted to compare against.
	result.BaselineVersionAtScan = &majorVersion

	// No master baseline configured for this OS major version/file type.
	if len(baselines) == 0 {
		result.NoBaseline = true
		return 0, nil
	}

	// Build expected map from the active baselines. Baselines are global per OS type/version.
	expected := make(map[string]string)
	for i := range baselines {
		b := &baselines[i]
		expected[b.EntryKey] = b.EntryValue
	}

	deviations, err := s.deviationRepo.List(ctx, repository.DeviationFilters{
		Hostname: &host.Hostname,
		FileType: &fileType,
		Active:   &active,
	})
	if err != nil {
		return 0, fmt.Errorf("list deviations: %w", err)
	}

	allowed := make(map[string][]*models.AllowedDeviation)
	for i := range deviations {
		d := &deviations[i]
		if d.ExpiresAt == nil || d.ExpiresAt.After(time.Now().UTC()) {
			allowed[d.EntryKey] = append(allowed[d.EntryKey], d)
		}
	}

	deviationCount := 0

	// Check expected entries against actual.
	for key, expectedValue := range expected {
		actualValue, exists := actual[key]
		if !exists {
			// A missing expected entry is always a deviation; allowed deviations only
			// cover differences in values or extra entries, not absent baseline entries.
			if err := s.createIncident(ctx, result, host, fileType, key, &expectedValue, "", models.IncidentSeverityCritical); err != nil {
				return 0, err
			}
			deviationCount++
			continue
		}
		if actualValue != expectedValue {
			if s.isAllowed(allowed, key, actualValue) {
				result.AllowedDeviations = append(result.AllowedDeviations, models.AllowedDeviationFound{
					FileType:      fileType,
					EntryKey:      key,
					ExpectedValue: expectedValue,
					ActualValue:   actualValue,
				})
			} else {
				severity := severityForFileType(fileType)
				if err := s.createIncident(ctx, result, host, fileType, key, &expectedValue, actualValue, severity); err != nil {
					return 0, err
				}
				deviationCount++
			}
		}
	}

	// Check for extra entries not in baseline.
	for key, actualValue := range actual {
		if _, expected := expected[key]; !expected {
			if s.isAllowed(allowed, key, actualValue) {
				result.AllowedDeviations = append(result.AllowedDeviations, models.AllowedDeviationFound{
					FileType:    fileType,
					EntryKey:    key,
					ActualValue: actualValue,
				})
			} else {
				severity := severityForFileType(fileType)
				if err := s.createIncident(ctx, result, host, fileType, key, nil, actualValue, severity); err != nil {
					return 0, err
				}
				deviationCount++
			}
		}
	}

	slog.Debug("file comparison complete",
		"host_id", host.ID,
		"hostname", host.Hostname,
		"file_type", fileType,
		"expected_keys", len(expected),
		"actual_keys", len(actual),
		"deviations", deviationCount,
		"allowed_deviations", len(result.AllowedDeviations),
	)

	return deviationCount, nil
}

func (s *DefaultComparisonService) isAllowed(allowed map[string][]*models.AllowedDeviation, key, value string) bool {
	deviations, ok := allowed[key]
	if !ok {
		return false
	}
	for _, d := range deviations {
		if d.EntryValue == nil {
			return true
		}
		if *d.EntryValue == value {
			return true
		}
	}
	return false
}

func (s *DefaultComparisonService) createIncident(
	ctx context.Context,
	result *models.ScanResult,
	host *models.Host,
	fileType models.FileType,
	entryKey string,
	expectedValue *string,
	actualValue string,
	severity models.IncidentSeverity,
) error {
	now := time.Now().UTC()
	incident := &models.Incident{
		ID:             uuid.New(),
		IncidentNumber: fmt.Sprintf("INC-%s", uuid.New().String()[:8]),
		ScanResultID:   result.ID,
		HostID:         host.ID,
		FileType:       fileType,
		EntryKey:       entryKey,
		ExpectedValue:  expectedValue,
		ActualValue:    actualValue,
		Severity:       severity,
		Status:         models.IncidentStatusOpen,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.incidentRepo.Create(ctx, incident); err != nil {
		return fmt.Errorf("create incident: %w", err)
	}
	return nil
}

// parseMajorVersion extracts the leading integer from an OS version string such as "7.1" or "8.10".
func parseMajorVersion(osVersion string) int {
	if osVersion == "" {
		return 0
	}
	parts := strings.Split(osVersion, ".")
	major, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || major < 0 {
		return 0
	}
	return major
}

func parseSnapshotContent(fileType models.FileType, content string) map[string]string {
	result := make(map[string]string)
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]
		if fileType == models.FileTypeGroup {
			value = normalizeGroupSnapshotValue(value)
		}
		result[key] = value
	}
	return result
}

func normalizeGroupSnapshotValue(value string) string {
	fields := strings.Split(value, ":")
	if len(fields) < 2 {
		return value
	}
	base := fields[0] + ":" + fields[1]
	if len(fields) < 3 || strings.TrimSpace(fields[2]) == "" {
		return base
	}
	members := strings.Split(fields[2], ",")
	for i := range members {
		members[i] = strings.TrimSpace(members[i])
	}
	sort.Strings(members)
	return base + ":" + strings.Join(members, ",")
}

func severityForFileType(fileType models.FileType) models.IncidentSeverity {
	switch fileType {
	case models.FileTypePasswd:
		return models.IncidentSeverityCritical
	case models.FileTypeGroup:
		return models.IncidentSeverityHigh
	default:
		return models.IncidentSeverityMedium
	}
}
