package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"ulas-service/models"
)

type fakeScanServiceForReport struct{ detail *ScanDetail }

func (f *fakeScanServiceForReport) InitiateScan(context.Context, string, string) (*models.ScanJob, error) {
	return nil, nil
}
func (f *fakeScanServiceForReport) ProcessCallback(context.Context, models.CallbackPayload) error {
	return nil
}
func (f *fakeScanServiceForReport) ProcessCallbackEnvelope(context.Context, string, []models.CallbackPayload, []string) error {
	return nil
}
func (f *fakeScanServiceForReport) RecordFailedHosts(context.Context, string, []string) error { return nil }
func (f *fakeScanServiceForReport) ListScanJobs(context.Context) ([]models.ScanJob, error) {
	return nil, nil
}
func (f *fakeScanServiceForReport) ListScanJobsPaginated(context.Context, int, int, bool, string) (*PaginatedScanJobs, error) {
	return nil, nil
}
func (f *fakeScanServiceForReport) GetScanDetail(context.Context, uuid.UUID, bool) (*ScanDetail, error) {
	return f.detail, nil
}
func (f *fakeScanServiceForReport) GetScanDetailPaginated(context.Context, uuid.UUID, int, int, bool) (*PaginatedScanDetail, error) {
	return nil, nil
}
func (f *fakeScanServiceForReport) GetHostResult(context.Context, uuid.UUID, uuid.UUID) (*HostScanDetail, error) {
	return nil, nil
}
func (f *fakeScanServiceForReport) GetScanJobByAnsibleJobID(context.Context, string) (*models.ScanJob, error) {
	return nil, nil
}
func (f *fakeScanServiceForReport) PollActiveScans(context.Context) error { return nil }

func TestGenerateScanReport_LongText(t *testing.T) {
	jobID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	hostID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	longLimit := strings.Repeat("host-pattern-very-long-name-that-keeps-going,", 30)
	longError := strings.Repeat("This error message is extremely long and should wrap instead of running off the page. ", 10)
	longDescription := strings.Repeat("A very long baseline description that should wrap correctly within the description column. ", 5)
	longEntry := strings.Repeat("username:with:a:very:long:entry:that:exceeds:the:column:width:and:should:wrap:naturally,", 4)
	longExpected := strings.Repeat("expected-value-that-is-way-too-long-to-fit-in-one-line-and-must-wrap,", 5)
	longActual := strings.Repeat("actual-value-that-is-way-too-long-to-fit-in-one-line-and-must-wrap,", 5)
	longHostname := strings.Repeat("very-long-hostname-that-needs-wrapping-", 4)

	fake := &fakeScanServiceForReport{
		detail: &ScanDetail{
			Job: &models.ScanJob{
				ID:               jobID,
				AnsibleJobID:     "12345",
				JobTemplateID:    42,
				Limit:            longLimit,
				Status:           models.ScanJobStatusCompleted,
				SuccessfulHosts:  1,
				FailedHosts:      1,
				FailedHostNames:  []string{"failed-host-with-a-very-long-name-that-should-wrap-in-the-host-results-table"},
				ErrorMessage:     longError,
				BaselineSnapshot: []models.BaselineVersionSnapshot{{Description: longDescription, OSType: models.OSTypeLinux, FileType: models.FileTypePasswd, Version: 1, EntryCount: 5}},
				CreatedAt:        time.Now(),
			},
			Results: []HostScanDetail{
				{
					ScanResult: models.ScanResult{
						ID:                uuid.New(),
						ScanJobID:         jobID,
						HostID:            hostID,
						Status:            models.ScanResultStatusSuccess,
						DeviationsFound:   2,
						AllowedDeviations: models.AllowedDeviations{{EntryKey: "allowed-1"}},
					},
					Hostname:    longHostname,
					OSType:      "linux",
					OSVersion:   "8.9",
					Environment: "production",
					Datacenter:  "dc1",
					Incidents: []models.Incident{
						{
							IncidentNumber: "INC001",
							EntryKey:       longEntry,
							ExpectedValue:  ptr(longExpected),
							ActualValue:    longActual,
							Severity:       models.IncidentSeverityHigh,
							Status:         models.IncidentStatusOpen,
						},
					},
				},
			},
		},
	}

	fake.detail.Results[0].AllowedDeviations = models.AllowedDeviations{
		{
			FileType:      models.FileTypePasswd,
			EntryKey:      longEntry,
			ExpectedValue: longExpected,
			ActualValue:   longActual,
		},
	}

	svc := NewReportService(fake)
	data, err := svc.GenerateScanReport(context.Background(), jobID)
	if err != nil {
		t.Fatalf("generate report: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty PDF")
	}
}

func ptr(s string) *string { return &s }
