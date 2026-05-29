package platform

import "fmt"

// SelfDownloadURL returns the GitHub release download URL for a helm-env binary
// built for the given version and platform.
//
// Asset naming convention: helm-env-{os}-{arch}
// Example: https://github.com/mmpyro/helm-env/releases/download/v0.2.0/helm-env-darwin-arm64
func SelfDownloadURL(version string, info Info, ownerRepo string) string {
	return fmt.Sprintf(
		"https://github.com/%s/releases/download/v%s/helm-env-%s-%s",
		ownerRepo, version, info.OS, info.Arch,
	)
}
