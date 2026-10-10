package semver

import (
	"fmt"
	"strconv"
	"strings"
)

// Aliases.
const (
	// AliasLatest and AliasLatestStable resolve to the highest stable
	// (non-prerelease) candidate.
	AliasLatest       = "latest"
	AliasLatestStable = "latest-stable"
	AliasStable       = "stable"

	// AliasLatestInstalled resolves to the highest candidate — regardless
	// of pre-release status.  Semantically the caller is expected to pass
	// only *installed* versions as candidates, so this is effectively
	// "highest installed".
	AliasLatestInstalled = "latest-installed"
)

// IsAlias reports whether s is one of the recognised alias strings.
func IsAlias(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case AliasLatest, AliasLatestStable, AliasStable, AliasLatestInstalled:
		return true
	}
	return false
}

// IsExact reports whether s is a plain MAJOR.MINOR.PATCH[-prerelease] string
// — i.e. not a partial, alias, or range constraint.  This is used by the
// shim fast-path to avoid shelling out when the user has stored an exact
// version in their config files.
func IsExact(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// Any range operator or alias marker disqualifies.
	if strings.ContainsAny(s, "~^<>=,") {
		return false
	}
	if IsAlias(s) {
		return false
	}
	trimmed := strings.TrimPrefix(s, "v")
	core := strings.SplitN(trimmed, "-", 2)[0]
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		if _, err := strconv.Atoi(p); err != nil {
			return false
		}
	}
	return true
}

// Match resolves a version constraint against the supplied candidate list
// and returns the highest candidate that satisfies it.  The candidate list
// is expected to be a flat slice of version strings (any order).
//
// Supported forms:
//
//   - Exact:       "3.14.0"
//   - Partial:     "3.14" → highest 3.14.x; "3" → highest 3.x.y
//   - Aliases:     "latest", "latest-stable", "stable", "latest-installed"
//   - Ranges:      "~3.14.0", "~3.14", "^3.14.0", ">=3.12.0,<4.0.0",
//     ">3.14.0", ">=3.14.0", "<4.0.0", "<=3.14.0", "=3.14.0"
//
// Partial and bare-version constraints never match pre-release candidates
// (consistent with the usual semver / npm behaviour).  Explicit range
// constraints that pin a specific prerelease (e.g. ">=3.14.0-rc.1") *do*
// consider matching pre-releases.
//
// If no candidate satisfies the constraint, Match returns an error.
func Match(constraint string, candidates []string) (string, error) {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" {
		return "", fmt.Errorf("empty version constraint")
	}

	// Aliases first — they ignore everything else.
	switch strings.ToLower(constraint) {
	case AliasLatest, AliasLatestStable, AliasStable:
		return pickHighestStable(candidates)
	case AliasLatestInstalled:
		return pickHighest(candidates)
	}

	// Build a predicate that each candidate must satisfy.
	pred, allowPre, err := compileConstraint(constraint)
	if err != nil {
		return "", err
	}

	// Collect matches then pick the newest.
	var matches []Version
	for _, c := range candidates {
		v := Parse(c)
		if v.Original == "" || (v.Major == 0 && v.Minor == 0 && v.Patch == 0 && v.PreRelease == "" && c != "0.0.0" && c != "v0.0.0") {
			continue
		}
		if !allowPre && v.PreRelease != "" {
			continue
		}
		if pred(v) {
			matches = append(matches, v)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no version matches constraint %q", constraint)
	}
	// Pick the newest.
	best := matches[0]
	for _, v := range matches[1:] {
		if Less(best, v) {
			best = v
		}
	}
	return best.Original, nil
}

// pickHighestStable returns the highest non-prerelease candidate.
func pickHighestStable(candidates []string) (string, error) {
	var best Version
	found := false
	for _, c := range candidates {
		v := Parse(c)
		if v.Original == "" {
			continue
		}
		if !isParsed(v, c) {
			continue
		}
		if v.PreRelease != "" {
			continue
		}
		if !found || Less(best, v) {
			best = v
			found = true
		}
	}
	if !found {
		return "", fmt.Errorf("no stable versions available")
	}
	return best.Original, nil
}

// pickHighest returns the highest candidate overall (prerelease allowed).
func pickHighest(candidates []string) (string, error) {
	var best Version
	found := false
	for _, c := range candidates {
		v := Parse(c)
		if !isParsed(v, c) {
			continue
		}
		if !found || Less(best, v) {
			best = v
			found = true
		}
	}
	if !found {
		return "", fmt.Errorf("no versions available")
	}
	return best.Original, nil
}

// isParsed reports whether Parse successfully turned the input into a
// semver.  Parse falls back to a zero Version when parsing fails, so we
// detect that specifically.
func isParsed(v Version, raw string) bool {
	trimmed := strings.TrimPrefix(strings.TrimSpace(raw), "v")
	core := strings.SplitN(trimmed, "-", 2)[0]
	// If the core isn't three numeric segments, Parse zeros Major/Minor/Patch.
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if _, err := strconv.Atoi(p); err != nil {
			return false
		}
	}
	_ = v
	return true
}

// predicate reports whether a candidate version satisfies the constraint.
type predicate func(Version) bool

// compileConstraint parses a constraint string and returns a predicate.
// allowPre reports whether pre-release candidates may be considered.  It is
// enabled when the constraint explicitly pins a pre-release suffix.
func compileConstraint(constraint string) (pred predicate, allowPre bool, err error) {
	// AND of comma-separated sub-constraints.
	parts := strings.Split(constraint, ",")
	preds := make([]predicate, 0, len(parts))
	for _, raw := range parts {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil, false, fmt.Errorf("empty sub-constraint in %q", constraint)
		}
		p, allowPreOne, err := compileSingle(raw)
		if err != nil {
			return nil, false, err
		}
		if allowPreOne {
			allowPre = true
		}
		preds = append(preds, p)
	}
	combined := func(v Version) bool {
		for _, p := range preds {
			if !p(v) {
				return false
			}
		}
		return true
	}
	return combined, allowPre, nil
}

// compileSingle parses a single constraint token (no commas).
func compileSingle(raw string) (predicate, bool, error) {
	switch {
	case strings.HasPrefix(raw, "~"):
		return compileTilde(strings.TrimPrefix(raw, "~"))
	case strings.HasPrefix(raw, "^"):
		return compileCaret(strings.TrimPrefix(raw, "^"))
	case strings.HasPrefix(raw, ">="):
		return compileCompare(strings.TrimPrefix(raw, ">="), ">=")
	case strings.HasPrefix(raw, "<="):
		return compileCompare(strings.TrimPrefix(raw, "<="), "<=")
	case strings.HasPrefix(raw, ">"):
		return compileCompare(strings.TrimPrefix(raw, ">"), ">")
	case strings.HasPrefix(raw, "<"):
		return compileCompare(strings.TrimPrefix(raw, "<"), "<")
	case strings.HasPrefix(raw, "="):
		return compileCompare(strings.TrimPrefix(raw, "="), "=")
	}
	// Bare: either exact (M.m.p[-pre]), partial (M, M.m).
	return compileBare(raw)
}

// parsePartial parses "M", "M.m", or "M.m.p[-pre]" into a Version and a
// flag indicating how many segments were specified (1, 2, or 3).  Returns
// an error on any malformed input.
func parsePartial(raw string) (v Version, specifiedSegments int, err error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "v")
	if raw == "" {
		return Version{}, 0, fmt.Errorf("empty version")
	}
	// Separate optional prerelease.
	preRelease := ""
	if idx := strings.Index(raw, "-"); idx >= 0 {
		preRelease = raw[idx+1:]
		raw = raw[:idx]
	}
	parts := strings.Split(raw, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return Version{}, 0, fmt.Errorf("invalid version %q", raw)
	}
	nums := make([]int, 3)
	for i, p := range parts {
		if p == "" {
			return Version{}, 0, fmt.Errorf("invalid version segment in %q", raw)
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return Version{}, 0, fmt.Errorf("invalid version segment %q: %w", p, err)
		}
		nums[i] = n
	}
	v = Version{Major: nums[0], Minor: nums[1], Patch: nums[2], PreRelease: preRelease, Original: raw}
	return v, len(parts), nil
}

func compileBare(raw string) (predicate, bool, error) {
	v, segs, err := parsePartial(raw)
	if err != nil {
		return nil, false, err
	}
	switch segs {
	case 3:
		// Exact match — including prerelease.
		allowPre := v.PreRelease != ""
		pred := func(c Version) bool {
			return c.Major == v.Major && c.Minor == v.Minor && c.Patch == v.Patch && c.PreRelease == v.PreRelease
		}
		return pred, allowPre, nil
	case 2:
		// Match any M.m.*
		pred := func(c Version) bool {
			return c.Major == v.Major && c.Minor == v.Minor && c.PreRelease == ""
		}
		return pred, false, nil
	case 1:
		pred := func(c Version) bool {
			return c.Major == v.Major && c.PreRelease == ""
		}
		return pred, false, nil
	}
	return nil, false, fmt.Errorf("invalid version %q", raw)
}

func compileTilde(raw string) (predicate, bool, error) {
	v, segs, err := parsePartial(raw)
	if err != nil {
		return nil, false, fmt.Errorf("invalid tilde constraint ~%s: %w", raw, err)
	}
	// ~M.m or ~M.m.p → >=M.m.(p|0), <M.(m+1).0
	// ~M → >=M.0.0, <(M+1).0.0  (same as caret for a major-only partial)
	allowPre := v.PreRelease != ""
	lower := Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch, PreRelease: v.PreRelease}
	var upper Version
	switch segs {
	case 1:
		upper = Version{Major: v.Major + 1, Minor: 0, Patch: 0}
	default:
		upper = Version{Major: v.Major, Minor: v.Minor + 1, Patch: 0}
	}
	pred := func(c Version) bool {
		// c >= lower AND c < upper.
		if Less(c, lower) {
			return false
		}
		if !Less(c, upper) {
			return false
		}
		return true
	}
	return pred, allowPre, nil
}

func compileCaret(raw string) (predicate, bool, error) {
	v, _, err := parsePartial(raw)
	if err != nil {
		return nil, false, fmt.Errorf("invalid caret constraint ^%s: %w", raw, err)
	}
	// ^M.m.p → >=M.m.p, <(M+1).0.0  (for M >= 1 — we don't do 0.x special-casing)
	// This matches the task's example ^3.14.0 → >=3.14.0, <4.0.0.
	allowPre := v.PreRelease != ""
	lower := v
	upper := Version{Major: v.Major + 1, Minor: 0, Patch: 0}
	pred := func(c Version) bool {
		if Less(c, lower) {
			return false
		}
		if !Less(c, upper) {
			return false
		}
		return true
	}
	return pred, allowPre, nil
}

func compileCompare(raw, op string) (predicate, bool, error) {
	v, segs, err := parsePartial(raw)
	if err != nil {
		return nil, false, fmt.Errorf("invalid comparator %s%s: %w", op, raw, err)
	}
	// For partial comparators like ">=3.14", treat missing segments as 0.
	_ = segs
	allowPre := v.PreRelease != ""
	switch op {
	case ">":
		return func(c Version) bool { return Less(v, c) }, allowPre, nil
	case ">=":
		return func(c Version) bool { return !Less(c, v) }, allowPre, nil
	case "<":
		return func(c Version) bool { return Less(c, v) }, allowPre, nil
	case "<=":
		return func(c Version) bool { return !Less(v, c) }, allowPre, nil
	case "=":
		return func(c Version) bool {
			return c.Major == v.Major && c.Minor == v.Minor && c.Patch == v.Patch && c.PreRelease == v.PreRelease
		}, allowPre, nil
	}
	return nil, false, fmt.Errorf("unsupported operator %q", op)
}
