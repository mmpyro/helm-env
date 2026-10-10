package commands

import (
	"strings"
	"testing"
)

func TestCommandHelps(t *testing.T) {
	cases := []struct {
		name string
		fn   func()
		want []string
	}{
		{"upgrade", UpgradeHelp, []string{"Usage: helm-env upgrade", "latest release", "never triggers an upgrade"}},
		{"uninstall", UninstallHelp, []string{"Usage: helm-env uninstall <version>"}},
		{"shell", ShellHelp, []string{"Usage: helm-env shell [<version>]", "HELMENV_VERSION"}},
		{"local", LocalHelp, []string{"Usage: helm-env local [<version>]", ".helm-version"}},
		{"global", GlobalHelp, []string{"Usage: helm-env global [<version>]", "$HELMENV_ROOT/version"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := captureStdout(t, tc.fn)
			for _, w := range tc.want {
				if !strings.Contains(out, w) {
					t.Errorf("%s help missing %q; got:\n%s", tc.name, w, out)
				}
			}
			if !strings.Contains(out, "-h, --help") {
				t.Errorf("%s help should document -h/--help; got:\n%s", tc.name, out)
			}
		})
	}
}

func TestHelp_ListsCompletionNotAutocompletion(t *testing.T) {
	out := captureStdout(t, Help)
	if !strings.Contains(out, "completion") {
		t.Errorf("help should list completion; got:\n%s", out)
	}
	if strings.Contains(out, "autocompletion") {
		t.Errorf("deprecated autocompletion alias must be hidden from help; got:\n%s", out)
	}
	if !strings.Contains(out, "upgrade") {
		t.Errorf("help should list upgrade; got:\n%s", out)
	}
}
