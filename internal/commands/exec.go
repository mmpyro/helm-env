package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/user/helm-env/internal/config"
	"github.com/user/helm-env/internal/semver"
)

// Exec runs a specific helm version without changing the active version.
//
// Version resolution:
//   - If version is a plain exact semver AND is installed, use it as-is.
//   - Otherwise, resolve it as a constraint against installed versions
//     (fuzzy / alias / range support).
func Exec(version string, args []string) error {
	if err := config.RequireInit(); err != nil {
		return err
	}

	if version == "" {
		return fmt.Errorf("version not specified. Usage: helm-env exec <version> <command> [args...]")
	}
	if len(args) == 0 {
		return fmt.Errorf("command not specified. Usage: helm-env exec <version> <command> [args...]")
	}

	resolved, err := resolveExecVersion(version)
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
// concrete installed version.
func resolveExecVersion(version string) (string, error) {
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
	return "", fmt.Errorf("version %s is not installed", version)
}
