package commands

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout captures stdout output from a function call.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// captureBoth captures stdout and stderr from a function call.
//
// Shared test helper used by stacked PRs: the `which` and `prune` tests
// introduced by Feature 5 both need combined stdout/stderr capture because
// their --explain and dry-run flows split output across the two streams.
// Keeping the helper here lets Feature 5 land as a leaf change that only
// adds new _test.go files instead of reaching back into this package.
//
//nolint:unused // consumed by which_test.go and prune_test.go in stacked PRs
func captureBoth(t *testing.T, fn func()) (string, string) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = wOut
	os.Stderr = wErr

	fn()

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr

	var bufOut, bufErr bytes.Buffer
	if _, err := io.Copy(&bufOut, rOut); err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(&bufErr, rErr); err != nil {
		t.Fatal(err)
	}
	return bufOut.String(), bufErr.String()
}

func TestPrintVersion(t *testing.T) {
	t.Run("prints version", func(t *testing.T) {
		Version = "1.2.3"
		defer func() { Version = "dev" }()

		output := captureStdout(t, PrintVersion)
		if strings.TrimSpace(output) != "1.2.3" {
			t.Fatalf("expected '1.2.3', got %q", strings.TrimSpace(output))
		}
	})

	t.Run("prints dev when not set", func(t *testing.T) {
		Version = "dev"
		output := captureStdout(t, PrintVersion)
		if strings.TrimSpace(output) != "dev" {
			t.Fatalf("expected 'dev', got %q", strings.TrimSpace(output))
		}
	})
}
