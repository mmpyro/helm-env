package cache

// baselineVersions is a hardcoded list of historically known helm
// stable releases up to the time this version of helm-env was built.
//
// Purpose: provide a zero-network starting point so that:
//  1. The very first invocation (no disk cache yet) can return a useful list
//     without fetching all pages from GitHub.
//  2. If the network is unavailable, the command still returns something
//     meaningful rather than failing entirely.
//
// Maintenance: append new versions here when cutting a new helm-env release.
// The delta-fetch logic will automatically pick up anything newer than the
// last entry in this list, so the list does not need to be exhaustively
// up-to-date — it just needs to be a reasonable lower bound.
//
// Versions are stored newest-first (descending semver order) to match the
// output format of list-remote and to make it easy to find the newest entry.
var baselineVersions = []string{
	// 4.x
	"4.1.4",
	"4.1.3",
	"4.1.1",
	"4.1.0",
	"4.0.5",
	"4.0.4",
	"4.0.2",
	"4.0.1",
	"4.0.0",
	// 3.20.x
	"3.20.2",
	"3.20.1",
	"3.20.0",
	// 3.19.x
	"3.19.5",
	"3.19.4",
	"3.19.3",
	"3.19.2",
	"3.19.1",
	"3.19.0",
	// 3.18.x
	"3.18.6",
	"3.18.5",
	"3.18.4",
	"3.18.3",
	"3.18.2",
	"3.18.1",
	"3.18.0",
	// 3.17.x
	"3.17.4",
	"3.17.3",
	"3.17.2",
	"3.17.1",
	"3.17.0",
	// 3.16.x
	"3.16.4",
	"3.16.3",
	"3.16.2",
	"3.16.1",
	"3.16.0",
	// 3.15.x
	"3.15.4",
	"3.15.3",
	"3.15.2",
	"3.15.1",
	"3.15.0",
	// 3.14.x
	"3.14.4",
	"3.14.3",
	"3.14.2",
	"3.14.1",
	"3.14.0",
	// 3.13.x
	"3.13.3",
	"3.13.2",
	"3.13.1",
	"3.13.0",
	// 3.12.x
	"3.12.3",
	"3.12.2",
	"3.12.1",
	"3.12.0",
	// 3.11.x
	"3.11.3",
	"3.11.2",
	"3.11.1",
	"3.11.0",
	// 3.10.x
	"3.10.3",
	"3.10.2",
	"3.10.1",
	"3.10.0",
}

// baselinePrereleaseVersions is the same as baselineVersions but also includes
// known pre-release versions.  It is used when --prereleases is requested.
var baselinePrereleaseVersions = []string{
	// 4.x
	"4.1.4",
	"4.1.3",
	"4.1.1",
	"4.1.0",
	"4.1.0-rc.1",
	"4.0.5",
	"4.0.4",
	"4.0.2",
	"4.0.1",
	"4.0.0",
	"4.0.0-rc.1",
	"4.0.0-beta.2",
	"4.0.0-beta.1",
	"4.0.0-alpha.1",
	// 3.20.x
	"3.20.2",
	"3.20.1",
	"3.20.0",
	"3.20.0-rc.1",
	// 3.19.x
	"3.19.5",
	"3.19.4",
	"3.19.3",
	"3.19.2",
	"3.19.1",
	"3.19.0",
	"3.19.0-rc.1",
	// 3.18.x
	"3.18.6",
	"3.18.5",
	"3.18.4",
	"3.18.3",
	"3.18.2",
	"3.18.1",
	"3.18.0",
	"3.18.0-rc.2",
	"3.18.0-rc.1",
	// 3.17.x
	"3.17.4",
	"3.17.3",
	"3.17.2",
	"3.17.1",
	"3.17.0",
	"3.17.0-rc.1",
	// 3.16.x
	"3.16.4",
	"3.16.3",
	"3.16.2",
	"3.16.1",
	"3.16.0",
	"3.16.0-rc.1",
	// 3.15.x
	"3.15.4",
	"3.15.3",
	"3.15.2",
	"3.15.1",
	"3.15.0",
	"3.15.0-rc.2",
	"3.15.0-rc.1",
	// 3.14.x
	"3.14.4",
	"3.14.3",
	"3.14.2",
	"3.14.1",
	"3.14.0",
	"3.14.0-rc.1",
	// 3.13.x
	"3.13.3",
	"3.13.2",
	"3.13.1",
	"3.13.0",
	"3.13.0-rc.1",
	// 3.12.x
	"3.12.3",
	"3.12.2",
	"3.12.1",
	"3.12.0",
	"3.12.0-rc.1",
	// 3.11.x
	"3.11.3",
	"3.11.2",
	"3.11.1",
	"3.11.0",
	"3.11.0-rc.2",
	"3.11.0-rc.1",
	// 3.10.x
	"3.10.3",
	"3.10.2",
	"3.10.1",
	"3.10.0",
	"3.10.0-rc.1",
}

// BaselineVersions returns a copy of the hardcoded stable version list.
// The caller receives a fresh slice and may modify it freely.
func BaselineVersions() []string {
	out := make([]string, len(baselineVersions))
	copy(out, baselineVersions)
	return out
}

// BaselinePrereleaseVersions returns a copy of the hardcoded version list
// that includes pre-release entries.
func BaselinePrereleaseVersions() []string {
	out := make([]string, len(baselinePrereleaseVersions))
	copy(out, baselinePrereleaseVersions)
	return out
}

// BaselineNewest returns the newest version present in the baseline list,
// or an empty string if the baseline is empty.  This is used as the anchor
// for delta-fetching: only releases newer than this version are fetched from
// the GitHub API.
func BaselineNewest() string {
	if len(baselineVersions) == 0 {
		return ""
	}
	// The list is stored newest-first, so index 0 is the newest.
	return baselineVersions[0]
}
