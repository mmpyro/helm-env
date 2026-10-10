package commands

import (
	"fmt"
	"os"

	"github.com/user/helm-env/internal/config"
)

// UninstallHelp prints help for the uninstall command.
func UninstallHelp() {
	fmt.Println(`Usage: helm-env uninstall <version>

Remove an installed helm version from $HELMENV_ROOT/versions.

Flags:
  -h, --help   Show this help message.`)
}

// Uninstall removes an installed helm version.
func Uninstall(version string) error {
	if err := config.RequireInit(); err != nil {
		return err
	}

	if version == "" {
		return fmt.Errorf("version argument is required. Usage: helm-env uninstall <version>")
	}

	// Check if version is installed
	installed, err := config.IsVersionInstalled(version)
	if err != nil {
		return err
	}
	if !installed {
		return fmt.Errorf("version %s is not installed", version)
	}

	// Remove version directory
	versionDir, err := config.GetVersionDir(version)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(versionDir); err != nil {
		return fmt.Errorf("failed to remove version %s: %w", version, err)
	}

	fmt.Printf("version %s uninstalled\n", version)
	return nil
}
