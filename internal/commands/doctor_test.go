package commands

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/helm-env/internal/shim"
)

// stubHTTP is a stub implementation of httpClient for doctor tests.
type stubHTTP struct {
	err error
}

func (s *stubHTTP) Head(url string) (*http.Response, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
}

func TestDoctor_FailWhenRootUnset(t *testing.T) {
	t.Setenv("HELMENV_ROOT", "")
	// Use a stub HTTP client so the test is offline-safe.
	orig := doctorHTTPClient
	defer func() { doctorHTTPClient = orig }()
	doctorHTTPClient = &stubHTTP{}

	out := captureStdout(t, func() {
		err := Doctor(false)
		if err == nil {
			t.Fatal("expected error when root unset")
		}
	})
	if !strings.Contains(out, "[FAIL] HELMENV_ROOT is set") {
		t.Fatalf("expected FAIL for HELMENV_ROOT, got %q", out)
	}
}

func TestDoctor_HappyPath(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_VERSION", "3.14.0")

	// Scaffold: versions dir, shim, installed version.
	if err := os.MkdirAll(filepath.Join(root, "versions", "3.14.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "versions", "3.14.0", "helm"), []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := shim.GenerateShimScript(root); err != nil {
		t.Fatalf("GenerateShimScript: %v", err)
	}
	// Prepend shim dir to PATH so the LookPath check hits our shim first.
	t.Setenv("PATH", filepath.Join(root, "shims")+string(os.PathListSeparator)+os.Getenv("PATH"))

	// Stub HTTP client.
	orig := doctorHTTPClient
	defer func() { doctorHTTPClient = orig }()
	doctorHTTPClient = &stubHTTP{}

	out := captureStdout(t, func() {
		if err := Doctor(false); err != nil {
			t.Fatalf("unexpected FAIL: %v", err)
		}
	})
	if strings.Contains(out, "[FAIL]") {
		t.Fatalf("expected no FAIL in output, got %q", out)
	}
	if !strings.Contains(out, "[OK] HELMENV_ROOT is set") {
		t.Fatalf("expected OK HELMENV_ROOT, got %q", out)
	}
	if !strings.Contains(out, "[OK] active version resolves") {
		t.Fatalf("expected OK active version resolves, got %q", out)
	}
}

func TestDoctor_WarnOnNetworkFailure(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	if err := os.MkdirAll(filepath.Join(root, "versions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := shim.GenerateShimScript(root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(root, "shims")+string(os.PathListSeparator)+os.Getenv("PATH"))

	orig := doctorHTTPClient
	defer func() { doctorHTTPClient = orig }()
	doctorHTTPClient = &stubHTTP{err: errors.New("boom")}

	out := captureStdout(t, func() {
		// Should not fail even with network errors (reachability is WARN).
		_ = Doctor(false)
	})
	if !strings.Contains(out, "[WARN] reachable") {
		t.Fatalf("expected WARN for reachability, got %q", out)
	}
}

func TestDoctor_TamperedShimFailsAndFixRegenerates(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	if err := os.MkdirAll(filepath.Join(root, "versions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := shim.GenerateShimScript(root); err != nil {
		t.Fatal(err)
	}
	shimPath := filepath.Join(root, "shims", "helm")
	if err := os.WriteFile(shimPath, []byte("#!/bin/sh\necho tampered\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(root, "shims")+string(os.PathListSeparator)+os.Getenv("PATH"))

	orig := doctorHTTPClient
	defer func() { doctorHTTPClient = orig }()
	doctorHTTPClient = &stubHTTP{}

	out := captureStdout(t, func() {
		if err := Doctor(false); err == nil {
			t.Fatal("expected FAIL for a tampered shim")
		}
	})
	if !strings.Contains(out, "[FAIL] $HELMENV_ROOT/shims/helm is up to date") {
		t.Fatalf("expected shim drift FAIL, got %q", out)
	}

	out = captureStdout(t, func() {
		if err := Doctor(true); err != nil {
			t.Fatalf("unexpected FAIL after --fix: %v", err)
		}
	})
	if !strings.Contains(out, "[FIX] regenerated "+shimPath) {
		t.Fatalf("expected [FIX] line, got %q", out)
	}
	data, err := os.ReadFile(shimPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != shim.ShimScript(root) {
		t.Fatal("--fix should restore the generated shim")
	}
}

func TestDoctor_FixMakesBinariesExecutable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	vdir := filepath.Join(root, "versions", "3.14.0")
	if err := os.MkdirAll(vdir, 0o755); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(vdir, "helm")
	if err := os.WriteFile(binary, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() { doctorFix(os.Stdout) })
	info, err := os.Stat(binary)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("expected %s to be executable after fix; output %q", binary, out)
	}
}
