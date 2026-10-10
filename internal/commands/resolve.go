package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/user/helm-env/internal/config"
	"github.com/user/helm-env/internal/github"
)

// ResolveHelp prints help for the resolve command.
func ResolveHelp() {
	fmt.Println(`Usage: helm-env resolve [<spec>] [flags]

Resolve a helm version spec to a concrete installed version and print only
that version on stdout.

<spec> may be an exact version (3.14.0), a partial version (3.14), an alias
(latest, stable, latest-installed) or a range (~3.14, ^3, >=3.12,<3.15).
When omitted, the active spec is used (shell > local > global).

Installed versions are matched first. When nothing installed matches and
--install (or HELMENV_AUTO_INSTALL) is set, the best remote match is
installed and printed. Progress notices go to stderr.

Flags:
  --install      Install the best remote match when nothing installed matches.
  -s, --silent   Suppress auto-install notices on stderr.
  -h, --help     Show this help message.

Environment:
  HELMENV_AUTO_INSTALL  When truthy (1, true, yes, on), enables --install.

Exits 1 when the spec cannot be resolved.`)
}

// Resolve prints the concrete installed helm version for spec (or for the
// active spec when spec is empty).  When nothing installed matches and
// install is true or HELMENV_AUTO_INSTALL is truthy, the best remote match is
// installed first.  silent suppresses the auto-install notice on stderr.
//
// The legacy --concrete flag is accepted by the dispatcher as a no-op: the
// output is always a concrete version, and existing on-disk shims still
// pass it.
func Resolve(spec string, install, silent bool) error {
	return resolveWithClient(github.NewClient(), spec, install, silent)
}

func resolveWithClient(client *github.Client, spec string, install, silent bool) error {
	if err := config.RequireInit(); err != nil {
		return err
	}
	mode := AutoFromEnv()
	if install {
		mode = AutoForce
	}
	var stderr io.Writer = os.Stderr
	if silent {
		stderr = io.Discard
	}
	v, err := autoInstallSpecWith(spec, mode, client, stderr)
	if err != nil {
		return err
	}
	fmt.Println(v)
	return nil
}
