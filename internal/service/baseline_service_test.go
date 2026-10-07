package service

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/google/uuid"

	"ulas-service/internal/repository"
	"ulas-service/models"
)

// memBaselineRepoForService is an in-memory BaselineRepository for service-level tests.
type memBaselineRepoForService struct {
	creators     map[string]string
	statuses     map[string]string // baselineScopeKey → approval status; "" means pending
	approveCalls []baselineApproveCall
	rejectCalls  []string
}

type baselineApproveCall struct {
	osType   models.OSType
	fileType models.FileType
	version  int
	approver string
}

func baselineScopeKey(osType models.OSType, fileType models.FileType, version int) string {
	return string(osType) + "/" + string(fileType) + "/" + strconv.Itoa(version)
}

func (r *memBaselineRepoForService) Create(ctx context.Context, baseline *models.MasterBaseline) error {
	return nil
}
func (r *memBaselineRepoForService) GetByID(ctx context.Context, id uuid.UUID) (*models.MasterBaseline, error) {
	return nil, repository.ErrBaselineNotFound
}
func (r *memBaselineRepoForService) List(ctx context.Context, filters repository.BaselineFilters) ([]models.MasterBaseline, error) {
	return nil, nil
}
func (r *memBaselineRepoForService) Update(ctx context.Context, baseline *models.MasterBaseline) error {
	return nil
}
func (r *memBaselineRepoForService) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (r *memBaselineRepoForService) CreateVersion(ctx context.Context, version *models.MasterBaselineVersion) error {
	return nil
}
func (r *memBaselineRepoForService) CreateVersionedEntries(ctx context.Context, osType models.OSType, fileType models.FileType, version int, entries []repository.BaselineEntryInput, createdBy, description string, active bool) error {
	return nil
}
func (r *memBaselineRepoForService) SetActiveVersion(ctx context.Context, osType models.OSType, fileType models.FileType, version int) (int64, error) {
	if r.statuses[baselineScopeKey(osType, fileType, version)] == "approved" {
		return 1, nil
	}
	return 0, nil
}
func (r *memBaselineRepoForService) DeactivateScope(ctx context.Context, osType models.OSType, fileType models.FileType, version int) error {
	return nil
}
func (r *memBaselineRepoForService) ListVersions(ctx context.Context) ([]repository.BaselineVersionSummary, error) {
	return nil, nil
}
func (r *memBaselineRepoForService) ListVersionsPaginated(ctx context.Context, page, limit int) ([]repository.BaselineVersionSummary, int, error) {
	return nil, 0, nil
}
func (r *memBaselineRepoForService) ListPendingVersions(ctx context.Context) ([]repository.BaselineVersionSummary, error) {
	return nil, nil
}
func (r *memBaselineRepoForService) ApproveVersion(ctx context.Context, osType models.OSType, fileType models.FileType, version int, approver string) (int64, error) {
	r.approveCalls = append(r.approveCalls, baselineApproveCall{osType: osType, fileType: fileType, version: version, approver: approver})
	if r.statuses == nil {
		r.statuses = map[string]string{}
	}
	r.statuses[baselineScopeKey(osType, fileType, version)] = "approved"
	return 1, nil
}
func (r *memBaselineRepoForService) RejectVersion(ctx context.Context, osType models.OSType, fileType models.FileType, version int) (int64, error) {
	r.rejectCalls = append(r.rejectCalls, baselineScopeKey(osType, fileType, version))
	if r.statuses[baselineScopeKey(osType, fileType, version)] == "" {
		if r.statuses == nil {
			r.statuses = map[string]string{}
		}
		r.statuses[baselineScopeKey(osType, fileType, version)] = "rejected"
		return 1, nil
	}
	return 0, nil
}
func (r *memBaselineRepoForService) GetVersionCreator(ctx context.Context, osType models.OSType, fileType models.FileType, version int) (string, error) {
	creator, ok := r.creators[baselineScopeKey(osType, fileType, version)]
	if !ok {
		return "", repository.ErrBaselineVersionNotFound
	}
	return creator, nil
}

func TestCreateBaseline_PendingAndInactive(t *testing.T) {
	ctx := context.Background()
	repo := &memBaselineRepoForService{}
	svc := NewDefaultBaselineService(repo, nil)

	baseline, err := svc.Create(ctx, CreateBaselineRequest{
		OSType:      models.OSTypeLinux,
		FileType:    models.FileTypePasswd,
		EntryKey:    "root",
		EntryValue:  "x:0:0:root:/root:/bin/bash",
		Version:     7,
		CreatedBy:   "alice",
		Description: "initial",
	})
	if err != nil {
		t.Fatalf("create should succeed: %v", err)
	}
	if baseline.IsActive {
		t.Fatalf("new baseline should be inactive until approved")
	}
	if baseline.ApprovalStatus != "pending" {
		t.Fatalf("expected approval status pending, got %q", baseline.ApprovalStatus)
	}
	if baseline.CreatedBy != "alice" {
		t.Fatalf("expected created_by alice, got %q", baseline.CreatedBy)
	}
}

func TestApproveBaselineVersion_Success(t *testing.T) {
	ctx := context.Background()
	repo := &memBaselineRepoForService{
		creators: map[string]string{baselineScopeKey(models.OSTypeLinux, models.FileTypePasswd, 7): "alice"},
	}
	svc := NewDefaultBaselineService(repo, nil)

	err := svc.ApproveVersion(ctx, models.OSTypeLinux, models.FileTypePasswd, 7, "bob")
	if err != nil {
		t.Fatalf("approve by different user should succeed: %v", err)
	}

	if len(repo.approveCalls) != 1 {
		t.Fatalf("expected 1 approve call, got %d", len(repo.approveCalls))
	}
	call := repo.approveCalls[0]
	if call.approver != "bob" || call.osType != models.OSTypeLinux || call.fileType != models.FileTypePasswd || call.version != 7 {
		t.Fatalf("unexpected approve call: %+v", call)
	}
}

func TestApproveBaselineVersion_SelfApprovalRejected(t *testing.T) {
	ctx := context.Background()
	repo := &memBaselineRepoForService{
		creators: map[string]string{baselineScopeKey(models.OSTypeLinux, models.FileTypePasswd, 7): "alice"},
	}
	svc := NewDefaultBaselineService(repo, nil)

	err := svc.ApproveVersion(ctx, models.OSTypeLinux, models.FileTypePasswd, 7, "alice")
	if err == nil {
		t.Fatalf("expected self-approval to be rejected")
	}
	if err != ErrSelfApproval {
		t.Fatalf("expected ErrSelfApproval, got %v", err)
	}

	// case-insensitive match must also be rejected
	err = svc.ApproveVersion(ctx, models.OSTypeLinux, models.FileTypePasswd, 7, "ALICE")
	if err != ErrSelfApproval {
		t.Fatalf("expected ErrSelfApproval for case-insensitive match, got %v", err)
	}

	if len(repo.approveCalls) != 0 {
		t.Fatalf("rejected approval must not call the repository, got %+v", repo.approveCalls)
	}
}

func TestApproveBaselineVersion_UnknownScope(t *testing.T) {
	ctx := context.Background()
	repo := &memBaselineRepoForService{}
	svc := NewDefaultBaselineService(repo, nil)

	err := svc.ApproveVersion(ctx, models.OSTypeLinux, models.FileTypePasswd, 9, "bob")
	if err == nil {
		t.Fatalf("expected error for unknown scope")
	}
	if !errors.Is(err, repository.ErrBaselineVersionNotFound) {
		t.Fatalf("expected ErrBaselineVersionNotFound, got %v", err)
	}
}

func TestApproveBaselineVersion_ZeroRowsTreatedAsNotFound(t *testing.T) {
	ctx := context.Background()
	repo := &memBaselineRepoForService{
		creators: map[string]string{baselineScopeKey(models.OSTypeLinux, models.FileTypePasswd, 7): "alice"},
	}
	svc := NewDefaultBaselineService(approveZeroRowsRepo{repo}, nil)

	err := svc.ApproveVersion(ctx, models.OSTypeLinux, models.FileTypePasswd, 7, "bob")
	if err == nil {
		t.Fatalf("expected error when the update affects no rows")
	}
	if !errors.Is(err, repository.ErrBaselineVersionNotFound) {
		t.Fatalf("expected ErrBaselineVersionNotFound, got %v", err)
	}
}

// approveZeroRowsRepo wraps the mem fake to simulate an ApproveVersion that matches no rows.
type approveZeroRowsRepo struct {
	*memBaselineRepoForService
}

func (r approveZeroRowsRepo) ApproveVersion(ctx context.Context, osType models.OSType, fileType models.FileType, version int, approver string) (int64, error) {
	return 0, nil
}

func TestListPendingBaselineVersions_Passthrough(t *testing.T) {
	ctx := context.Background()
	repo := &memBaselineRepoForService{}
	svc := NewDefaultBaselineService(repo, nil)

	versions, err := svc.ListPendingVersions(ctx)
	if err != nil {
		t.Fatalf("list pending versions should succeed: %v", err)
	}
	if versions != nil {
		t.Fatalf("expected nil versions from empty fake, got %v", versions)
	}
}

func TestParseMasterFileContent_Passwd(t *testing.T) {
	content := "akmods:x:966:965:User is used by akmods to build akmod packages:/var/cache/akmods/:/sbin/nologin\nmpd:x:964:964:Music Player Daemon:/var/lib/mpd:/sbin/nologin"
	entries, err := parseMasterFileContent(models.FileTypePasswd, content, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].EntryKey != "akmods" {
		t.Errorf("expected key akmods, got %s", entries[0].EntryKey)
	}
	if entries[0].EntryValue != "x:966:965:User is used by akmods to build akmod packages:/var/cache/akmods/:/sbin/nologin" {
		t.Errorf("unexpected value: %s", entries[0].EntryValue)
	}
}

func TestParseMasterFileContent_Group(t *testing.T) {
	content := "akmods:x:965:\nmpd:x:964:sam3,sam1,sam2\nwheel:x:10:alice,bob"
	entries, err := parseMasterFileContent(models.FileTypeGroup, content, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	want := map[string]string{
		"akmods": "x:965",
		"mpd":    "x:964:sam1,sam2,sam3",
		"wheel":  "x:10:alice,bob",
	}
	got := make(map[string]string)
	for _, e := range entries {
		got[e.EntryKey] = e.EntryValue
	}

	for key, value := range want {
		if got[key] != value {
			t.Errorf("key %s: expected %q, got %q", key, value, got[key])
		}
	}
}

func TestNormalizeGroupSnapshotValue(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"x:965:", "x:965"},
		{"x:964:sam3,sam1,sam2", "x:964:sam1,sam2,sam3"},
		{"x:10:alice,bob", "x:10:alice,bob"},
	}
	for _, tc := range tests {
		got := normalizeGroupSnapshotValue(tc.input)
		if got != tc.want {
			t.Errorf("normalizeGroupSnapshotValue(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestParseSnapshotContent_GroupMembersNormalized(t *testing.T) {
	content := "mpd:x:964:sam3,sam1,sam2"
	got := parseSnapshotContent(models.FileTypeGroup, content)
	if got["mpd"] != "x:964:sam1,sam2,sam3" {
		t.Errorf("unexpected value: %q", got["mpd"])
	}
}

var _ = repository.BaselineEntryInput{}

func TestParsePrivilegeList(t *testing.T) {
	set := parsePrivilegeList("root\n\nwheel, sudo\n# comment\n  daemon  \n")
	for _, key := range []string{"root", "wheel", "sudo", "daemon"} {
		if !set[key] {
			t.Errorf("expected %q in privilege set", key)
		}
	}
	if set["comment"] || set[""] {
		t.Errorf("unexpected keys in privilege set: %v", set)
	}
	if len(set) != 4 {
		t.Errorf("expected 4 keys, got %d: %v", len(set), set)
	}
}

func TestParseMasterFileContent_PrivilegeList(t *testing.T) {
	content := "root:x:0:0:root:/root:/bin/bash\nadmin:x:1000:1000:admin:/home/admin:/bin/bash"
	privileged := map[string]bool{"root": true}

	entries, err := parseMasterFileContent(models.FileTypePasswd, content, privileged)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if !entries[0].CheckIDs {
		t.Errorf("expected root to have CheckIDs set")
	}
	if entries[1].CheckIDs {
		t.Errorf("expected admin to have CheckIDs unset")
	}
}

func TestRejectBaselineVersion_PendingScope(t *testing.T) {
	ctx := context.Background()
	repo := &memBaselineRepoForService{
		creators: map[string]string{baselineScopeKey(models.OSTypeLinux, models.FileTypeGroup, 7): "alice"},
	}
	svc := NewDefaultBaselineService(repo, nil)

	if err := svc.RejectVersion(ctx, models.OSTypeLinux, models.FileTypeGroup, 7); err != nil {
		t.Fatalf("reject pending scope should succeed: %v", err)
	}
	if len(repo.rejectCalls) != 1 {
		t.Fatalf("expected 1 reject call, got %d", len(repo.rejectCalls))
	}
}

func TestRejectBaselineVersion_CreatorMayRejectOwn(t *testing.T) {
	ctx := context.Background()
	repo := &memBaselineRepoForService{
		creators: map[string]string{baselineScopeKey(models.OSTypeLinux, models.FileTypeGroup, 7): "alice"},
	}
	svc := NewDefaultBaselineService(repo, nil)

	// Rejection is a withdrawal, not an activation — the creator may do it.
	if err := svc.RejectVersion(ctx, models.OSTypeLinux, models.FileTypeGroup, 7); err != nil {
		t.Fatalf("creator should be able to reject their own pending scope: %v", err)
	}
}

func TestRejectBaselineVersion_AlreadyApprovedRejected(t *testing.T) {
	ctx := context.Background()
	repo := &memBaselineRepoForService{
		creators: map[string]string{baselineScopeKey(models.OSTypeLinux, models.FileTypeGroup, 7): "alice"},
		statuses: map[string]string{baselineScopeKey(models.OSTypeLinux, models.FileTypeGroup, 7): "approved"},
	}
	svc := NewDefaultBaselineService(repo, nil)

	if err := svc.RejectVersion(ctx, models.OSTypeLinux, models.FileTypeGroup, 7); err != ErrVersionNotApproved {
		t.Fatalf("expected ErrVersionNotApproved rejecting an approved scope, got %v", err)
	}
}

func TestRejectBaselineVersion_UnknownScope(t *testing.T) {
	ctx := context.Background()
	repo := &memBaselineRepoForService{creators: map[string]string{}}
	svc := NewDefaultBaselineService(repo, nil)

	err := svc.RejectVersion(ctx, models.OSTypeLinux, models.FileTypeGroup, 99)
	if !errors.Is(err, repository.ErrBaselineVersionNotFound) {
		t.Fatalf("expected ErrBaselineVersionNotFound, got %v", err)
	}
}

func TestActivateBaselineVersion_RequiresApproval(t *testing.T) {
	ctx := context.Background()
	repo := &memBaselineRepoForService{creators: map[string]string{}}
	svc := NewDefaultBaselineService(repo, nil)

	// Pending scope: activation must fail so the 4-eyes rule cannot be bypassed.
	if err := svc.ActivateVersion(ctx, models.OSTypeLinux, models.FileTypePasswd, 7); err != ErrVersionNotApproved {
		t.Fatalf("expected ErrVersionNotApproved activating a pending scope, got %v", err)
	}

	// Approved scope: activation succeeds.
	repo.statuses = map[string]string{baselineScopeKey(models.OSTypeLinux, models.FileTypePasswd, 7): "approved"}
	if err := svc.ActivateVersion(ctx, models.OSTypeLinux, models.FileTypePasswd, 7); err != nil {
		t.Fatalf("activating an approved scope should succeed: %v", err)
	}
}
