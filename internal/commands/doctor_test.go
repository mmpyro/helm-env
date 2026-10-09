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
		err := Doctor()
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
		if err := Doctor(); err != nil {
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
		_ = Doctor()
	})
	if !strings.Contains(out, "[WARN] reachable") {
		t.Fatalf("expected WARN for reachability, got %q", out)
	}
}
