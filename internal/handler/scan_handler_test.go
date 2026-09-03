package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ulas-service/internal/repository"
	"ulas-service/internal/service"
	"ulas-service/models"
)

// fakeScanService records envelope calls so tests can assert what the handler dispatched.
type fakeScanService struct {
	envelopeJobID     string
	envelopeHosts     []models.CallbackPayload
	envelopeFailed    []string
	envelopeCalled    bool
	envelopeErr       error
}

func (f *fakeScanService) InitiateScan(context.Context, string, string, models.OSType) (*models.ScanJob, error) {
	return nil, nil
}
func (f *fakeScanService) ProcessCallback(context.Context, models.CallbackPayload) error { return nil }
func (f *fakeScanService) ProcessCallbackEnvelope(_ context.Context, ansibleJobID string, hosts []models.CallbackPayload, failedHosts []string) error {
	f.envelopeCalled = true
	f.envelopeJobID = ansibleJobID
	f.envelopeHosts = hosts
	f.envelopeFailed = failedHosts
	return f.envelopeErr
}
func (f *fakeScanService) RecordFailedHosts(context.Context, string, []string) error { return nil }
func (f *fakeScanService) ListScanJobs(context.Context) ([]models.ScanJob, error)    { return nil, nil }
func (f *fakeScanService) ListScanJobsPaginated(context.Context, int, int, bool, string, *time.Time, *time.Time) (*service.PaginatedScanJobs, error) {
	return nil, nil
}
func (f *fakeScanService) GetScanDetail(context.Context, uuid.UUID, bool) (*service.ScanDetail, error) {
	return nil, nil
}
func (f *fakeScanService) GetScanDetailPaginated(context.Context, uuid.UUID, int, int, bool) (*service.PaginatedScanDetail, error) {
	return nil, nil
}
func (f *fakeScanService) GetHostResult(context.Context, uuid.UUID, uuid.UUID) (*service.HostScanDetail, error) {
	return nil, nil
}
func (f *fakeScanService) GetScanJobByAnsibleJobID(context.Context, string) (*models.ScanJob, error) {
	return nil, nil
}
func (f *fakeScanService) PollActiveScans(context.Context) error { return nil }

func TestScanCallback_EnvelopeWithStringFailedHosts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &fakeScanService{}
	h := NewScanHandler(fake)

	body := `{"ansible_job_id": "438578", "failed_hosts": ["123", "345"]}`
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/callbacks/scan", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ScanCallback(c)
	c.Writer.WriteHeaderNow()

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !fake.envelopeCalled {
		t.Fatal("expected ProcessCallbackEnvelope to be called")
	}
	if fake.envelopeJobID != "438578" {
		t.Errorf("expected job id 438578, got %q", fake.envelopeJobID)
	}
	if len(fake.envelopeFailed) != 2 || fake.envelopeFailed[0] != "123" || fake.envelopeFailed[1] != "345" {
		t.Errorf("unexpected failed hosts: %v", fake.envelopeFailed)
	}
	if len(fake.envelopeHosts) != 0 {
		t.Errorf("expected no hosts in final summary, got %d", len(fake.envelopeHosts))
	}
}

func TestScanCallback_EnvelopeWithNumericFailedHosts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &fakeScanService{}
	h := NewScanHandler(fake)

	body := `{"ansible_job_id": "438578", "failed_hosts": [123, 345]}`
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/callbacks/scan", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ScanCallback(c)
	c.Writer.WriteHeaderNow()

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(fake.envelopeFailed) != 2 || fake.envelopeFailed[0] != "123" || fake.envelopeFailed[1] != "345" {
		t.Errorf("unexpected failed hosts: %v", fake.envelopeFailed)
	}
}

func TestScanCallback_EnvelopeUnknownJobReturns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &fakeScanService{envelopeErr: repository.ErrScanJobNotFound}
	h := NewScanHandler(fake)

	body := `{"ansible_job_id": "999999", "failed_hosts": ["123"]}`
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/callbacks/scan", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ScanCallback(c)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

// The final failed_hosts summary may key the job as "job_id" instead of
// "ansible_job_id" depending on the AAP version sending it.
func TestScanCallback_SummaryKeyedByJobID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &fakeScanService{}
	h := NewScanHandler(fake)

	body := `{"job_id": "451853", "failed_hosts": ["test.zit.com"]}`
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/callbacks/scan", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ScanCallback(c)
	c.Writer.WriteHeaderNow()

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !fake.envelopeCalled {
		t.Fatal("expected ProcessCallbackEnvelope to be called")
	}
	if fake.envelopeJobID != "451853" {
		t.Errorf("expected job id 451853, got %q", fake.envelopeJobID)
	}
	if len(fake.envelopeFailed) != 1 || fake.envelopeFailed[0] != "test.zit.com" {
		t.Errorf("unexpected failed hosts: %v", fake.envelopeFailed)
	}
	if len(fake.envelopeHosts) != 0 {
		t.Errorf("expected no hosts in final summary, got %d", len(fake.envelopeHosts))
	}
}

// A legacy host payload keyed by job_id (with machine_name) must NOT be
// swallowed by the job_id fallback — it still flows through the legacy path
// with its host data intact.
func TestScanCallback_LegacyHostPayloadKeyedByJobID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &fakeScanService{}
	h := NewScanHandler(fake)

	body := `{"job_id": "451853", "machine_name": "host1.example.com", "os_type": "Linux"}`
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/callbacks/scan", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ScanCallback(c)
	c.Writer.WriteHeaderNow()

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if fake.envelopeJobID != "451853" {
		t.Errorf("expected job id 451853, got %q", fake.envelopeJobID)
	}
	if len(fake.envelopeHosts) != 1 || fake.envelopeHosts[0].MachineName != "host1.example.com" {
		t.Errorf("legacy host payload lost: hosts=%v", fake.envelopeHosts)
	}
	if len(fake.envelopeFailed) != 0 {
		t.Errorf("unexpected failed hosts on legacy payload: %v", fake.envelopeFailed)
	}
}
