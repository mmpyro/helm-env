package shim

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeFakeHelmEnv writes a fake `helm-env` script into dir and makes it
// executable.  The script writes every invocation (as space-separated
// arguments) as one line to logPath.  When called with "install --silent
// <version>", it creates $HELMENV_ROOT/versions/<version>/helm so a
// subsequent lookup succeeds.  When called with "resolve --concrete", it
// prints the first installed version it finds.
func writeFakeHelmEnv(t *testing.T, dir, logPath string) {
	t.Helper()
	script := `#!/bin/sh
LOG="` + logPath + `"
printf '%s\n' "$*" >> "$LOG"
case "$1" in
    install)
        shift
        # Skip --silent / -s if present.
        if [ "$1" = "--silent" ] || [ "$1" = "-s" ]; then
            shift
        fi
        VER="$1"
        if [ -z "$VER" ]; then
            exit 0
        fi
        mkdir -p "$HELMENV_ROOT/versions/$VER"
        # Make the fake helm binary echo its version.
        {
            echo '#!/bin/sh'
            echo 'echo "fake helm '"$VER"'"'
        } > "$HELMENV_ROOT/versions/$VER/helm"
        chmod +x "$HELMENV_ROOT/versions/$VER/helm"
        ;;
    resolve)
        # Return the first installed version we see.
        for d in "$HELMENV_ROOT/versions"/*; do
            [ -d "$d" ] || continue
            echo "$(basename "$d")"
            exit 0
        done
        exit 1
        ;;
    *)
        ;;
esac
exit 0
`
	path := filepath.Join(dir, "helm-env")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

// TestShim_FastPath_ExactInstalled verifies that the shim executes the
// installed binary without shelling out when the raw version is a plain
// semver string that is already installed.
func TestShim_FastPath_ExactInstalled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh-based shim not applicable on windows")
	}
	root := t.TempDir()
	if err := GenerateShimScript(root); err != nil {
		t.Fatalf("GenerateShimScript: %v", err)
	}
	// Install a fake helm 3.14.0 that prints a known string.
	binDir := filepath.Join(root, "versions", "3.14.0")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	helmBinary := "#!/bin/sh\necho installed-3.14.0\n"
	if err := os.WriteFile(filepath.Join(binDir, "helm"), []byte(helmBinary), 0o755); err != nil {
		t.Fatal(err)
	}

	// Fake helm-env on PATH that would fail if called — fast path must not call it.
	fakeDir := t.TempDir()
	logPath := filepath.Join(fakeDir, "calls.log")
	writeFakeHelmEnv(t, fakeDir, logPath)

	cmd := exec.Command(filepath.Join(root, "shims", "helm"))
	cmd.Env = append(os.Environ(),
		"HELMENV_ROOT="+root,
		"HELMENV_VERSION=3.14.0",
		"PATH="+fakeDir+":"+os.Getenv("PATH"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shim failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "installed-3.14.0") {
		t.Fatalf("expected shim to exec installed binary, got %q", out)
	}
	// Fast path should NOT have called helm-env.
	if data, _ := os.ReadFile(logPath); len(data) > 0 {
		t.Fatalf("fast path called helm-env unexpectedly: %q", data)
	}
}

// TestShim_AutoInstall_WhenEnabled runs the shim in a tempdir where the
// requested version is not yet installed.  With HELMENV_AUTO_INSTALL=1 set,
// the shim must invoke `helm-env install --silent <version>` and then exec
// the newly installed binary.
func TestShim_AutoInstall_WhenEnabled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh-based shim not applicable on windows")
	}
	root := t.TempDir()
	if err := GenerateShimScript(root); err != nil {
		t.Fatalf("GenerateShimScript: %v", err)
	}
	// versions dir must exist.
	if err := os.MkdirAll(filepath.Join(root, "versions"), 0o755); err != nil {
		t.Fatal(err)
	}

	fakeDir := t.TempDir()
	logPath := filepath.Join(fakeDir, "calls.log")
	writeFakeHelmEnv(t, fakeDir, logPath)

	cmd := exec.Command(filepath.Join(root, "shims", "helm"))
	cmd.Env = append(os.Environ(),
		"HELMENV_ROOT="+root,
		"HELMENV_VERSION=3.14.0",
		"HELMENV_AUTO_INSTALL=1",
		"PATH="+fakeDir+":"+os.Getenv("PATH"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shim failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "fake helm 3.14.0") {
		t.Fatalf("expected auto-installed binary to be executed, got %q", out)
	}
	// Verify helm-env install was called.
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("log read: %v", err)
	}
	if !strings.Contains(string(data), "install") {
		t.Fatalf("expected helm-env install to be called, got log:\n%s", data)
	}
}

// TestShim_NoAutoInstall_PrintsHint verifies that when auto-install is
// disabled, the shim prints an error with a hint about HELMENV_AUTO_INSTALL
// and exits non-zero.
func TestShim_NoAutoInstall_PrintsHint(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh-based shim not applicable on windows")
	}
	root := t.TempDir()
	if err := GenerateShimScript(root); err != nil {
		t.Fatalf("GenerateShimScript: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "versions"), 0o755); err != nil {
		t.Fatal(err)
	}

	fakeDir := t.TempDir()
	logPath := filepath.Join(fakeDir, "calls.log")
	writeFakeHelmEnv(t, fakeDir, logPath)

	cmd := exec.Command(filepath.Join(root, "shims", "helm"))
	cmd.Env = append(os.Environ(),
		"HELMENV_ROOT="+root,
		"HELMENV_VERSION=3.14.0",
		// Deliberately NOT setting HELMENV_AUTO_INSTALL.
		"PATH="+fakeDir+":"+os.Getenv("PATH"),
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected shim to fail when auto-install disabled, got output %q", out)
	}
	if !strings.Contains(string(out), "HELMENV_AUTO_INSTALL") {
		t.Fatalf("expected hint about HELMENV_AUTO_INSTALL, got %q", out)
	}
}
