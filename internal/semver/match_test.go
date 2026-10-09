package semver

import "testing"

func TestIsAlias(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"latest", true},
		{"LATEST", true},
		{"latest-stable", true},
		{"stable", true},
		{"latest-installed", true},
		{"3.14.0", false},
		{"", false},
		{"~3.14", false},
	}
	for _, tt := range tests {
		if got := IsAlias(tt.in); got != tt.want {
			t.Errorf("IsAlias(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestIsExact(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"3.14.0", true},
		{"v3.14.0", true},
		{"3.14.0-rc.1", true},
		{"3.14", false},
		{"3", false},
		{"~3.14.0", false},
		{"^3.14.0", false},
		{">=3.14.0", false},
		{"latest", false},
		{"", false},
		{"abc", false},
	}
	for _, tt := range tests {
		if got := IsExact(tt.in); got != tt.want {
			t.Errorf("IsExact(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestMatch_Exact(t *testing.T) {
	candidates := []string{"3.14.0", "3.14.1", "3.15.0"}
	got, err := Match("3.14.0", candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "3.14.0" {
		t.Fatalf("got %q, want %q", got, "3.14.0")
	}
}

func TestMatch_Partial(t *testing.T) {
	candidates := []string{
		"3.14.0", "3.14.1", "3.14.2",
		"3.15.0", "3.15.1",
		"4.0.0",
	}
	tests := []struct {
		constraint string
		want       string
	}{
		{"3.14", "3.14.2"},
		{"3.15", "3.15.1"},
		{"3", "3.15.1"},
		{"4", "4.0.0"},
	}
	for _, tt := range tests {
		got, err := Match(tt.constraint, candidates)
		if err != nil {
			t.Errorf("Match(%q) returned error: %v", tt.constraint, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Match(%q) = %q, want %q", tt.constraint, got, tt.want)
		}
	}
}

func TestMatch_PartialSkipsPrereleases(t *testing.T) {
	candidates := []string{"3.14.0", "3.14.1-rc.1", "3.14.2-alpha.1"}
	got, err := Match("3.14", candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "3.14.0" {
		t.Fatalf("got %q, want 3.14.0 (should skip prereleases)", got)
	}
}

func TestMatch_Aliases(t *testing.T) {
	candidates := []string{"3.14.0", "3.15.0", "3.16.0-alpha.1"}

	for _, alias := range []string{"latest", "latest-stable", "stable", "LATEST"} {
		got, err := Match(alias, candidates)
		if err != nil {
			t.Fatalf("Match(%q) error: %v", alias, err)
		}
		if got != "3.15.0" {
			t.Fatalf("Match(%q) = %q, want %q", alias, got, "3.15.0")
		}
	}

	got, err := Match("latest-installed", candidates)
	if err != nil {
		t.Fatalf("Match(latest-installed) error: %v", err)
	}
	if got != "3.16.0-alpha.1" {
		t.Fatalf("Match(latest-installed) = %q, want 3.16.0-alpha.1", got)
	}
}

func TestMatch_Tilde(t *testing.T) {
	candidates := []string{"3.13.9", "3.14.0", "3.14.5", "3.14.9", "3.15.0", "4.0.0"}

	tests := []struct {
		constraint string
		want       string
	}{
		{"~3.14.0", "3.14.9"},
		{"~3.14", "3.14.9"},
		{"~3.14.5", "3.14.9"},
	}
	for _, tt := range tests {
		got, err := Match(tt.constraint, candidates)
		if err != nil {
			t.Errorf("Match(%q) error: %v", tt.constraint, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Match(%q) = %q, want %q", tt.constraint, got, tt.want)
		}
	}
}

func TestMatch_Caret(t *testing.T) {
	candidates := []string{"3.14.0", "3.15.0", "3.20.1", "4.0.0", "4.1.0"}
	got, err := Match("^3.14.0", candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "3.20.1" {
		t.Fatalf("Match(^3.14.0) = %q, want 3.20.1", got)
	}
}

func TestMatch_Comparators(t *testing.T) {
	candidates := []string{"3.12.0", "3.13.0", "3.14.0", "3.15.0", "4.0.0", "4.1.0"}

	tests := []struct {
		constraint string
		want       string
	}{
		{">3.14.0", "4.1.0"},
		{">=3.14.0", "4.1.0"},
		{"<4.0.0", "3.15.0"},
		{"<=3.14.0", "3.14.0"},
		{"=3.14.0", "3.14.0"},
	}
	for _, tt := range tests {
		got, err := Match(tt.constraint, candidates)
		if err != nil {
			t.Errorf("Match(%q) error: %v", tt.constraint, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Match(%q) = %q, want %q", tt.constraint, got, tt.want)
		}
	}
}

func TestMatch_CombinedConstraints(t *testing.T) {
	candidates := []string{"3.11.0", "3.12.0", "3.13.0", "3.14.0", "3.15.0", "4.0.0"}
	got, err := Match(">=3.12.0,<4.0.0", candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "3.15.0" {
		t.Fatalf("Match(>=3.12.0,<4.0.0) = %q, want 3.15.0", got)
	}
}

func TestMatch_PrereleaseConstraint(t *testing.T) {
	candidates := []string{"3.14.0", "3.14.1-rc.1", "3.14.2-rc.1"}

	// Explicit prerelease constraint should allow prereleases.
	got, err := Match(">=3.14.1-rc.1", candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "3.14.2-rc.1" {
		t.Fatalf("got %q, want 3.14.2-rc.1", got)
	}

	// Exact prerelease pin.
	got, err = Match("3.14.1-rc.1", candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "3.14.1-rc.1" {
		t.Fatalf("got %q, want 3.14.1-rc.1", got)
	}
}

func TestMatch_NoMatch(t *testing.T) {
	candidates := []string{"3.14.0"}
	_, err := Match("4.0.0", candidates)
	if err == nil {
		t.Fatal("expected error for no match")
	}
}

func TestMatch_InvalidConstraint(t *testing.T) {
	candidates := []string{"3.14.0"}
	for _, bad := range []string{"", "abc", "~abc", "^", ">=", "3..0", ">=3,,<4"} {
		if _, err := Match(bad, candidates); err == nil {
			t.Errorf("Match(%q) expected error, got nil", bad)
		}
	}
}

func TestMatch_EmptyCandidates(t *testing.T) {
	if _, err := Match("latest", nil); err == nil {
		t.Fatal("expected error for empty candidate list")
	}
	if _, err := Match("3.14", []string{}); err == nil {
		t.Fatal("expected error for empty candidate list")
	}
}
