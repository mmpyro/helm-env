package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatus(t *testing.T) {
	t.Run("shows not initialized when HELMENV_ROOT not set", func(t *testing.T) {
		t.Setenv("HELMENV_ROOT", "")
		output := captureStdout(t, func() {
			err := Status()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(output, "helm-env is not initialized") {
			t.Fatalf("expected output to contain not initialized warning, got %q", output)
		}
	})

	t.Run("shows full status when initialized", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HELMENV_ROOT", tmpDir)
		t.Setenv("HELMENV_VERSION", "0.31.0")

		// Create versions directory and a version
		for _, v := range []string{"0.31.0", "0.30.0"} {
			if err := os.MkdirAll(filepath.Join(tmpDir, "versions", v), 0o755); err != nil {
				t.Fatal(err)
			}
		}

		output := captureStdout(t, func() {
			err := Status()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		if !strings.Contains(output, "HELMENV_ROOT:") || !strings.Contains(output, tmpDir) {
			t.Errorf("expected output to contain ROOT path %q, got %q", tmpDir, output)
		}
		if !strings.Contains(output, "Active version:") || !strings.Contains(output, "0.31.0") {
			t.Errorf("expected output to contain active version 0.31.0, got %q", output)
		}
		if !strings.Contains(output, "set by HELMENV_VERSION environment variable") {
			t.Errorf("expected output to contain source, got %q", output)
		}
		if !strings.Contains(output, "* 0.31.0") {
			t.Errorf("expected output to mark active version in list, got %q", output)
		}
		if !strings.Contains(output, "  0.30.0") {
			t.Errorf("expected output to contain other versions, got %q", output)
		}
		if !strings.Contains(output, "Auto-install:") {
			t.Errorf("expected output to include Auto-install row, got %q", output)
		}
	})

	t.Run("auto-install enabled reflected in status", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HELMENV_ROOT", tmpDir)
		t.Setenv("HELMENV_AUTO_INSTALL", "yes")
		if err := os.MkdirAll(filepath.Join(tmpDir, "versions"), 0o755); err != nil {
			t.Fatal(err)
		}
		output := captureStdout(t, func() {
			if err := Status(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(output, "Auto-install:") || !strings.Contains(output, "enabled") {
			t.Fatalf("expected Auto-install: enabled, got %q", output)
		}
	})
}
