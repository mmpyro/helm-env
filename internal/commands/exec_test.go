package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExec(t *testing.T) {
	t.Run("fails when not initialized", func(t *testing.T) {
		t.Setenv("HELMENV_ROOT", "")
		err := Exec("0.31.0", []string{"version"})
		if err == nil {
			t.Fatal("expected error when not initialized")
		}
	})

	t.Run("fails when version not installed", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HELMENV_ROOT", tmpDir)
		t.Setenv("HELMENV_AUTO_INSTALL", "")
		if err := os.MkdirAll(filepath.Join(tmpDir, "versions"), 0o755); err != nil {
			t.Fatal(err)
		}

		err := Exec("0.31.0", []string{"version"})
		if err == nil {
			t.Fatal("expected error when version not installed")
		}
	})

	t.Run("resolves fuzzy version to installed", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HELMENV_ROOT", tmpDir)
		t.Setenv("HELMENV_AUTO_INSTALL", "")
		for _, v := range []string{"3.14.0", "3.14.5"} {
			vdir := filepath.Join(tmpDir, "versions", v)
			if err := os.MkdirAll(vdir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(vdir, "helm"), []byte("binary"), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		// Use resolveExecVersion directly so we don't actually fork.
		got, err := resolveExecVersion("3.14", AutoDefault)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "3.14.5" {
			t.Fatalf("got %q, want 3.14.5", got)
		}
	})

	t.Run("fails when version or command missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HELMENV_ROOT", tmpDir)
		if err := os.MkdirAll(filepath.Join(tmpDir, "versions", "0.31.0"), 0o755); err != nil {
			t.Fatal(err)
		}

		err := Exec("", []string{"version"})
		if err == nil || err.Error() != "version not specified. Usage: helm-env exec <version> <command> [args...]" {
			t.Fatalf("expected version missing error, got %v", err)
		}

		err = Exec("0.31.0", []string{})
		if err == nil || err.Error() != "command not specified. Usage: helm-env exec <version> <command> [args...]" {
			t.Fatalf("expected command missing error, got %v", err)
		}
	})
}
