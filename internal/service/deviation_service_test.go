package service

import (
	"context"
	"testing"

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
			return &r.deviations[i], nil
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
