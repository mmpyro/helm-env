package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/user/helm-env/internal/semver"
)

// VersionSource identifies which layer provided the active version.
type VersionSource string

const (
	// SourceShell is the HELMENV_VERSION environment variable.
	SourceShell VersionSource = "shell"

	// SourceLocal is a .helm-version file found by walking up the current
	// working directory.
	SourceLocal VersionSource = "local"

	// SourceGlobal is $HELMENV_ROOT/version.
	SourceGlobal VersionSource = "global"

	// SourceArgument is a spec passed explicitly on the command line
	// (e.g. `helm-env resolve ~3.14`).
	SourceArgument VersionSource = "argument"
)

// NotInstalledError is returned by ResolveConcreteVersion when a version
// has been resolved from the configured source (constraint) but is not
// installed on disk.  Callers (such as the auto-install flow) can use
// errors.As to detect it and act accordingly.
type NotInstalledError struct {
	// Requested is the raw constraint string from the config source
	// (e.g. "~3.14", "latest", "3.14.0").
	Requested string

	// Resolved is the concrete version string the constraint resolved to
	// against the *installed* candidates.  It is empty if no installed
	// version matches the constraint (common first-run scenario).
	Resolved string

	// Source identifies where the constraint came from (shell/local/global).
	Source VersionSource
}

// Error implements error.
func (e *NotInstalledError) Error() string {
	if e.Resolved != "" {
		return fmt.Sprintf("version %s is not installed", e.Resolved)
	}
	return fmt.Sprintf("no installed version matches %q", e.Requested)
}

// TraceStep is a single entry in a version-resolution trace.  It records
// what was examined and the outcome.  TraceStep values are produced by
// ResolveVersionTraced.
type TraceStep struct {
	// Source is a short label describing what layer this step belongs to
	// (shell / local / global / match / result).
	Source string

	// Detail is a human-readable description of the step.
	Detail string

	// Hit reports whether this step produced the final answer.
	Hit bool
}

// ResolveConcreteVersion is like ResolveVersion but also matches the
// resolved constraint against installed versions on disk.  It returns the
// concrete installed version string.  If the raw string resolves to a
// version that is not installed, it returns a *NotInstalledError.
func ResolveConcreteVersion() (string, error) {
	raw, err := ResolveVersion()
	if err != nil {
		return "", err
	}
	return matchAgainstInstalled(raw, deriveSource())
}

// ResolveSpec matches an explicit version spec (exact, partial, alias or
// range) against installed versions on disk and returns the concrete
// installed version.  It ignores the shell/local/global configuration.  If
// nothing installed matches, it returns a *NotInstalledError with
// Source == SourceArgument.
func ResolveSpec(spec string) (string, error) {
	if spec == "" {
		return "", fmt.Errorf("version spec must not be empty")
	}
	return matchAgainstInstalled(spec, SourceArgument)
}

// ResolveVersionTraced is like ResolveConcreteVersion but also returns a
// per-step trace of what was checked.  The trace is intended for the
// `helm-env which --explain` command.
func ResolveVersionTraced() (string, []TraceStep, error) {
	var trace []TraceStep

	// 1. Shell version.
	if v := os.Getenv("HELMENV_VERSION"); v != "" {
		trace = append(trace, TraceStep{
			Source: "shell",
			Detail: fmt.Sprintf("HELMENV_VERSION=%s", v),
			Hit:    true,
		})
		resolved, err := matchAgainstInstalledTraced(v, SourceShell, &trace)
		return resolved, trace, err
	}
	trace = append(trace, TraceStep{
		Source: "shell",
		Detail: "HELMENV_VERSION not set",
	})

	// 2. Local .helm-version — record every directory walked.
	cwd, err := os.Getwd()
	if err == nil {
		if v, hitDir, ok := findLocalWithTrace(cwd, &trace); ok {
			trace = append(trace, TraceStep{
				Source: "local",
				Detail: fmt.Sprintf("found .helm-version in %s: %s", hitDir, v),
				Hit:    true,
			})
			resolved, err := matchAgainstInstalledTraced(v, SourceLocal, &trace)
			return resolved, trace, err
		}
		trace = append(trace, TraceStep{
			Source: "local",
			Detail: "no .helm-version found walking up from " + cwd,
		})
	}

	// 3. Global version file.
	root, ok := GetHelmEnvRoot()
	if ok {
		if v, err := ReadGlobalVersion(root); err == nil && v != "" {
			trace = append(trace, TraceStep{
				Source: "global",
				Detail: fmt.Sprintf("$HELMENV_ROOT/version = %s", v),
				Hit:    true,
			})
			resolved, err := matchAgainstInstalledTraced(v, SourceGlobal, &trace)
			return resolved, trace, err
		}
		trace = append(trace, TraceStep{
			Source: "global",
			Detail: "$HELMENV_ROOT/version not set",
		})
	} else {
		trace = append(trace, TraceStep{
			Source: "global",
			Detail: "HELMENV_ROOT not set",
		})
	}

	return "", trace, fmt.Errorf("no helm version configured. Set a version using 'helm-env shell', 'helm-env local', or 'helm-env global'")
}

// InstalledVersions lists the directory names under $HELMENV_ROOT/versions
// that are considered installed (i.e. contain a `helm` binary).  Entries
// without a helm binary are skipped.
func InstalledVersions() ([]string, error) {
	root, ok := GetHelmEnvRoot()
	if !ok {
		return nil, fmt.Errorf("HELMENV_ROOT not set")
	}
	versionsDir := filepath.Join(root, "versions")
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var versions []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		binary := filepath.Join(versionsDir, e.Name(), "helm")
		info, err := os.Stat(binary)
		if err != nil || info.IsDir() {
			continue
		}
		versions = append(versions, e.Name())
	}
	return versions, nil
}

// deriveSource reports which layer the current ResolveVersion call reads
// from.  It mirrors ResolveVersion exactly to allow NotInstalledError to
// report where the raw constraint came from.
func deriveSource() VersionSource {
	if v := os.Getenv("HELMENV_VERSION"); v != "" {
		return SourceShell
	}
	cwd, err := os.Getwd()
	if err == nil {
		if v, _ := FindLocalVersionFrom(cwd); v != "" {
			return SourceLocal
		}
	}
	return SourceGlobal
}

// matchAgainstInstalled resolves raw (possibly a constraint) against the
// installed candidates on disk.  It returns the concrete installed version,
// or a *NotInstalledError if the constraint resolves to something not
// present on disk.
func matchAgainstInstalled(raw string, src VersionSource) (string, error) {
	installed, err := InstalledVersions()
	if err != nil {
		return "", err
	}

	// Fast path: if raw is an exact semver AND it is installed, return it.
	if semver.IsExact(raw) {
		for _, v := range installed {
			if v == raw {
				return v, nil
			}
		}
		return "", &NotInstalledError{Requested: raw, Resolved: raw, Source: src}
	}

	// Range / partial / alias — match against installed.
	matched, err := semver.Match(raw, installed)
	if err != nil {
		// For aliases/latest-installed this triggers only when there are no
		// installed versions.  For other constraints it means the constraint
		// does not match any installed version — which Feature 2 may want to
		// auto-install from the remote list.
		return "", &NotInstalledError{Requested: raw, Resolved: "", Source: src}
	}
	return matched, nil
}

func matchAgainstInstalledTraced(raw string, src VersionSource, trace *[]TraceStep) (string, error) {
	resolved, err := matchAgainstInstalled(raw, src)
	if err != nil {
		*trace = append(*trace, TraceStep{
			Source: "match",
			Detail: fmt.Sprintf("resolved %q to no installed version", raw),
		})
		return "", err
	}
	if resolved == raw {
		*trace = append(*trace, TraceStep{
			Source: "match",
			Detail: fmt.Sprintf("%q matches installed version %q", raw, resolved),
			Hit:    true,
		})
	} else {
		*trace = append(*trace, TraceStep{
			Source: "match",
			Detail: fmt.Sprintf("%q resolves to installed version %q", raw, resolved),
			Hit:    true,
		})
	}
	return resolved, nil
}

// findLocalWithTrace walks up from dir looking for .helm-version and
// records each parent directory considered.  When the file is found, it
// returns (version, dir, true).
func findLocalWithTrace(dir string, trace *[]TraceStep) (string, string, bool) {
	for {
		path := filepath.Join(dir, ".helm-version")
		if data, err := os.ReadFile(path); err == nil {
			v := trimNewline(string(data))
			if v != "" {
				return v, dir, true
			}
			*trace = append(*trace, TraceStep{
				Source: "local",
				Detail: "found empty .helm-version in " + dir,
			})
		} else {
			*trace = append(*trace, TraceStep{
				Source: "local",
				Detail: "no .helm-version in " + dir,
			})
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", false
		}
		dir = parent
	}
}

// trimNewline is a minimal strings.TrimSpace wrapper that keeps this
// package free of the strings import if we ever drop other callers.
func trimNewline(s string) string {
	// We trim leading/trailing whitespace to match the behaviour of
	// FindLocalVersionFrom (which uses strings.TrimSpace).
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 {
		last := s[len(s)-1]
		if last == ' ' || last == '\t' || last == '\n' || last == '\r' {
			s = s[:len(s)-1]
			continue
		}
		break
	}
	return s
}
