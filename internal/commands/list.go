package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/user/helm-env/internal/config"
	"github.com/user/helm-env/internal/semver"
)

// List prints all installed helm versions.
func List() error {
	if err := config.RequireInit(); err != nil {
		return err
	}

	root, _ := config.GetHelmEnvRoot()
	versionsDir := filepath.Join(root, "versions")

	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return fmt.Errorf("failed to read versions directory: %w", err)
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			versions = append(versions, entry.Name())
		}
	}

	versions = semver.SortDescending(versions)

	for _, v := range versions {
		fmt.Println(v)
	}

	return nil
}
