package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOpenCrashLogCreatesPerRunLogAndLatestPointer(t *testing.T) {
	logDir := t.TempDir()
	oldPath := filepath.Join(logDir, "crashes", "forte-20260613-140000-111.log")
	if err := os.MkdirAll(filepath.Dir(oldPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldPath, []byte("old fatal trace\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 6, 13, 15, 51, 12, 0, time.UTC)
	f, path, err := openCrashLog(logDir, now, 2257898)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("new run\n")
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	wantSuffix := filepath.Join("crashes", "forte-20260613-155112-2257898.log")
	if !strings.HasSuffix(path, wantSuffix) {
		t.Fatalf("crash log path = %q, want suffix %q", path, wantSuffix)
	}

	latest, err := os.ReadFile(filepath.Join(logDir, "crash.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(latest), path) {
		t.Fatalf("latest pointer %q does not mention %q", string(latest), path)
	}

	old, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(old) != "old fatal trace\n" {
		t.Fatalf("old crash log was changed to %q", string(old))
	}
}

func TestWriteCrashLogHeaderIncludesRuntimeContext(t *testing.T) {
	now := time.Date(2026, 6, 13, 15, 51, 12, 0, time.UTC)
	var out bytes.Buffer

	if err := writeCrashLogHeader(&out, crashLogHeader{
		StartedAt: now,
		PID:       2257898,
		Exe:       "/nix/store/example/bin/forte",
		Args:      []string{"forte", "--debug"},
	}); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{
		"=== Forte crash capture startup ===",
		"started_at=2026-06-13T15:51:12Z",
		"pid=2257898",
		"exe=/nix/store/example/bin/forte",
		"args=forte --debug",
		"go_version=",
		"go_os=",
		"go_arch=",
		"build_main=github.com/willfish/forte",
		"=== end startup ===",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("header missing %q:\n%s", want, got)
		}
	}
}
