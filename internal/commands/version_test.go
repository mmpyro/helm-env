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
