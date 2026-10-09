package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/user/helm-env/internal/config"
	"github.com/user/helm-env/internal/github"
	"github.com/user/helm-env/internal/semver"
)

// Exec runs a specific helm version without changing the active version.
// It preserves the historical "exact version required" semantics for
// backwards compatibility with existing callers; use ExecWithOptions for
// the richer surface (fuzzy version + auto-install control).
func Exec(version string, args []string) error {
	return ExecWithOptions(version, args, AutoFromEnv())
}

// ExecWithOptions is the richer form of Exec that accepts an explicit
// auto-install mode.
//
// Version resolution:
//   - If version is a plain exact semver AND is installed, use it as-is.
//   - Otherwise, resolve it as a constraint against installed versions
//     (fuzzy / alias / range support via Feature 1).
//   - If no installed version matches AND auto-install is enabled, resolve
//     against the remote release list and install the match before exec.
func ExecWithOptions(version string, args []string, auto AutoInstallMode) error {
	if err := config.RequireInit(); err != nil {
		return err
	}

	if version == "" {
		return fmt.Errorf("version not specified. Usage: helm-env exec <version> <command> [args...]")
	}
	if len(args) == 0 {
		return fmt.Errorf("command not specified. Usage: helm-env exec <version> <command> [args...]")
	}

	resolved, err := resolveExecVersion(version, auto)
	if err != nil {
		return err
	}

	binaryPath, err := config.GetBinaryPath(resolved)
	if err != nil {
		return err
	}

	cmd := exec.Command(binaryPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("HELMENV_VERSION=%s", resolved))
	return cmd.Run()
}

// resolveExecVersion resolves the user-supplied version argument to a
// concrete installed version, honouring auto-install when applicable.
func resolveExecVersion(version string, auto AutoInstallMode) (string, error) {
	// Fast path: exact semver already installed.
	if semver.IsExact(version) {
		installed, err := config.IsVersionInstalled(version)
		if err != nil {
			return "", err
		}
		if installed {
			return version, nil
		}
	}

	installed, err := config.InstalledVersions()
	if err != nil {
		return "", err
	}
	if resolved, err := semver.Match(version, installed); err == nil {
		return resolved, nil
	}

	// Not installed — maybe auto-install.
	if !AutoInstallEnabled(auto) {
		return "", fmt.Errorf("version %s is not installed", version)
	}
	return autoInstallFromRemote(version)
}

// autoInstallFromRemote resolves the constraint against the remote release
// list and installs the matched version.  Returns the concrete installed
// version on success.
func autoInstallFromRemote(constraint string) (string, error) {
	client := github.NewClient()
	stable, pre, err := getRemoteVersions(client)
	if err != nil {
		return "", fmt.Errorf("auto-install: failed to fetch remote versions: %w", err)
	}
	candidates := stable
	// If the constraint explicitly pins a prerelease, include prereleases.
	if hasPrereleaseHint(constraint) {
		candidates = pre
	}
	matched, matchErr := semver.Match(constraint, candidates)
	if matchErr != nil {
		if alt, altErr := semver.Match(constraint, pre); altErr == nil {
			matched = alt
		} else {
			return "", fmt.Errorf("auto-install: no remote version matches %q", constraint)
		}
	}
	fmt.Fprintf(os.Stderr, "helm-env: auto-installing %s…\n", matched)
	if err := installerFunc(client, matched, true); err != nil {
		return "", fmt.Errorf("auto-install failed: %w", err)
	}
	return matched, nil
}
