package commands

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// seedFakeHelm installs a POSIX shell script as $HELMENV_ROOT/versions/<v>/helm.
func seedFakeHelm(t *testing.T, root, version, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake helm is a POSIX shell script")
	}
	vdir := filepath.Join(root, "versions", version)
	if err := os.MkdirAll(vdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vdir, "helm"), []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

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

func TestExec_ExitCodePassthrough(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_AUTO_INSTALL", "")
	seedFakeHelm(t, root, "3.14.0", "exit 3")

	err := ExecWithOptions("3.14.0", []string{"version"}, AutoDisabled)
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if got := exitErr.ExitCode(); got != 3 {
		t.Fatalf("exit code = %d, want 3", got)
	}
}

func TestExec_StripsOneLeadingDoubleDash(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_AUTO_INSTALL", "")
	argsFile := filepath.Join(root, "args")
	seedFakeHelm(t, root, "3.14.0", `printf '%s|' "$@" > "`+argsFile+`"; printf '%s' "$HELMENV_VERSION" > "`+argsFile+`.ver"`)

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"single dash-dash stripped", []string{"--", "version", "--short"}, "version|--short|"},
		{"only one stripped", []string{"--", "--", "x"}, "--|x|"},
		{"no dash-dash untouched", []string{"version"}, "version|"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ExecWithOptions("3.14", tc.args, AutoDisabled); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got, err := os.ReadFile(argsFile)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("helm got args %q, want %q", got, tc.want)
			}
			ver, _ := os.ReadFile(argsFile + ".ver")
			if string(ver) != "3.14.0" {
				t.Fatalf("HELMENV_VERSION = %q, want 3.14.0", ver)
			}
		})
	}

	t.Run("bare dash-dash is a missing command", func(t *testing.T) {
		err := ExecWithOptions("3.14.0", []string{"--"}, AutoDisabled)
		if err == nil || !strings.Contains(err.Error(), "command not specified") {
			t.Fatalf("expected command-missing error, got %v", err)
		}
	})
}

func TestExecHelp(t *testing.T) {
	out := captureStdout(t, ExecHelp)
	for _, want := range []string{"exec [--auto|--no-auto] <spec> [--]", "exit code", "HELMENV_VERSION"} {
		if !strings.Contains(out, want) {
			t.Errorf("exec help missing %q; got:\n%s", want, out)
		}
	}
}
