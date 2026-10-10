package commands

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStderr captures stderr output from a function call. Analogous to
// captureStdout defined in version_test.go.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// shellMarker is a per-shell string that MUST appear in that shell's
// generated completion script.
type shellMarker struct {
	shell  string
	marker string
}

var shellMarkers = []shellMarker{
	{"bash", "complete -F _helm_env_completions helm-env"},
	{"zsh", "#compdef helm-env"},
	{"fish", "complete -c helm-env"},
	{"powershell", "Register-ArgumentCompleter"},
}

func TestCompletion_AllShellsEmitExpectedMarkers(t *testing.T) {
	for _, sm := range shellMarkers {
		sm := sm
		t.Run(sm.shell, func(t *testing.T) {
			output := captureStdout(t, func() {
				if err := Completion(sm.shell); err != nil {
					t.Fatalf("Completion(%q) returned error: %v", sm.shell, err)
				}
			})
			if !strings.Contains(output, sm.marker) {
				t.Errorf("%s script should contain %q; got:\n%s", sm.shell, sm.marker, output)
			}
		})
	}
}

func TestCompletion_AllShellsListEverySubcommand(t *testing.T) {
	for _, sm := range shellMarkers {
		sm := sm
		t.Run(sm.shell, func(t *testing.T) {
			output := captureStdout(t, func() {
				if err := Completion(sm.shell); err != nil {
					t.Fatalf("Completion(%q) returned error: %v", sm.shell, err)
				}
			})

			for _, cmd := range subcommands {
				if !strings.Contains(output, cmd) {
					t.Errorf("%s script missing subcommand %q", sm.shell, cmd)
				}
			}
		})
	}
}

func TestCompletion_BashBackwardCompatAliasPresent(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Completion("bash"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Original function name must still be the one registered with
	// `complete -F` so previously-sourced scripts keep working.
	if !strings.Contains(output, "complete -F _helm_env_completions helm-env") {
		t.Errorf("bash output must preserve `complete -F _helm_env_completions helm-env`; got:\n%s", output)
	}
	// Alias name is kept available for any external callers that may
	// reference the historical/alternative name.
	if !strings.Contains(output, "_helm_env_complete") {
		t.Errorf("bash output should define a backward-compat `_helm_env_complete` alias; got:\n%s", output)
	}
}

func TestCompletion_ZshUsesArgumentsAndDescribe(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Completion("zsh"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{"_arguments", "_describe", "compdef _helm_env helm-env"} {
		if !strings.Contains(output, want) {
			t.Errorf("zsh script should contain %q; got:\n%s", want, output)
		}
	}
}

func TestCompletion_FishUsesUseSubcommand(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Completion("fish"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "__fish_use_subcommand") {
		t.Errorf("fish script should use __fish_use_subcommand for top-level completions; got:\n%s", output)
	}
	if !strings.Contains(output, "__fish_seen_subcommand_from") {
		t.Errorf("fish script should use __fish_seen_subcommand_from for sub-completions; got:\n%s", output)
	}
	if !strings.Contains(output, "__helm_env_installed_versions") {
		t.Errorf("fish script should define __helm_env_installed_versions helper; got:\n%s", output)
	}
}

func TestCompletion_PowershellUsesNativeArgumentCompleter(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Completion("powershell"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{
		"Register-ArgumentCompleter",
		"-CommandName helm-env",
		"-Native",
		"-ScriptBlock",
		"CompletionResult",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("powershell script should contain %q; got:\n%s", want, output)
		}
	}
}

func TestCompletion_SubcommandSpecificCompletions(t *testing.T) {
	t.Run("bash install flags present", func(t *testing.T) {
		output := captureStdout(t, func() { _ = Completion("bash") })
		if !strings.Contains(output, "-s --silent -h --help") {
			t.Errorf("bash install completion should offer flags; got:\n%s", output)
		}
	})

	t.Run("bash list-remote/latest offer --prerelease", func(t *testing.T) {
		output := captureStdout(t, func() { _ = Completion("bash") })
		if !strings.Contains(output, "--prerelease --help") {
			t.Errorf("bash list-remote/latest completion should offer --prerelease and --help; got:\n%s", output)
		}
	})

	t.Run("fish install -s flag present", func(t *testing.T) {
		output := captureStdout(t, func() { _ = Completion("fish") })
		if !strings.Contains(output, "-s silent") && !strings.Contains(output, "-s s -l silent") {
			t.Errorf("fish install completion should offer -s/--silent; got:\n%s", output)
		}
	})
}

func TestCompletion_NewCommandsAndFlagsPresent(t *testing.T) {
	// Flag/token fragments that each shell's script MUST contain for the
	// newly-added commands and flags. Each entry maps shell -> required
	// substrings in that shell's generated script. The matchers are chosen
	// to be loose enough to survive non-semantic formatting tweaks.
	wantPerShell := map[string][]string{
		"bash": {
			// which --explain
			"--explain -h --help",
			// exec --auto / --no-auto
			"--auto --no-auto",
			// prune flags
			"--keep-last --older-than --dry-run --yes -h --help",
			// resolve
			"--install -s --silent -h --help",
			// doctor
			"--fix -h --help",
		},
		"zsh": {
			"--explain[",
			"--auto[",
			"--no-auto[",
			"--keep-last[",
			"--older-than[",
			"--dry-run[",
			"--yes[",
			"--install[",
			"--fix[",
		},
		"fish": {
			"-l explain",
			"-l auto",
			"-l no-auto",
			"-l keep-last",
			"-l older-than",
			"-l dry-run",
			"-l yes",
			"-l install",
			"-l fix",
		},
		"powershell": {
			"'--explain'",
			"'--auto'",
			"'--no-auto'",
			"'--keep-last'",
			"'--older-than'",
			"'--dry-run'",
			"'--yes'",
			"'--install'",
			"'--fix'",
		},
	}

	for shell, wants := range wantPerShell {
		shell := shell
		wants := wants
		t.Run(shell, func(t *testing.T) {
			output := captureStdout(t, func() {
				if err := Completion(shell); err != nil {
					t.Fatalf("Completion(%q) returned error: %v", shell, err)
				}
			})
			for _, w := range wants {
				if !strings.Contains(output, w) {
					t.Errorf("%s script missing %q; got:\n%s", shell, w, output)
				}
			}
		})
	}
}

func TestCompletion_UnknownShellReturnsError(t *testing.T) {
	err := Completion("tcsh")
	if err == nil {
		t.Fatal("expected error for unknown shell, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "tcsh") {
		t.Errorf("error should mention the offending shell; got: %q", msg)
	}
	for _, s := range supportedShells {
		if !strings.Contains(msg, s) {
			t.Errorf("error should list supported shell %q; got: %q", s, msg)
		}
	}
}

func TestCompletion_DefaultDetectsShellAndPrintsNotice(t *testing.T) {
	t.Setenv("SHELL", "/usr/local/bin/fish")

	var stdoutStr, stderrStr string
	stderrStr = captureStderr(t, func() {
		stdoutStr = captureStdout(t, func() {
			if err := Completion(""); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})

	if !strings.Contains(stderrStr, "# helm-env: generating completion for fish") {
		t.Errorf("stderr should announce the detected shell; got: %q", stderrStr)
	}
	if !strings.Contains(stdoutStr, "complete -c helm-env") {
		t.Errorf("stdout should contain fish completion; got:\n%s", stdoutStr)
	}
}

func TestCompletion_DefaultFallsBackToBashWhenShellUnset(t *testing.T) {
	t.Setenv("SHELL", "")

	var stdoutStr, stderrStr string
	stderrStr = captureStderr(t, func() {
		stdoutStr = captureStdout(t, func() {
			if err := Completion(""); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})

	if !strings.Contains(stderrStr, "# helm-env: generating completion for bash") {
		t.Errorf("stderr should announce bash fallback; got: %q", stderrStr)
	}
	if !strings.Contains(stdoutStr, "complete -F _helm_env_completions helm-env") {
		t.Errorf("stdout should contain bash completion; got:\n%s", stdoutStr)
	}
}

func TestCompletion_ExplicitShellSuppressesStderrNotice(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")

	stderrStr := captureStderr(t, func() {
		_ = captureStdout(t, func() {
			if err := Completion("zsh"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})

	if stderrStr != "" {
		t.Errorf("explicit shell should not print a notice to stderr; got: %q", stderrStr)
	}
}

func TestDetectShell(t *testing.T) {
	cases := []struct {
		name   string
		env    string
		expect string
	}{
		{"zsh", "/bin/zsh", "zsh"},
		{"fish", "/usr/local/bin/fish", "fish"},
		{"bash", "/bin/bash", "bash"},
		{"empty", "", ""},
		{"unknown-shell", "/usr/bin/tcsh", ""},
		{"powershell-never-detected", "/opt/pwsh/pwsh", ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SHELL", tc.env)
			got := DetectShell()
			if got != tc.expect {
				t.Errorf("DetectShell() with SHELL=%q = %q; want %q", tc.env, got, tc.expect)
			}
		})
	}
}

func TestCompletionHelp_MentionsSupportedShells(t *testing.T) {
	output := captureStdout(t, CompletionHelp)

	if !strings.Contains(output, "helm-env completion [SHELL] [--shell SHELL]") {
		t.Errorf("help should document the new syntax; got:\n%s", output)
	}
	for _, s := range supportedShells {
		if !strings.Contains(output, s) {
			t.Errorf("help should mention supported shell %q; got:\n%s", s, output)
		}
	}
}

func TestCompletionHelp_ShortCircuitsFromDispatcher(t *testing.T) {
	// Simulates the main.go dispatcher's -h / --help short-circuit:
	// when the user passes -h or --help, CompletionHelp is called
	// and Completion() is NEVER invoked, so no shell script should
	// land on stdout.
	for _, flag := range []string{"-h", "--help"} {
		flag := flag
		t.Run(flag, func(t *testing.T) {
			output := captureStdout(t, CompletionHelp)

			for _, sm := range shellMarkers {
				if strings.Contains(output, sm.marker) {
					t.Errorf("help output should not contain shell marker %q from %s", sm.marker, sm.shell)
				}
			}
			// And it should still contain the help preamble.
			if !strings.Contains(output, "Usage: helm-env completion") {
				t.Errorf("help output missing usage preamble when invoked for %q; got:\n%s", flag, output)
			}
		})
	}
}

func TestCompletion_EveryShellCompletesShellNames(t *testing.T) {
	for _, sm := range shellMarkers {
		t.Run(sm.shell, func(t *testing.T) {
			output := captureStdout(t, func() {
				if err := Completion(sm.shell); err != nil {
					t.Fatalf("Completion(%q) returned error: %v", sm.shell, err)
				}
			})
			if !strings.Contains(output, "completion") {
				t.Errorf("%s script should complete the completion subcommand", sm.shell)
			}
			shellFlag := "--shell"
			if sm.shell == "fish" {
				shellFlag = "-l shell"
			}
			for _, name := range append(append([]string{}, supportedShells...), "pwsh", shellFlag) {
				if !strings.Contains(output, name) {
					t.Errorf("%s script should offer %q after completion", sm.shell, name)
				}
			}
		})
	}
}

func TestCompletion_DeprecatedAliasHidden(t *testing.T) {
	for _, c := range subcommands {
		if c == "autocompletion" {
			t.Fatal("deprecated autocompletion alias must not be in the subcommands list")
		}
	}
	for _, sm := range shellMarkers {
		output := captureStdout(t, func() { _ = Completion(sm.shell) })
		if strings.Contains(output, "autocompletion") {
			t.Errorf("%s script must not offer the deprecated autocompletion alias", sm.shell)
		}
	}
	if AutocompletionDeprecation != "helm-env: 'autocompletion' is deprecated; use 'helm-env completion'" {
		t.Errorf("unexpected deprecation text %q", AutocompletionDeprecation)
	}
}

func TestCompletion_PwshAlias(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Completion("pwsh"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(output, "Register-ArgumentCompleter") {
		t.Errorf("pwsh should emit the powershell script; got:\n%s", output)
	}
}

func TestParseCompletionArgs(t *testing.T) {
	cases := []struct {
		name      string
		args      []string
		wantShell string
		wantHelp  bool
		wantErr   bool
	}{
		{"none", nil, "", false, false},
		{"positional", []string{"zsh"}, "zsh", false, false},
		{"--shell", []string{"--shell", "fish"}, "fish", false, false},
		{"--shell=", []string{"--shell=pwsh"}, "pwsh", false, false},
		{"same twice", []string{"bash", "--shell", "bash"}, "bash", false, false},
		{"conflict", []string{"bash", "--shell", "zsh"}, "", false, true},
		{"missing value", []string{"--shell"}, "", false, true},
		{"unknown flag", []string{"--bogus"}, "", false, true},
		{"-h", []string{"zsh", "-h"}, "", true, false},
		{"--help", []string{"--help"}, "", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			shell, help, err := ParseCompletionArgs(tc.args)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if shell != tc.wantShell || help != tc.wantHelp {
				t.Fatalf("got (%q, %v), want (%q, %v)", shell, help, tc.wantShell, tc.wantHelp)
			}
		})
	}
}
