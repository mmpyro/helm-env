// Package config provides configuration and version resolution logic for helm-env.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetHelmEnvRoot reads the HELMENV_ROOT environment variable.
// Returns the path and true if set, or empty string and false if not.
func GetHelmEnvRoot() (string, bool) {
	root := os.Getenv("HELMENV_ROOT")
	if root == "" {
		return "", false
	}
	return root, true
}

// IsInitialized checks if HELMENV_ROOT is set and the versions directory exists.
func IsInitialized() bool {
	root, ok := GetHelmEnvRoot()
	if !ok {
		return false
	}
	versionsDir := filepath.Join(root, "versions")
	info, err := os.Stat(versionsDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// RequireInit checks that helm-env is initialized. If not, it prints an error
// message and returns a non-nil error.
func RequireInit() error {
	root, ok := GetHelmEnvRoot()
	if !ok {
		return fmt.Errorf("helm-env is not initialized. HELMENV_ROOT is not set.\nRun 'helm-env init' to initialize")
	}
	versionsDir := filepath.Join(root, "versions")
	if _, err := os.Stat(versionsDir); os.IsNotExist(err) {
		return fmt.Errorf("helm-env is not initialized. Run 'helm-env init' to initialize")
	}
	return nil
}

// ResolveVersion determines which helm version to use based on priority:
//  1. HELMENV_VERSION environment variable (shell version)
//  2. .helm-version file in current or parent directories (local version)
//  3. $HELMENV_ROOT/version file (global version)
//
// Returns the version string and nil error, or empty string and error if no version is configured.
func ResolveVersion() (string, error) {
	// 1. Check HELMENV_VERSION env var (shell version)
	if v := os.Getenv("HELMENV_VERSION"); v != "" {
		return strings.TrimSpace(v), nil
	}

	// 2. Walk up directories looking for .helm-version (local version)
	if v, err := findLocalVersion(); err == nil && v != "" {
		return v, nil
	}

	// 3. Check global version file
	if v, err := readGlobalVersion(); err == nil && v != "" {
		return v, nil
	}

	return "", fmt.Errorf("no helm version configured. Set a version using 'helm-env shell', 'helm-env local', or 'helm-env global'")
}

// findLocalVersion walks up from the current working directory looking for
// a .helm-version file.
func findLocalVersion() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return FindLocalVersionFrom(dir)
}

// FindLocalVersionFrom walks up from the given directory looking for
// a .helm-version file. Exported for testing.
func FindLocalVersionFrom(dir string) (string, error) {
	for {
		versionFile := filepath.Join(dir, ".helm-version")
		data, err := os.ReadFile(versionFile)
		if err == nil {
			v := strings.TrimSpace(string(data))
			if v != "" {
				return v, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no .helm-version file found")
}

// readGlobalVersion reads the global version from $HELMENV_ROOT/version.
func readGlobalVersion() (string, error) {
	root, ok := GetHelmEnvRoot()
	if !ok {
		return "", fmt.Errorf("HELMENV_ROOT not set")
	}
	return ReadGlobalVersion(root)
}

// ReadGlobalVersion reads the global version from the given root directory.
// Exported for testing.
func ReadGlobalVersion(root string) (string, error) {
	versionFile := filepath.Join(root, "version")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(string(data))
	if v == "" {
		return "", fmt.Errorf("global version file is empty")
	}
	return v, nil
}

// GetVersionDir returns the path to the directory for a specific version.
func GetVersionDir(version string) (string, error) {
	root, ok := GetHelmEnvRoot()
	if !ok {
		return "", fmt.Errorf("HELMENV_ROOT not set")
	}
	return filepath.Join(root, "versions", version), nil
}

// GetBinaryPath returns the path to the helm binary for a specific version.
func GetBinaryPath(version string) (string, error) {
	versionDir, err := GetVersionDir(version)
	if err != nil {
		return "", err
	}
	return filepath.Join(versionDir, "helm"), nil
}

// IsVersionInstalled checks if a specific version is installed.
func IsVersionInstalled(version string) (bool, error) {
	binaryPath, err := GetBinaryPath(version)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(binaryPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
