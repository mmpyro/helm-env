package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// seedPruneVersion creates $HELMENV_ROOT/versions/<v>/helm with the given
// binary mtime.
func seedPruneVersion(t *testing.T, root, v string, mtime time.Time) {
	t.Helper()
	vdir := filepath.Join(root, "versions", v)
	if err := os.MkdirAll(vdir, 0o755); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(vdir, "helm")
	if err := os.WriteFile(binary, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(binary, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func TestParseDurationWithDays(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
		fail bool
	}{
		{"30m", 30 * time.Minute, false},
		{"24h", 24 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"1d12h", 36 * time.Hour, false},
		{"", 0, true},
		{"abc", 0, true},
	}
	for _, tt := range tests {
		got, err := parseDurationWithDays(tt.in)
		if tt.fail {
			if err == nil {
				t.Errorf("parseDurationWithDays(%q) expected error, got %v", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseDurationWithDays(%q) error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseDurationWithDays(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestPrune_RequiresFlags(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	if err := os.MkdirAll(filepath.Join(root, "versions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Prune(0, "", false, true); err == nil {
		t.Fatal("expected error when neither --keep nor --older-than supplied")
	}
}

func TestPrune_NotInitialized(t *testing.T) {
	t.Setenv("HELMENV_ROOT", "")
	if err := Prune(1, "", false, true); err == nil {
		t.Fatal("expected error when not initialized")
	}
}

func TestPrune_Keep(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_VERSION", "3.14.0")
	now := time.Now()
	for _, v := range []string{"3.12.0", "3.13.0", "3.14.0", "3.15.0"} {
		seedPruneVersion(t, root, v, now)
	}

	out := captureStdout(t, func() {
		if err := Prune(2, "", false, true); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// 3.14.0 and 3.15.0 are the newest 2 → keep.  3.13.0 and 3.12.0 should be pruned.
	for _, v := range []string{"3.12.0", "3.13.0"} {
		if _, err := os.Stat(filepath.Join(root, "versions", v)); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be pruned", v)
		}
		if !strings.Contains(out, "pruning "+v) {
			t.Errorf("expected output to mention pruning %s, got %q", v, out)
		}
	}
	for _, v := range []string{"3.14.0", "3.15.0"} {
		if _, err := os.Stat(filepath.Join(root, "versions", v)); err != nil {
			t.Fatalf("expected %s to be kept, stat err: %v", v, err)
		}
	}
}

func TestPrune_OlderThan(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_VERSION", "")
	// 3.12.0 is old; 3.13.0 is recent.
	old := time.Now().Add(-48 * time.Hour)
	recent := time.Now().Add(-10 * time.Minute)
	seedPruneVersion(t, root, "3.12.0", old)
	seedPruneVersion(t, root, "3.13.0", recent)

	if err := Prune(0, "24h", false, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "versions", "3.12.0")); !os.IsNotExist(err) {
		t.Fatal("expected 3.12.0 to be pruned by --older-than 24h")
	}
	if _, err := os.Stat(filepath.Join(root, "versions", "3.13.0")); err != nil {
		t.Fatal("expected 3.13.0 to be kept")
	}
}

func TestPrune_DryRun(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_VERSION", "")
	old := time.Now().Add(-48 * time.Hour)
	seedPruneVersion(t, root, "3.12.0", old)

	out := captureStdout(t, func() {
		if err := Prune(0, "24h", true, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "[dry-run] would prune 3.12.0") {
		t.Fatalf("expected dry-run output, got %q", out)
	}
	if _, err := os.Stat(filepath.Join(root, "versions", "3.12.0")); err != nil {
		t.Fatal("dry-run should not have removed anything")
	}
}

func TestPrune_CombinedKeepAndOlderThan(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_VERSION", "")
	old := time.Now().Add(-72 * time.Hour)
	recent := time.Now().Add(-10 * time.Minute)
	// 3.12.0 (old), 3.13.0 (recent), 3.14.0 (recent)
	seedPruneVersion(t, root, "3.12.0", old)
	seedPruneVersion(t, root, "3.13.0", recent)
	seedPruneVersion(t, root, "3.14.0", recent)

	// --keep 2 would keep 3.14.0 and 3.13.0 (newest 2). --older-than 24h
	// would keep 3.13.0 and 3.14.0 (recent).  BOTH rules keep the two
	// recents → 3.12.0 should be pruned.
	if err := Prune(2, "24h", false, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "versions", "3.12.0")); !os.IsNotExist(err) {
		t.Fatal("expected 3.12.0 to be pruned")
	}
	if _, err := os.Stat(filepath.Join(root, "versions", "3.13.0")); err != nil {
		t.Fatal("expected 3.13.0 to be kept")
	}
	if _, err := os.Stat(filepath.Join(root, "versions", "3.14.0")); err != nil {
		t.Fatal("expected 3.14.0 to be kept")
	}
}

func TestPrune_ProtectsActiveVersion(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_VERSION", "3.12.0")
	old := time.Now().Add(-48 * time.Hour)
	seedPruneVersion(t, root, "3.12.0", old)
	seedPruneVersion(t, root, "3.13.0", old)

	stdout, stderr := captureBoth(t, func() {
		if err := Prune(0, "24h", false, true); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	// 3.12.0 is active → protected despite being old.
	if _, err := os.Stat(filepath.Join(root, "versions", "3.12.0")); err != nil {
		t.Fatal("expected active 3.12.0 to be preserved")
	}
	if !strings.Contains(stderr, "refusing to prune active version 3.12.0") {
		t.Fatalf("expected warning on stderr, got %q", stderr)
	}
	// 3.13.0 should still be pruned.
	if _, err := os.Stat(filepath.Join(root, "versions", "3.13.0")); !os.IsNotExist(err) {
		t.Fatal("expected 3.13.0 to be pruned")
	}
	_ = stdout
}

func TestPrune_DryRunByDefault(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_VERSION", "")
	old := time.Now().Add(-48 * time.Hour)
	seedPruneVersion(t, root, "3.12.0", old)

	out := captureStdout(t, func() {
		if err := Prune(0, "24h", false, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "[dry-run] would prune 3.12.0") {
		t.Fatalf("expected dry-run output without --yes, got %q", out)
	}
	if !strings.Contains(out, "--yes") {
		t.Errorf("expected a hint to re-run with --yes, got %q", out)
	}
	if _, err := os.Stat(filepath.Join(root, "versions", "3.12.0")); err != nil {
		t.Fatal("prune without --yes must not remove anything")
	}
}

func TestPrune_DryRunWinsOverYes(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_VERSION", "")
	old := time.Now().Add(-48 * time.Hour)
	seedPruneVersion(t, root, "3.12.0", old)

	out := captureStdout(t, func() {
		if err := Prune(0, "24h", true, true); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "[dry-run] would prune 3.12.0") {
		t.Fatalf("expected dry-run output, got %q", out)
	}
	if _, err := os.Stat(filepath.Join(root, "versions", "3.12.0")); err != nil {
		t.Fatal("--dry-run must win over --yes")
	}
}

func TestPrune_InvalidOlderThan(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	if err := os.MkdirAll(filepath.Join(root, "versions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Prune(0, "bogus", false, true); err == nil {
		t.Fatal("expected error for --older-than bogus")
	}
}

func TestPruneHelp_DocumentsNewFlags(t *testing.T) {
	out := captureStdout(t, PruneHelp)
	for _, want := range []string{"--keep-last N", "--keep N", "Deprecated", "--yes", "DEFAULT"} {
		if !strings.Contains(out, want) {
			t.Errorf("prune help missing %q; got:\n%s", want, out)
		}
	}
}
