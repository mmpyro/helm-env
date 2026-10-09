package commands

import (
	"fmt"

	"github.com/user/helm-env/internal/config"
)

// ResolveHelp prints help for the resolve command.
func ResolveHelp() {
	fmt.Println(`Usage: helm-env resolve [flags]

Resolve the active helm version/constraint against installed versions and
print the result to stdout.

Flags:
  --concrete     Print only the concrete installed version (default).
  -h, --help     Show this help message.`)
}

// Resolve prints the concrete installed helm version produced by
// config.ResolveConcreteVersion.  If no version is configured, or the
// configured constraint does not match any installed version, it exits with
// a non-zero status.
//
// The --concrete flag is accepted for forward-compatibility and is currently
// the default (and only) behaviour.  Keeping it in the surface lets the
// shim pin on the semantics even if we later add a non-concrete form.
func Resolve(_ bool) error {
	if err := config.RequireInit(); err != nil {
		return err
	}
	v, err := config.ResolveConcreteVersion()
	if err != nil {
		return err
	}
	fmt.Println(v)
	return nil
}
