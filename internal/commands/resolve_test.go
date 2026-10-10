package commands

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/helm-env/internal/config"
	"github.com/user/helm-env/internal/github"
)

// seedInstalled creates $HELMENV_ROOT/versions/<v>/helm for each version.
func seedInstalled(t *testing.T, root string, versions ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "versions"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, v := range versions {
		vdir := filepath.Join(root, "versions", v)
		if err := os.MkdirAll(vdir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(vdir, "helm"), []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

// stubInstaller swaps installerFunc for one that just seeds the version on
// disk, and returns a pointer to the list of versions it was asked for.
func stubInstaller(t *testing.T, root string) *[]string {
	t.Helper()
	orig := installerFunc
	t.Cleanup(func() { installerFunc = orig })
	calls := &[]string{}
	installerFunc = func(_ *github.Client, version string, _ bool) error {
		*calls = append(*calls, version)
		seedInstalled(t, root, version)
		return nil
	}
	return calls
}

func TestResolve(t *testing.T) {
	t.Run("fails when not initialized", func(t *testing.T) {
		t.Setenv("HELMENV_ROOT", "")
		if err := Resolve("", false, false); err == nil {
			t.Fatal("expected error when not initialized")
		}
	})

	t.Run("prints exact installed version from shell", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("HELMENV_ROOT", root)
		t.Setenv("HELMENV_VERSION", "3.14.0")
		seedInstalled(t, root, "3.14.0")

		out := captureStdout(t, func() {
			if err := Resolve("", false, false); err != nil {
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
		seedInstalled(t, root, "3.14.0", "3.14.5", "3.15.0")

		out := captureStdout(t, func() {
			if err := Resolve("", false, false); err != nil {
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
		t.Setenv("HELMENV_AUTO_INSTALL", "")
		seedInstalled(t, root)
		if err := Resolve("", false, false); err == nil {
			t.Fatal("expected error when version not installed")
		}
	})
}

func TestResolve_ExplicitSpec(t *testing.T) {
	t.Run("explicit spec overrides active version", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("HELMENV_ROOT", root)
		t.Setenv("HELMENV_VERSION", "3.15.0")
		seedInstalled(t, root, "3.14.0", "3.14.5", "3.15.0")

		out := captureStdout(t, func() {
			if err := Resolve("~3.14", false, false); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if strings.TrimSpace(out) != "3.14.5" {
			t.Fatalf("got %q, want 3.14.5", out)
		}
	})

	t.Run("works without any configured version", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("HELMENV_ROOT", root)
		t.Setenv("HELMENV_VERSION", "")
		t.Chdir(t.TempDir())
		seedInstalled(t, root, "3.14.0")

		out := captureStdout(t, func() {
			if err := Resolve("3.14.0", false, false); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if strings.TrimSpace(out) != "3.14.0" {
			t.Fatalf("got %q, want 3.14.0", out)
		}
	})

	t.Run("unresolved spec fails with NotInstalledError", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("HELMENV_ROOT", root)
		t.Setenv("HELMENV_AUTO_INSTALL", "")
		seedInstalled(t, root, "3.14.0")

		err := Resolve("~3.99", false, false)
		var nie *config.NotInstalledError
		if !errors.As(err, &nie) {
			t.Fatalf("expected NotInstalledError, got %v", err)
		}
		if nie.Source != config.SourceArgument {
			t.Errorf("Source = %q, want %q", nie.Source, config.SourceArgument)
		}
	})

	t.Run("--install installs a missing exact version", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("HELMENV_ROOT", root)
		t.Setenv("HELMENV_AUTO_INSTALL", "")
		seedInstalled(t, root)
		calls := stubInstaller(t, root)

		var out string
		stderr := captureStderr(t, func() {
			out = captureStdout(t, func() {
				if err := resolveWithClient(github.NewClient(), "3.14.0", true, false); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		})
		if strings.TrimSpace(out) != "3.14.0" {
			t.Fatalf("stdout = %q, want only the bare version", out)
		}
		if len(*calls) != 1 || (*calls)[0] != "3.14.0" {
			t.Fatalf("installer calls = %v, want [3.14.0]", *calls)
		}
		if !strings.Contains(stderr, "auto-installing 3.14.0") {
			t.Errorf("expected auto-install notice on stderr, got %q", stderr)
		}
	})

	t.Run("HELMENV_AUTO_INSTALL enables install; --silent hides notice", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("HELMENV_ROOT", root)
		t.Setenv("HELMENV_AUTO_INSTALL", "1")
		seedInstalled(t, root)
		calls := stubInstaller(t, root)

		stderr := captureStderr(t, func() {
			_ = captureStdout(t, func() {
				if err := resolveWithClient(github.NewClient(), "3.14.0", false, true); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		})
		if len(*calls) != 1 {
			t.Fatalf("expected installer to be called once, got %v", *calls)
		}
		if stderr != "" {
			t.Errorf("--silent should suppress stderr, got %q", stderr)
		}
	})

	t.Run("without --install nothing is installed", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("HELMENV_ROOT", root)
		t.Setenv("HELMENV_AUTO_INSTALL", "")
		seedInstalled(t, root)
		calls := stubInstaller(t, root)

		if err := resolveWithClient(github.NewClient(), "3.14.0", false, false); err == nil {
			t.Fatal("expected error")
		}
		if len(*calls) != 0 {
			t.Fatalf("installer must not be called, got %v", *calls)
		}
	})
}

func TestResolveHelp(t *testing.T) {
	out := captureStdout(t, ResolveHelp)
	for _, want := range []string{"resolve [<spec>]", "--install", "-s, --silent", "HELMENV_AUTO_INSTALL"} {
		if !strings.Contains(out, want) {
			t.Errorf("resolve help missing %q; got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "--concrete") {
		t.Errorf("--concrete is a hidden legacy flag and must not be in help; got:\n%s", out)
	}
}
