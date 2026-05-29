package commands

import (
	"fmt"
	"os"

	"github.com/user/helm-env/internal/config"
)

// Local manages the local (directory-level) helm version.
// With a version argument: verifies it's installed and writes .helm-version.
// Without argument: reads and prints the local version or errors.
func Local(version string) error {
	if err := config.RequireInit(); err != nil {
		return err
	}

	if version == "" {
		// Read local version
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		v, err := config.FindLocalVersionFrom(cwd)
		if err != nil {
			return fmt.Errorf("no local version configured for this directory")
		}
		fmt.Println(v)
		return nil
	}

	// Verify version is installed
	installed, err := config.IsVersionInstalled(version)
	if err != nil {
		return err
	}
	if !installed {
		return fmt.Errorf("version %s not installed", version)
	}

	// Write .helm-version in current directory
	if err := os.WriteFile(".helm-version", []byte(version+"\n"), 0o644); err != nil {
		return fmt.Errorf("failed to write .helm-version: %w", err)
	}

	return nil
}
