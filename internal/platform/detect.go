// Package platform provides OS and architecture detection for downloading
// the correct helm binary.
package platform

import (
	"fmt"
	"runtime"
)

// Info holds the detected platform information.
type Info struct {
	OS   string
	Arch string
}

// Detect returns the current platform's OS and architecture.
func Detect() (Info, error) {
	osName, err := mapOS(runtime.GOOS)
	if err != nil {
		return Info{}, err
	}
	archName, err := mapArch(runtime.GOARCH)
	if err != nil {
		return Info{}, err
	}
	return Info{OS: osName, Arch: archName}, nil
}

// mapOS maps Go's runtime.GOOS to the helm release naming convention.
func mapOS(goos string) (string, error) {
	switch goos {
	case "linux", "darwin", "windows":
		return goos, nil
	default:
		return "", fmt.Errorf("unsupported operating system: %s", goos)
	}
}

// mapArch maps Go's runtime.GOARCH to the helm release naming convention.
func mapArch(goarch string) (string, error) {
	switch goarch {
	case "amd64":
		return "amd64", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s", goarch)
	}
}

// DownloadPath returns the path part of the helm binary download URL.
// Helm release tarballs are hosted at https://get.helm.sh/ (not on GitHub
// release assets) and the filename embeds a "v" prefix in the version.
// Format: helm-v{version}-{os}-{arch}.tar.gz
func DownloadPath(version string, info Info) string {
	return ArchiveName(version, info)
}

// ArchiveName returns the helm archive name for the given version and platform.
func ArchiveName(version string, info Info) string {
	return fmt.Sprintf("helm-v%s-%s-%s.tar.gz", version, info.OS, info.Arch)
}

// BinaryName returns the name of the helm binary inside the archive.
func BinaryName(info Info) string {
	if info.OS == "windows" {
		return "helm.exe"
	}
	return "helm"
}

// ChecksumPath returns the path part of the URL for the per-file .sha256sum checksum.
// Helm publishes <archive>.sha256sum files at https://get.helm.sh/
// (format: "<sha>  <filename>").
func ChecksumPath(version string, info Info) string {
	return ArchiveName(version, info) + ".sha256sum"
}
