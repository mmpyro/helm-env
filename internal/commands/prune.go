package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/helm-env/internal/config"
	"github.com/user/helm-env/internal/semver"
)

// PruneHelp prints help for the prune command.
func PruneHelp() {
	fmt.Println(`Usage: helm-env prune [flags]

Remove installed helm versions that are no longer needed.

Flags:
  --keep N          Keep the N newest installed versions.
  --older-than DUR  Remove versions whose helm binary mtime is older than DUR.
                    Accepts Go duration strings (e.g. 30m, 24h) plus "<N>d" for
                    whole days.
  --dry-run         Print what would be removed without touching the disk.
  -h, --help        Show this help message.

When --keep and --older-than are both supplied, a version must satisfy BOTH
rules to be kept.  The currently-resolved version is always protected and
will be skipped with a warning if it would otherwise be removed.`)
}

// Prune implements the `helm-env prune` command.
//
//	keep       Keep this many newest installed versions.
//	olderThan  Remove versions whose binary mtime is older than the parsed
//	           duration.  Empty string disables this check.
//	dryRun     Only print what would happen.
//
// Returns a non-nil error on bad flags, when helm-env is not initialized,
// or on filesystem errors.
func Prune(keep int, olderThan string, dryRun bool) error {
	if err := config.RequireInit(); err != nil {
		return err
	}
	if keep < 0 {
		return fmt.Errorf("--keep must be >= 0")
	}

	var olderThanDur time.Duration
	hasOlderThan := olderThan != ""
	if hasOlderThan {
		d, err := parseDurationWithDays(olderThan)
		if err != nil {
			return fmt.Errorf("invalid --older-than value %q: %w", olderThan, err)
		}
		olderThanDur = d
	}
	if !hasOlderThan && keep == 0 {
		return fmt.Errorf("at least one of --keep or --older-than is required")
	}

	installed, err := config.InstalledVersions()
	if err != nil {
		return err
	}
	if len(installed) == 0 {
		return nil
	}

	sorted := semver.SortDescending(installed)

	// Compute the "keep" set: initially everything.
	keepSet := make(map[string]bool, len(sorted))
	for _, v := range sorted {
		keepSet[v] = true
	}

	// Apply --keep: only the N newest stay in the keep set.
	if keep > 0 {
		for i, v := range sorted {
			if i >= keep {
				keepSet[v] = false
			}
		}
	}

	// Apply --older-than: ANYTHING newer than cutoff stays.  Anything older
	// is removed, but only if the other rule hasn't already decided to keep
	// it … actually: the spec says BOTH rules must agree.  Interpretation:
	// a version is kept iff it satisfies ALL supplied rules.  So we subtract.
	now := time.Now()
	if hasOlderThan {
		cutoff := now.Add(-olderThanDur)
		for _, v := range sorted {
			binary := binaryPathForVersion(v)
			info, err := os.Stat(binary)
			if err != nil {
				// Missing binary → treat as very old (remove candidate).
				keepSet[v] = false
				continue
			}
			if info.ModTime().After(cutoff) {
				// Newer than cutoff → candidate for keep.  BUT if --keep
				// already evicted it, keep that decision.
				// Nothing to do: keepSet[v] stays as --keep decided.
				continue
			}
			// Older than cutoff → remove.
			keepSet[v] = false
		}
	}

	// Protect the currently-resolved version.
	activeVersion, _ := config.ResolveConcreteVersion()

	for _, v := range sorted {
		if keepSet[v] {
			continue
		}
		if v == activeVersion {
			fmt.Fprintf(os.Stderr, "warning: refusing to prune active version %s\n", v)
			continue
		}
		if dryRun {
			fmt.Printf("[dry-run] would prune %s\n", v)
			continue
		}
		fmt.Printf("pruning %s\n", v)
		versionDir, err := config.GetVersionDir(v)
		if err != nil {
			return err
		}
		if err := os.RemoveAll(versionDir); err != nil {
			return fmt.Errorf("failed to remove %s: %w", versionDir, err)
		}
	}
	return nil
}

// binaryPathForVersion returns the expected helm binary path for a version
// without erroring if HELMENV_ROOT changes mid-flight.
func binaryPathForVersion(version string) string {
	root, _ := config.GetHelmEnvRoot()
	return filepath.Join(root, "versions", version, "helm")
}

// parseDurationWithDays extends time.ParseDuration to accept a trailing
// "d" for days.  "30d" → 30 * 24h.  Mixed strings like "1d12h" are also
// supported by looking for the first "d" and splitting there.
func parseDurationWithDays(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty duration")
	}

	// If "d" is present outside of pre-existing Go-recognised units ("ns",
	// "us", "ms", "s", "m", "h"), convert it to hours manually.
	// Simplest reliable implementation: scan for a numeric block followed by
	// a literal "d" and replace it with the equivalent hours expression.
	normalized, err := replaceDaysUnit(s)
	if err != nil {
		return 0, err
	}
	return time.ParseDuration(normalized)
}

// replaceDaysUnit converts occurrences of "<N>d" (where N is a positive
// integer) into "<N*24>h".  Fractional days are not supported.
func replaceDaysUnit(s string) (string, error) {
	var b strings.Builder
	i := 0
	for i < len(s) {
		// Try to match [0-9]+d.
		start := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i > start && i < len(s) && s[i] == 'd' {
			// Convert N days to N*24 hours.
			num := s[start:i]
			var days int
			if _, err := fmt.Sscanf(num, "%d", &days); err != nil {
				return "", fmt.Errorf("invalid days value %q: %w", num, err)
			}
			b.WriteString(fmt.Sprintf("%dh", days*24))
			i++ // skip the 'd'
			continue
		}
		// Not a days unit — copy from `start` through this char.
		if i > start {
			b.WriteString(s[start:i])
		}
		if i < len(s) {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String(), nil
}
