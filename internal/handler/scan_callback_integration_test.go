//go:build integration

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ulas-service/internal/config"
	"ulas-service/internal/repository"
	"ulas-service/internal/service"
	"ulas-service/models"
)

// noopComparison satisfies service.ComparisonService without running comparisons.
type noopComparison struct{}

func (noopComparison) CompareScanResult(context.Context, uuid.UUID) error { return nil }

// TestScanCallback_FailedHostsStored_DB runs the real handler → service →
// repository path against a live database and verifies the final
// failed_hosts callback is persisted (both string and numeric element shapes).
func TestScanCallback_FailedHostsStored_DB(t *testing.T) {
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
	defer sqlDB.Close()

	scanRepo := repository.NewPgScanRepository(sqlDB)
	hostRepo := repository.NewPgHostRepository(sqlDB)
	snapshotRepo := repository.NewPgSnapshotRepository(sqlDB)
	incidentRepo := repository.NewPgIncidentRepository(sqlDB)
	baselineRepo := repository.NewPgBaselineRepository(sqlDB)

	svc := service.NewDefaultScanService(
		scanRepo,
		hostRepo,
		snapshotRepo,
		incidentRepo,
		baselineRepo,
		nil,
		nil,
		noopComparison{},
		"test-template",
		"test-template-solaris",
		"http://localhost:8080",
		"test",
		cfg,
	)
	h := NewScanHandler(svc)

	cases := []struct {
		name     string
		body     string
		expected []string
	}{
		{
			name:     "string hostnames",
			body:     `{"ansible_job_id": "%s", "failed_hosts": ["test.com", "new1.com"]}`,
			expected: []string{"test.com", "new1.com"},
		},
		{
			name:     "numeric host ids",
			body:     `{"ansible_job_id": "%s", "failed_hosts": [123, 345]}`,
			expected: []string{"123", "345"},
		},
		{
			// Some AAP versions key the final summary as job_id instead of ansible_job_id.
			name:     "summary keyed by job_id",
			body:     `{"job_id": "%s", "failed_hosts": ["test.zit.com"]}`,
			expected: []string{"test.zit.com"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			ansibleJobID := fmt.Sprintf("e2e-failed-%s", uuid.New().String())
			now := time.Now().UTC()
			job := &models.ScanJob{
				ID:            uuid.New(),
				AnsibleJobID:  ansibleJobID,
				JobTemplateID: 1,
				OSType:        models.OSTypeLinux,
				Status:        models.ScanJobStatusRunning,
				InitiatedBy:   "e2e-test",
				StartedAt:     &now,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			if err := scanRepo.CreateScanJob(ctx, job); err != nil {
				t.Fatalf("create scan job: %v", err)
			}
			defer sqlDB.ExecContext(ctx, `DELETE FROM scan_jobs WHERE id = $1`, job.ID)

			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			body := fmt.Sprintf(tc.body, ansibleJobID)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/callbacks/scan", bytes.NewBufferString(body))
			c.Request.Header.Set("Content-Type", "application/json")

			h.ScanCallback(c)
			// Invoke WriteHeaderNow because gin buffers the status code when a
			// handler is called directly instead of through the engine.
			c.Writer.WriteHeaderNow()

			if w.Code != http.StatusAccepted {
				t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
			}

			// The failed-host write happens in a goroutine; poll until it lands.
			var stored *models.ScanJob
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				stored, err = scanRepo.GetScanJobByID(ctx, job.ID)
				if err != nil {
					t.Fatalf("get scan job: %v", err)
				}
				if stored.FailedHosts == len(tc.expected) {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}

			if stored.FailedHosts != len(tc.expected) {
				t.Fatalf("failed_hosts = %d, want %d", stored.FailedHosts, len(tc.expected))
			}
			if len(stored.FailedHostNames) != len(tc.expected) {
				b, _ := json.Marshal(stored.FailedHostNames)
				t.Fatalf("failed_host_names = %s, want %v", b, tc.expected)
			}
			for i, name := range tc.expected {
				if stored.FailedHostNames[i] != name {
					t.Fatalf("failed_host_names[%d] = %q, want %q", i, stored.FailedHostNames[i], name)
				}
			}
		})
	}
}
