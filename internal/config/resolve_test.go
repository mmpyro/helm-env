package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// seedVersion creates $HELMENV_ROOT/versions/<version>/helm so
// InstalledVersions sees it.
func seedVersion(t *testing.T, root, version string) {
	t.Helper()
	dir := filepath.Join(root, "versions", version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "helm"), []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestInstalledVersions(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	for _, v := range []string{"3.14.0", "3.15.0"} {
		seedVersion(t, root, v)
	}

	// Also add a stray directory with no binary to ensure it is skipped.
	if err := os.MkdirAll(filepath.Join(root, "versions", "stray"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := InstalledVersions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 installed versions, got %v", got)
	}
}

func TestResolveConcreteVersion_Exact(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_VERSION", "3.14.0")
	seedVersion(t, root, "3.14.0")

	v, err := ResolveConcreteVersion()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "3.14.0" {
		t.Fatalf("got %q, want 3.14.0", v)
	}
}

func TestResolveConcreteVersion_PartialResolvesInstalled(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_VERSION", "3.14")
	seedVersion(t, root, "3.14.0")
	seedVersion(t, root, "3.14.5")
	seedVersion(t, root, "3.15.0")

	v, err := ResolveConcreteVersion()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "3.14.5" {
		t.Fatalf("got %q, want 3.14.5", v)
	}
}

func TestResolveConcreteVersion_AliasLatestInstalled(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_VERSION", "latest-installed")
	seedVersion(t, root, "3.14.0")
	seedVersion(t, root, "3.15.0")

	v, err := ResolveConcreteVersion()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "3.15.0" {
		t.Fatalf("got %q, want 3.15.0", v)
	}
}

func TestResolveConcreteVersion_NotInstalled_Exact(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_VERSION", "3.14.0")
	if err := os.MkdirAll(filepath.Join(root, "versions"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := ResolveConcreteVersion()
	if err == nil {
		t.Fatal("expected error for missing exact version")
	}
	var nie *NotInstalledError
	if !errors.As(err, &nie) {
		t.Fatalf("expected NotInstalledError, got %T: %v", err, err)
	}
	if nie.Resolved != "3.14.0" {
		t.Fatalf("expected Resolved=3.14.0, got %q", nie.Resolved)
	}
	if nie.Source != SourceShell {
		t.Fatalf("expected source shell, got %q", nie.Source)
	}
}

func TestResolveConcreteVersion_NotInstalled_Constraint(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_VERSION", "~3.14")
	if err := os.MkdirAll(filepath.Join(root, "versions"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Install a non-matching version.
	seedVersion(t, root, "3.15.0")

	_, err := ResolveConcreteVersion()
	if err == nil {
		t.Fatal("expected error for missing constraint match")
	}
	var nie *NotInstalledError
	if !errors.As(err, &nie) {
		t.Fatalf("expected NotInstalledError, got %T: %v", err, err)
	}
	if nie.Requested != "~3.14" {
		t.Fatalf("expected Requested=~3.14, got %q", nie.Requested)
	}
	if nie.Resolved != "" {
		t.Fatalf("expected empty Resolved for constraint miss, got %q", nie.Resolved)
	}
}

func TestResolveConcreteVersion_LocalPrecedence(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_VERSION", "")
	seedVersion(t, root, "3.14.0")
	seedVersion(t, root, "3.15.0")

	// Write a .helm-version in a work dir.
	work := t.TempDir()
	if err := os.WriteFile(filepath.Join(work, ".helm-version"), []byte("3.14.0"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Set global too (should be ignored).
	if err := os.WriteFile(filepath.Join(root, "version"), []byte("3.15.0"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	v, err := ResolveConcreteVersion()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "3.14.0" {
		t.Fatalf("got %q, want 3.14.0", v)
	}
}

func TestResolveConcreteVersion_GlobalWithConstraint(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_VERSION", "")
	seedVersion(t, root, "3.14.0")
	seedVersion(t, root, "3.14.5")
	// Global file uses a partial constraint.
	if err := os.WriteFile(filepath.Join(root, "version"), []byte("3.14"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Switch to a dir without .helm-version.
	work := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	v, err := ResolveConcreteVersion()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "3.14.5" {
		t.Fatalf("got %q, want 3.14.5", v)
	}
}

func TestResolveVersionTraced(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	seedVersion(t, root, "3.14.0")

	t.Run("shell wins", func(t *testing.T) {
		t.Setenv("HELMENV_VERSION", "3.14.0")
		v, trace, err := ResolveVersionTraced()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "3.14.0" {
			t.Fatalf("got %q", v)
		}
		if len(trace) < 2 {
			t.Fatalf("expected trace with >=2 steps, got %d", len(trace))
		}
		if trace[0].Source != "shell" || !trace[0].Hit {
			t.Fatalf("expected shell hit, got %+v", trace[0])
		}
	})

	t.Run("local hit after walking", func(t *testing.T) {
		t.Setenv("HELMENV_VERSION", "")
		work := filepath.Join(t.TempDir(), "deep", "nested")
		if err := os.MkdirAll(work, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(work, ".helm-version"), []byte("3.14.0"), 0o644); err != nil {
			t.Fatal(err)
		}
		orig, _ := os.Getwd()
		if err := os.Chdir(work); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chdir(orig) }()

		v, trace, err := ResolveVersionTraced()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "3.14.0" {
			t.Fatalf("got %q", v)
		}
		foundLocalHit := false
		for _, step := range trace {
			if step.Source == "local" && step.Hit {
				foundLocalHit = true
			}
		}
		if !foundLocalHit {
			t.Fatalf("expected local hit in trace, got %+v", trace)
		}
	})

	t.Run("global fallback", func(t *testing.T) {
		t.Setenv("HELMENV_VERSION", "")
		if err := os.WriteFile(filepath.Join(root, "version"), []byte("3.14.0"), 0o644); err != nil {
			t.Fatal(err)
		}
		work := t.TempDir()
		orig, _ := os.Getwd()
		if err := os.Chdir(work); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chdir(orig) }()

		v, trace, err := ResolveVersionTraced()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "3.14.0" {
			t.Fatalf("got %q", v)
		}
		foundGlobalHit := false
		for _, step := range trace {
			if step.Source == "global" && step.Hit {
				foundGlobalHit = true
			}
		}
		if !foundGlobalHit {
			t.Fatalf("expected global hit in trace, got %+v", trace)
		}
	})
}
