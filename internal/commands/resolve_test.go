package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	t.Run("fails when not initialized", func(t *testing.T) {
		t.Setenv("HELMENV_ROOT", "")
		if err := Resolve(true); err == nil {
			t.Fatal("expected error when not initialized")
		}
	})

	t.Run("prints exact installed version from shell", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("HELMENV_ROOT", root)
		t.Setenv("HELMENV_VERSION", "3.14.0")
		vdir := filepath.Join(root, "versions", "3.14.0")
		if err := os.MkdirAll(vdir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(vdir, "helm"), []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}

		out := captureStdout(t, func() {
			if err := Resolve(true); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if strings.TrimSpace(out) != "3.14.0" {
			t.Fatalf("got %q, want 3.14.0", out)
		}
	})

	t.Run("resolves partial constraint against installed", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("HELMENV_ROOT", root)
		t.Setenv("HELMENV_VERSION", "3.14")
		for _, v := range []string{"3.14.0", "3.14.5", "3.15.0"} {
			vdir := filepath.Join(root, "versions", v)
			if err := os.MkdirAll(vdir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(vdir, "helm"), []byte("x"), 0o755); err != nil {
				t.Fatal(err)
			}
		}

		out := captureStdout(t, func() {
			if err := Resolve(true); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if strings.TrimSpace(out) != "3.14.5" {
			t.Fatalf("got %q, want 3.14.5", out)
		}
	})

	t.Run("fails when configured version is not installed", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("HELMENV_ROOT", root)
		t.Setenv("HELMENV_VERSION", "3.14.0")
		if err := os.MkdirAll(filepath.Join(root, "versions"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := Resolve(true); err == nil {
			t.Fatal("expected error when version not installed")
		}
	})
}
