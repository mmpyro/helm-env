package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/user/helm-env/internal/config"
	"github.com/user/helm-env/internal/github"
	"github.com/user/helm-env/internal/semver"
)

// AutoInstallMode controls whether missing versions should be installed
// automatically when resolution fails.
type AutoInstallMode int

const (
	// AutoDefault means "no explicit override — fall back to the
	// HELMENV_AUTO_INSTALL environment variable".
	AutoDefault AutoInstallMode = iota

	// AutoForce explicitly enables auto-install for this invocation.
	AutoForce

	// AutoDisabled explicitly disables auto-install for this invocation,
	// overriding the environment variable.
	AutoDisabled
)

// envAutoInstallVar is the environment variable consulted for auto-install.
const envAutoInstallVar = "HELMENV_AUTO_INSTALL"

// ParseAutoInstallEnv returns true when the given string is a truthy value
// for the HELMENV_AUTO_INSTALL env var.  Recognised values are:
// 1, true, yes, on (case-insensitive).  All other values (including empty)
// return false.
func ParseAutoInstallEnv(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// AutoFromEnv translates the current value of HELMENV_AUTO_INSTALL into an
// AutoInstallMode value: AutoForce when the env var is set to a truthy value,
// otherwise AutoDefault (so callers can still override via --no-auto).
func AutoFromEnv() AutoInstallMode {
	if ParseAutoInstallEnv(os.Getenv(envAutoInstallVar)) {
		return AutoForce
	}
	return AutoDefault
}

// AutoInstallEnabled reports whether the given mode results in auto-install
// being enabled for this invocation.  The rule is:
//
//   - AutoForce               → true
//   - AutoDisabled            → false
//   - AutoDefault             → value of HELMENV_AUTO_INSTALL env var
func AutoInstallEnabled(mode AutoInstallMode) bool {
	switch mode {
	case AutoForce:
		return true
	case AutoDisabled:
		return false
	}
	return ParseAutoInstallEnv(os.Getenv(envAutoInstallVar))
}

// installerFunc installs the given concrete helm version.  It is a package
// variable so tests can swap it out.
var installerFunc = defaultInstaller

func defaultInstaller(client *github.Client, version string, silent bool) error {
	return installWithClient(client, version, silent)
}

// autoInstallIfEnabledWith is the central entry point used by every command
// that wants to transparently install a missing version.
//
// It calls config.ResolveConcreteVersion first.  If the result is a
// NotInstalledError AND auto-install is enabled, it:
//
//  1. Resolves the constraint against the *remote* release list (reusing
//     getRemoteVersions / the baseline fallback).
//  2. Prints `helm-env: auto-installing <version>…` to stderr.
//  3. Calls Install(..., silent=true) with the concrete version.
//  4. Returns the newly installed concrete version.
//
// If auto-install is disabled, the original error is returned untouched.
//
// Callers inject the github client and stderr writer so this stays testable
// without reaching for package globals.
func autoInstallIfEnabledWith(mode AutoInstallMode, client *github.Client, stderr io.Writer) (string, error) {
	v, err := config.ResolveConcreteVersion()
	if err == nil {
		return v, nil
	}

	var nie *config.NotInstalledError
	if !errors.As(err, &nie) {
		return "", err
	}

	if !AutoInstallEnabled(mode) {
		return "", err
	}

	target := nie.Resolved
	if target == "" {
		// Resolve constraint against remote list.
		stable, pre, remoteErr := getRemoteVersions(client)
		if remoteErr != nil {
			return "", fmt.Errorf("auto-install: failed to fetch remote versions: %w", remoteErr)
		}
		candidates := stable
		// If the constraint explicitly wants prereleases, expose them too.
		// We also fall back to the prerelease list if the stable list has
		// no match — Feature 1's matcher will filter appropriately.
		if lower := strings.ToLower(nie.Requested); lower == semver.AliasLatestInstalled {
			candidates = pre
		} else if strings.ContainsAny(nie.Requested, "-") && !semver.IsAlias(nie.Requested) {
			// Explicit prerelease constraint.
			candidates = pre
		}
		matched, matchErr := semver.Match(nie.Requested, candidates)
		if matchErr != nil {
			// Try prerelease list as a last resort before giving up.
			if alt, altErr := semver.Match(nie.Requested, pre); altErr == nil {
				matched = alt
			} else {
				return "", fmt.Errorf("auto-install: no remote version matches %q: %w", nie.Requested, matchErr)
			}
		}
		target = matched
	}

	fmt.Fprintf(stderr, "helm-env: auto-installing %s…\n", target)
	if err := installerFunc(client, target, true); err != nil {
		return "", fmt.Errorf("auto-install failed: %w", err)
	}
	return target, nil
}
