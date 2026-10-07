package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"ulas-service/internal/repository"
	"ulas-service/models"
)

type memDeviationRepoForService struct {
	deviations []models.AllowedDeviation
}

func (r *memDeviationRepoForService) Create(ctx context.Context, deviation *models.AllowedDeviation) error {
	r.deviations = append(r.deviations, *deviation)
	return nil
}

func (r *memDeviationRepoForService) GetByID(ctx context.Context, id uuid.UUID) (*models.AllowedDeviation, error) {
	for i := range r.deviations {
		if r.deviations[i].ID == id {
			d := r.deviations[i]
			return &d, nil
		}
	}
	return nil, repository.ErrDeviationNotFound
}

func (r *memDeviationRepoForService) GetByHostFileKey(ctx context.Context, hostname string, fileType models.FileType, entryKey string) (*models.AllowedDeviation, error) {
	for i := range r.deviations {
		d := &r.deviations[i]
		if d.Hostname == hostname && d.FileType == fileType && d.EntryKey == entryKey {
			return d, nil
		}
	}
	return nil, repository.ErrDeviationNotFound
}

func (r *memDeviationRepoForService) List(ctx context.Context, filters repository.DeviationFilters) ([]models.AllowedDeviation, error) {
	return r.deviations, nil
}
func (r *memDeviationRepoForService) ListPaginated(ctx context.Context, filters repository.DeviationFilters, page, limit int) ([]models.AllowedDeviation, int, error) {
	return r.deviations, len(r.deviations), nil
}
func (r *memDeviationRepoForService) CountDeviations(ctx context.Context, filters repository.DeviationFilters) (active, inactive int, err error) {
	for _, d := range r.deviations {
		if d.IsActive {
			active++
		} else {
			inactive++
		}
	}
	return active, inactive, nil
}

func (r *memDeviationRepoForService) Update(ctx context.Context, deviation *models.AllowedDeviation) error {
	for i := range r.deviations {
		if r.deviations[i].ID == deviation.ID {
			r.deviations[i] = *deviation
			return nil
		}
	}
	return repository.ErrDeviationNotFound
}

func (r *memDeviationRepoForService) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (r *memDeviationRepoForService) SetApproved(ctx context.Context, id uuid.UUID, approver string) error {
	for i := range r.deviations {
		if r.deviations[i].ID == id {
			r.deviations[i].IsActive = true
			r.deviations[i].ApprovalStatus = "approved"
			r.deviations[i].ApprovedBy = approver
			r.deviations[i].ApprovedAt = time.Now().UTC()
			r.deviations[i].UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return repository.ErrDeviationNotFound
}

func (r *memDeviationRepoForService) SetRejected(ctx context.Context, id uuid.UUID) error {
	for i := range r.deviations {
		if r.deviations[i].ID == id && r.deviations[i].ApprovalStatus == "pending" {
			r.deviations[i].IsActive = false
			r.deviations[i].ApprovalStatus = "rejected"
			r.deviations[i].UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return repository.ErrDeviationNotFound
}

func (r *memDeviationRepoForService) ListPending(ctx context.Context) ([]models.AllowedDeviation, error) {
	var out []models.AllowedDeviation
	for _, d := range r.deviations {
		if d.ApprovalStatus == "pending" {
			out = append(out, d)
		}
	}
	return out, nil
}

func TestCreateDeviation_DuplicateHostFileKeyRejected(t *testing.T) {
	ctx := context.Background()
	repo := &memDeviationRepoForService{}
	svc := NewDefaultDeviationService(repo)

	req := CreateDeviationRequest{
		Hostname:      "host001.example.com",
		FileType:      models.FileTypePasswd,
		EntryLine:     "admin:x:0:0:admin:/home/admin:/bin/bash",
		Justification: "service account",
		ApprovedBy:    "admin",
	}

	if _, err := svc.Create(ctx, req); err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}

	_, err := svc.Create(ctx, req)
	if err == nil {
		t.Fatalf("expected duplicate deviation error")
	}
	if err != repository.ErrDuplicateDeviation {
		t.Fatalf("expected ErrDuplicateDeviation, got %v", err)
	}
}

func TestUpdateDeviation_DuplicateHostFileKeyRejected(t *testing.T) {
	ctx := context.Background()
	repo := &memDeviationRepoForService{}
	svc := NewDefaultDeviationService(repo)

	first, err := svc.Create(ctx, CreateDeviationRequest{
		Hostname:      "host001.example.com",
		FileType:      models.FileTypePasswd,
		EntryLine:     "admin:x:0:0:admin:/home/admin:/bin/bash",
		Justification: "service account",
		ApprovedBy:    "admin",
	})
	if err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}

	second, err := svc.Create(ctx, CreateDeviationRequest{
		Hostname:      "host001.example.com",
		FileType:      models.FileTypePasswd,
		EntryLine:     "backup:x:0:0:backup:/home/backup:/bin/bash",
		Justification: "backup account",
		ApprovedBy:    "admin",
	})
	if err != nil {
		t.Fatalf("second create should succeed: %v", err)
	}

	// Attempt to update the second deviation to use the same key as the first.
	_, err = svc.Update(ctx, second.ID, UpdateDeviationRequest{
		Hostname:      first.Hostname,
		FileType:      first.FileType,
		EntryLine:     "admin:x:0:0:admin:/home/admin:/bin/bash",
		Justification: first.Justification,
		ApprovedBy:    first.ApprovedBy,
		IsActive:      first.IsActive,
	})
	if err == nil {
		t.Fatalf("expected duplicate deviation error on update")
	}
	if err != repository.ErrDuplicateDeviation {
		t.Fatalf("expected ErrDuplicateDeviation, got %v", err)
	}
}

func TestCreateDeviation_DifferentKeysAllowedForSameHost(t *testing.T) {
	ctx := context.Background()
	repo := &memDeviationRepoForService{}
	svc := NewDefaultDeviationService(repo)

	keys := []string{"admin", "backup", "service"}
	for _, key := range keys {
		_, err := svc.Create(ctx, CreateDeviationRequest{
			Hostname:      "host001.example.com",
			FileType:      models.FileTypePasswd,
			EntryLine:     key + ":x:0:0:" + key + ":/home/" + key + ":/bin/bash",
			Justification: "account " + key,
			ApprovedBy:    "admin",
		})
		if err != nil {
			t.Fatalf("create %s should succeed: %v", key, err)
		}
	}

	if len(repo.deviations) != 3 {
		t.Fatalf("expected 3 deviations, got %d", len(repo.deviations))
	}
}

func TestCreateDeviation_ParseEntryLine(t *testing.T) {
	ctx := context.Background()
	repo := &memDeviationRepoForService{}
	svc := NewDefaultDeviationService(repo)

	deviation, err := svc.Create(ctx, CreateDeviationRequest{
		Hostname:      "host001.example.com",
		FileType:      models.FileTypePasswd,
		EntryLine:     "root:x:0:0:root:/root:/bin/bash",
		Justification: "root account",
		ApprovedBy:    "admin",
	})
	if err != nil {
		t.Fatalf("create should succeed: %v", err)
	}

	if deviation.EntryKey != "root" {
		t.Fatalf("expected entry key root, got %s", deviation.EntryKey)
	}
	if deviation.EntryValue == nil || *deviation.EntryValue != "x:0:0:root:/root:/bin/bash" {
		v := "<nil>"
		if deviation.EntryValue != nil {
			v = *deviation.EntryValue
		}
		t.Fatalf("expected entry value x:0:0:root:/root:/bin/bash, got %s", v)
	}
}

func TestCreateDeviation_InvalidEntryLineRejected(t *testing.T) {
	ctx := context.Background()
	repo := &memDeviationRepoForService{}
	svc := NewDefaultDeviationService(repo)

	_, err := svc.Create(ctx, CreateDeviationRequest{
		Hostname:      "host001.example.com",
		FileType:      models.FileTypePasswd,
		EntryLine:     "invalid-line-without-colon",
		Justification: "bad entry",
		ApprovedBy:    "admin",
	})
	if err == nil {
		t.Fatalf("expected error for invalid entry line")
	}
}

func TestCreateDeviation_PendingAndInactive(t *testing.T) {
	ctx := context.Background()
	repo := &memDeviationRepoForService{}
	svc := NewDefaultDeviationService(repo)

	deviation, err := svc.Create(ctx, CreateDeviationRequest{
		Hostname:      "host001.example.com",
		FileType:      models.FileTypePasswd,
		EntryLine:     "admin:x:0:0:admin:/home/admin:/bin/bash",
		Justification: "service account",
		CreatedBy:     "alice",
	})
	if err != nil {
		t.Fatalf("create should succeed: %v", err)
	}

	if deviation.IsActive {
		t.Fatalf("new deviation should be inactive until approved")
	}
	if deviation.ApprovalStatus != "pending" {
		t.Fatalf("expected approval status pending, got %q", deviation.ApprovalStatus)
	}
	if deviation.ApprovedBy != "" {
		t.Fatalf("expected empty approved_by before approval, got %q", deviation.ApprovedBy)
	}
	if !deviation.ApprovedAt.IsZero() {
		t.Fatalf("expected zero approved_at before approval, got %v", deviation.ApprovedAt)
	}
	if deviation.CreatedBy != "alice" {
		t.Fatalf("expected created_by alice, got %q", deviation.CreatedBy)
	}

	stored := repo.deviations[0]
	if stored.IsActive || stored.ApprovalStatus != "pending" || stored.ApprovedBy != "" {
		t.Fatalf("stored deviation should be pending and inactive, got %+v", stored)
	}
}

func TestApproveDeviation_Success(t *testing.T) {
	ctx := context.Background()
	repo := &memDeviationRepoForService{}
	svc := NewDefaultDeviationService(repo)

	deviation, err := svc.Create(ctx, CreateDeviationRequest{
		Hostname:      "host001.example.com",
		FileType:      models.FileTypePasswd,
		EntryLine:     "admin:x:0:0:admin:/home/admin:/bin/bash",
		Justification: "service account",
		CreatedBy:     "alice",
	})
	if err != nil {
		t.Fatalf("create should succeed: %v", err)
	}

	if err := svc.Approve(ctx, deviation.ID, "bob"); err != nil {
		t.Fatalf("approve by different user should succeed: %v", err)
	}

	stored := repo.deviations[0]
	if !stored.IsActive {
		t.Fatalf("deviation should be active after approval")
	}
	if stored.ApprovalStatus != "approved" {
		t.Fatalf("expected approval status approved, got %q", stored.ApprovalStatus)
	}
	if stored.ApprovedBy != "bob" {
		t.Fatalf("expected approved_by bob, got %q", stored.ApprovedBy)
	}
	if stored.ApprovedAt.IsZero() {
		t.Fatalf("expected approved_at to be set after approval")
	}
}

func TestApproveDeviation_SelfApprovalRejected(t *testing.T) {
	ctx := context.Background()
	repo := &memDeviationRepoForService{}
	svc := NewDefaultDeviationService(repo)

	deviation, err := svc.Create(ctx, CreateDeviationRequest{
		Hostname:      "host001.example.com",
		FileType:      models.FileTypePasswd,
		EntryLine:     "admin:x:0:0:admin:/home/admin:/bin/bash",
		Justification: "service account",
		CreatedBy:     "alice",
	})
	if err != nil {
		t.Fatalf("create should succeed: %v", err)
	}

	err = svc.Approve(ctx, deviation.ID, "alice")
	if err == nil {
		t.Fatalf("expected self-approval to be rejected")
	}
	if err != ErrSelfApproval {
		t.Fatalf("expected ErrSelfApproval, got %v", err)
	}

	// case-insensitive match must also be rejected
	err = svc.Approve(ctx, deviation.ID, "ALICE")
	if err != ErrSelfApproval {
		t.Fatalf("expected ErrSelfApproval for case-insensitive match, got %v", err)
	}

	stored := repo.deviations[0]
	if stored.IsActive || stored.ApprovalStatus != "pending" || stored.ApprovedBy != "" {
		t.Fatalf("rejected approval must not modify the deviation, got %+v", stored)
	}
}

func TestApproveDeviation_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &memDeviationRepoForService{}
	svc := NewDefaultDeviationService(repo)

	err := svc.Approve(ctx, uuid.New(), "bob")
	if err == nil {
		t.Fatalf("expected error for unknown deviation")
	}
	if !errors.Is(err, repository.ErrDeviationNotFound) {
		t.Fatalf("expected ErrDeviationNotFound, got %v", err)
	}
}

func TestApproveDeviation_AlreadyApprovedIsNoOp(t *testing.T) {
	ctx := context.Background()
	repo := &memDeviationRepoForService{}
	svc := NewDefaultDeviationService(repo)

	deviation, err := svc.Create(ctx, CreateDeviationRequest{
		Hostname:      "host001.example.com",
		FileType:      models.FileTypePasswd,
		EntryLine:     "admin:x:0:0:admin:/home/admin:/bin/bash",
		Justification: "service account",
		CreatedBy:     "alice",
	})
	if err != nil {
		t.Fatalf("create should succeed: %v", err)
	}
	if err := svc.Approve(ctx, deviation.ID, "bob"); err != nil {
		t.Fatalf("approve should succeed: %v", err)
	}

	approvedAt := repo.deviations[0].ApprovedAt
	if err := svc.Approve(ctx, deviation.ID, "carol"); err != nil {
		t.Fatalf("re-approving an approved deviation should be a no-op: %v", err)
	}
	if repo.deviations[0].ApprovedBy != "bob" {
		t.Fatalf("re-approval must not change the original approver, got %q", repo.deviations[0].ApprovedBy)
	}
	if !repo.deviations[0].ApprovedAt.Equal(approvedAt) {
		t.Fatalf("re-approval must not change the original approval time")
	}
}

func TestUpdateDeviation_PendingCannotBeActivated(t *testing.T) {
	ctx := context.Background()
	repo := &memDeviationRepoForService{}
	svc := NewDefaultDeviationService(repo)

	deviation, err := svc.Create(ctx, CreateDeviationRequest{
		Hostname:      "host001.example.com",
		FileType:      models.FileTypePasswd,
		EntryLine:     "admin:x:0:0:admin:/home/admin:/bin/bash",
		Justification: "service account",
		CreatedBy:     "alice",
	})
	if err != nil {
		t.Fatalf("create should succeed: %v", err)
	}

	_, err = svc.Update(ctx, deviation.ID, UpdateDeviationRequest{
		Hostname:      deviation.Hostname,
		FileType:      deviation.FileType,
		EntryLine:     "admin:x:0:0:admin:/home/admin:/bin/bash",
		Justification: "service account",
		ApprovedBy:    "alice",
		IsActive:      true,
	})
	if err == nil {
		t.Fatalf("expected error when activating a pending deviation via update")
	}
	if err != ErrPendingApproval {
		t.Fatalf("expected ErrPendingApproval, got %v", err)
	}

	stored := repo.deviations[0]
	if stored.IsActive || stored.ApprovalStatus != "pending" {
		t.Fatalf("pending deviation must remain inactive, got %+v", stored)
	}
}

func TestListPendingDeviations_OnlyPending(t *testing.T) {
	ctx := context.Background()
	repo := &memDeviationRepoForService{}
	svc := NewDefaultDeviationService(repo)

	for i, key := range []string{"admin", "backup"} {
		_, err := svc.Create(ctx, CreateDeviationRequest{
			Hostname:      "host001.example.com",
			FileType:      models.FileTypePasswd,
			EntryLine:     key + ":x:0:0:" + key + ":/home/" + key + ":/bin/bash",
			Justification: "account " + key,
			CreatedBy:     "alice",
		})
		if err != nil {
			t.Fatalf("create %d should succeed: %v", i, err)
		}
	}

	pending, err := svc.ListPending(ctx)
	if err != nil {
		t.Fatalf("list pending should succeed: %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending deviations, got %d", len(pending))
	}

	if err := svc.Approve(ctx, repo.deviations[0].ID, "bob"); err != nil {
		t.Fatalf("approve should succeed: %v", err)
	}
	pending, err = svc.ListPending(ctx)
	if err != nil {
		t.Fatalf("list pending should succeed: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending deviation after approval, got %d", len(pending))
	}
}

func TestRejectDeviation_Pending(t *testing.T) {
	ctx := context.Background()
	repo := &memDeviationRepoForService{}
	svc := NewDefaultDeviationService(repo)

	deviation, err := svc.Create(ctx, CreateDeviationRequest{
		Hostname:      "host001.example.com",
		FileType:      models.FileTypePasswd,
		EntryLine:     "admin:x:0:0:admin:/home/admin:/bin/bash",
		Justification: "service account",
		CreatedBy:     "alice",
	})
	if err != nil {
		t.Fatalf("create should succeed: %v", err)
	}

	if err := svc.Reject(ctx, deviation.ID); err != nil {
		t.Fatalf("reject should succeed: %v", err)
	}
	stored := repo.deviations[0]
	if stored.IsActive || stored.ApprovalStatus != "rejected" {
		t.Fatalf("expected rejected+inactive, got %+v", stored)
	}
}

func TestRejectDeviation_CreatorMayRejectOwn(t *testing.T) {
	ctx := context.Background()
	repo := &memDeviationRepoForService{}
	svc := NewDefaultDeviationService(repo)

	deviation, err := svc.Create(ctx, CreateDeviationRequest{
		Hostname:      "host001.example.com",
		FileType:      models.FileTypePasswd,
		EntryLine:     "admin:x:0:0:admin:/home/admin:/bin/bash",
		Justification: "service account",
		CreatedBy:     "alice",
	})
	if err != nil {
		t.Fatalf("create should succeed: %v", err)
	}

	// Rejection is a withdrawal, not an activation — the creator may do it.
	if err := svc.Reject(ctx, deviation.ID); err != nil {
		t.Fatalf("creator should be able to reject their own pending deviation: %v", err)
	}
}

func TestRejectDeviation_ApprovedNotFound(t *testing.T) {
	ctx := context.Background()
	repo := &memDeviationRepoForService{}
	svc := NewDefaultDeviationService(repo)

	deviation, err := svc.Create(ctx, CreateDeviationRequest{
		Hostname:      "host001.example.com",
		FileType:      models.FileTypePasswd,
		EntryLine:     "admin:x:0:0:admin:/home/admin:/bin/bash",
		Justification: "service account",
		CreatedBy:     "alice",
	})
	if err != nil {
		t.Fatalf("create should succeed: %v", err)
	}
	if err := svc.Approve(ctx, deviation.ID, "bob"); err != nil {
		t.Fatalf("approve should succeed: %v", err)
	}

	if err := svc.Reject(ctx, deviation.ID); !errors.Is(err, repository.ErrDeviationNotFound) {
		t.Fatalf("expected ErrDeviationNotFound rejecting an approved deviation, got %v", err)
	}
}

func TestUpdateDeviation_RejectedCannotBeActivated(t *testing.T) {
	ctx := context.Background()
	repo := &memDeviationRepoForService{}
	svc := NewDefaultDeviationService(repo)

	deviation, err := svc.Create(ctx, CreateDeviationRequest{
		Hostname:      "host001.example.com",
		FileType:      models.FileTypePasswd,
		EntryLine:     "admin:x:0:0:admin:/home/admin:/bin/bash",
		Justification: "service account",
		CreatedBy:     "alice",
	})
	if err != nil {
		t.Fatalf("create should succeed: %v", err)
	}
	if err := svc.Reject(ctx, deviation.ID); err != nil {
		t.Fatalf("reject should succeed: %v", err)
	}

	_, err = svc.Update(ctx, deviation.ID, UpdateDeviationRequest{
		Hostname:      deviation.Hostname,
		FileType:      deviation.FileType,
		EntryLine:     "admin:x:0:0:admin:/home/admin:/bin/bash",
		Justification: "service account",
		IsActive:      true,
	})
	if err != ErrPendingApproval {
		t.Fatalf("expected ErrPendingApproval activating a rejected deviation, got %v", err)
	}
}
