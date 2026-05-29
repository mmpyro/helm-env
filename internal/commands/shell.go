package commands

import (
	"fmt"
	"os"

	"github.com/user/helm-env/internal/config"
)

// Shell manages the shell-level helm version.
// With a version argument: verifies it's installed and outputs export command.
// Without argument: prints the current shell version or errors.
func Shell(version string) error {
	if err := config.RequireInit(); err != nil {
		return err
	}

	if version == "" {
		// Print current shell version
		v := os.Getenv("HELMENV_VERSION")
		if v == "" {
			return fmt.Errorf("no shell version configured")
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

	// Output export command for the shell function wrapper to eval
	fmt.Printf("export HELMENV_VERSION=%s\n", version)
	return nil
}
