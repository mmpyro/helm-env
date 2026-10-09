package commands

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/helm-env/internal/config"
	"github.com/user/helm-env/internal/github"
)

func TestParseAutoInstallEnv(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"yes", true},
		{"on", true},
		{" 1 ", true},
		{"0", false},
		{"false", false},
		{"no", false},
		{"off", false},
		{"", false},
		{"something", false},
	}
	for _, tt := range tests {
		if got := ParseAutoInstallEnv(tt.in); got != tt.want {
			t.Errorf("ParseAutoInstallEnv(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestAutoInstallEnabled(t *testing.T) {
	t.Setenv("HELMENV_AUTO_INSTALL", "")
	if AutoInstallEnabled(AutoForce) != true {
		t.Fatal("AutoForce must be true regardless of env")
	}
	if AutoInstallEnabled(AutoDisabled) != false {
		t.Fatal("AutoDisabled must be false regardless of env")
	}
	if AutoInstallEnabled(AutoDefault) != false {
		t.Fatal("AutoDefault must be false when env not set")
	}
	t.Setenv("HELMENV_AUTO_INSTALL", "1")
	if AutoInstallEnabled(AutoDefault) != true {
		t.Fatal("AutoDefault must honour env=1")
	}
	t.Setenv("HELMENV_AUTO_INSTALL", "no")
	if AutoInstallEnabled(AutoDefault) != false {
		t.Fatal("AutoDefault must respect env=no")
	}
}

func TestAutoFromEnv(t *testing.T) {
	t.Setenv("HELMENV_AUTO_INSTALL", "yes")
	if AutoFromEnv() != AutoForce {
		t.Fatal("expected AutoForce when env is yes")
	}
	t.Setenv("HELMENV_AUTO_INSTALL", "")
	if AutoFromEnv() != AutoDefault {
		t.Fatal("expected AutoDefault when env is empty")
	}
}

// TestAutoInstallIfEnabled_StubbedInstaller verifies that
// autoInstallIfEnabledWith triggers the install flow when enabled, and
// skips it when disabled.
func TestAutoInstallIfEnabled_StubbedInstaller(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HELMENV_ROOT", root)
	t.Setenv("HELMENV_VERSION", "3.14.0")
	if err := os.MkdirAll(filepath.Join(root, "versions"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Swap installerFunc so we don't hit the network.
	orig := installerFunc
	defer func() { installerFunc = orig }()
	installed := []string{}
	installerFunc = func(_ *github.Client, version string, silent bool) error {
		installed = append(installed, version)
		vdir := filepath.Join(root, "versions", version)
		if err := os.MkdirAll(vdir, 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(vdir, "helm"), []byte("x"), 0o755)
	}

	t.Run("disabled: returns NotInstalledError", func(t *testing.T) {
		t.Setenv("HELMENV_AUTO_INSTALL", "")
		_, err := autoInstallIfEnabledWith(AutoDefault, github.NewClient(), io_discard())
		var nie *config.NotInstalledError
		if !errors.As(err, &nie) {
			t.Fatalf("expected NotInstalledError, got %v", err)
		}
	})

	t.Run("enabled via --auto: installs resolved version", func(t *testing.T) {
		t.Setenv("HELMENV_AUTO_INSTALL", "")
		var stderr bytes.Buffer
		v, err := autoInstallIfEnabledWith(AutoForce, github.NewClient(), &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "3.14.0" {
			t.Fatalf("got %q, want 3.14.0", v)
		}
		if len(installed) == 0 {
			t.Fatal("expected installerFunc to be called")
		}
		if !strings.Contains(stderr.String(), "auto-installing 3.14.0") {
			t.Fatalf("expected stderr message, got %q", stderr.String())
		}

		// Clean up for the next subtest.
		if err := os.RemoveAll(filepath.Join(root, "versions", "3.14.0")); err != nil {
			t.Fatal(err)
		}
		installed = nil
	})

	t.Run("enabled via env: installs resolved version", func(t *testing.T) {
		t.Setenv("HELMENV_AUTO_INSTALL", "1")
		var stderr bytes.Buffer
		v, err := autoInstallIfEnabledWith(AutoDefault, github.NewClient(), &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "3.14.0" {
			t.Fatalf("got %q, want 3.14.0", v)
		}
		if len(installed) == 0 {
			t.Fatal("expected installerFunc to be called")
		}
	})
}

// io_discard is a tiny wrapper to avoid importing "io" at the top of this test
// file (keeps the file compact).  It just returns io.Discard.
func io_discard() *bytes.Buffer { //nolint:revive
	return &bytes.Buffer{}
}
