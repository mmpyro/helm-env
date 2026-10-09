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

func TestAutocompletion_AllShellsEmitExpectedMarkers(t *testing.T) {
	for _, sm := range shellMarkers {
		sm := sm
		t.Run(sm.shell, func(t *testing.T) {
			output := captureStdout(t, func() {
				if err := Autocompletion(sm.shell); err != nil {
					t.Fatalf("Autocompletion(%q) returned error: %v", sm.shell, err)
				}
			})
			if !strings.Contains(output, sm.marker) {
				t.Errorf("%s script should contain %q; got:\n%s", sm.shell, sm.marker, output)
			}
		})
	}
}

func TestAutocompletion_AllShellsListEverySubcommand(t *testing.T) {
	for _, sm := range shellMarkers {
		sm := sm
		t.Run(sm.shell, func(t *testing.T) {
			output := captureStdout(t, func() {
				if err := Autocompletion(sm.shell); err != nil {
					t.Fatalf("Autocompletion(%q) returned error: %v", sm.shell, err)
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

func TestAutocompletion_BashBackwardCompatAliasPresent(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Autocompletion("bash"); err != nil {
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

func TestAutocompletion_ZshUsesArgumentsAndDescribe(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Autocompletion("zsh"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{"_arguments", "_describe", "compdef _helm_env helm-env"} {
		if !strings.Contains(output, want) {
			t.Errorf("zsh script should contain %q; got:\n%s", want, output)
		}
	}
}

func TestAutocompletion_FishUsesUseSubcommand(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Autocompletion("fish"); err != nil {
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

func TestAutocompletion_PowershellUsesNativeArgumentCompleter(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Autocompletion("powershell"); err != nil {
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

func TestAutocompletion_SubcommandSpecificCompletions(t *testing.T) {
	t.Run("bash install flags present", func(t *testing.T) {
		output := captureStdout(t, func() { _ = Autocompletion("bash") })
		if !strings.Contains(output, "-s --silent -h --help") {
			t.Errorf("bash install completion should offer flags; got:\n%s", output)
		}
	})

	t.Run("bash list-remote/latest offer --prerelease", func(t *testing.T) {
		output := captureStdout(t, func() { _ = Autocompletion("bash") })
		if !strings.Contains(output, "--prerelease --help") {
			t.Errorf("bash list-remote/latest completion should offer --prerelease and --help; got:\n%s", output)
		}
	})

	t.Run("fish install -s flag present", func(t *testing.T) {
		output := captureStdout(t, func() { _ = Autocompletion("fish") })
		if !strings.Contains(output, "-s silent") && !strings.Contains(output, "-s s -l silent") {
			t.Errorf("fish install completion should offer -s/--silent; got:\n%s", output)
		}
	})
}

func TestAutocompletion_NewCommandsAndFlagsPresent(t *testing.T) {
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
			"--keep --older-than --dry-run -h --help",
			// resolve
			"--concrete -h --help",
		},
		"zsh": {
			"--explain[",
			"--auto[",
			"--no-auto[",
			"--keep[",
			"--older-than[",
			"--dry-run[",
			"--concrete[",
		},
		"fish": {
			"-l explain",
			"-l auto",
			"-l no-auto",
			"-l keep",
			"-l older-than",
			"-l dry-run",
			"-l concrete",
		},
		"powershell": {
			"'--explain'",
			"'--auto'",
			"'--no-auto'",
			"'--keep'",
			"'--older-than'",
			"'--dry-run'",
			"'--concrete'",
		},
	}

	for shell, wants := range wantPerShell {
		shell := shell
		wants := wants
		t.Run(shell, func(t *testing.T) {
			output := captureStdout(t, func() {
				if err := Autocompletion(shell); err != nil {
					t.Fatalf("Autocompletion(%q) returned error: %v", shell, err)
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

func TestAutocompletion_UnknownShellReturnsError(t *testing.T) {
	err := Autocompletion("tcsh")
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

func TestAutocompletion_DefaultDetectsShellAndPrintsNotice(t *testing.T) {
	t.Setenv("SHELL", "/usr/local/bin/fish")

	var stdoutStr, stderrStr string
	stderrStr = captureStderr(t, func() {
		stdoutStr = captureStdout(t, func() {
			if err := Autocompletion(""); err != nil {
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

func TestAutocompletion_DefaultFallsBackToBashWhenShellUnset(t *testing.T) {
	t.Setenv("SHELL", "")

	var stdoutStr, stderrStr string
	stderrStr = captureStderr(t, func() {
		stdoutStr = captureStdout(t, func() {
			if err := Autocompletion(""); err != nil {
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

func TestAutocompletion_ExplicitShellSuppressesStderrNotice(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")

	stderrStr := captureStderr(t, func() {
		_ = captureStdout(t, func() {
			if err := Autocompletion("zsh"); err != nil {
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

func TestAutocompletionHelp_MentionsSupportedShells(t *testing.T) {
	output := captureStdout(t, AutocompletionHelp)

	if !strings.Contains(output, "helm-env autocompletion [SHELL]") {
		t.Errorf("help should document the new syntax; got:\n%s", output)
	}
	for _, s := range supportedShells {
		if !strings.Contains(output, s) {
			t.Errorf("help should mention supported shell %q; got:\n%s", s, output)
		}
	}
}

func TestAutocompletionHelp_ShortCircuitsFromDispatcher(t *testing.T) {
	// Simulates the main.go dispatcher's -h / --help short-circuit:
	// when the user passes -h or --help, AutocompletionHelp is called
	// and Autocompletion() is NEVER invoked, so no shell script should
	// land on stdout.
	for _, flag := range []string{"-h", "--help"} {
		flag := flag
		t.Run(flag, func(t *testing.T) {
			output := captureStdout(t, AutocompletionHelp)

			for _, sm := range shellMarkers {
				if strings.Contains(output, sm.marker) {
					t.Errorf("help output should not contain shell marker %q from %s", sm.marker, sm.shell)
				}
			}
			// And it should still contain the help preamble.
			if !strings.Contains(output, "Usage: helm-env autocompletion") {
				t.Errorf("help output missing usage preamble when invoked for %q; got:\n%s", flag, output)
			}
		})
	}
}
