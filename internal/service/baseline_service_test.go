package service

import (
	"testing"

	"ulas-service/internal/repository"
	"ulas-service/models"
)

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
