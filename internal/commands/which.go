package commands

import (
	"fmt"
	"os"

	"github.com/user/helm-env/internal/config"
)

// WhichHelp prints help for the which command.
func WhichHelp() {
	fmt.Println(`Usage: helm-env which [flags]

Print the full path to the active helm binary that would be used based on
version resolution.

Flags:
  --explain   Also print a trace of the resolution process: which source
              (shell / local / global) won, which parent directories were
              searched for .helm-version, and the final concrete version +
              binary path.
  -h, --help  Show this help message.`)
}

// Which prints the absolute path to the active helm binary.  When explain is
// true, a human-readable trace of how the version was resolved is printed to
// stderr before the final path is printed to stdout.
func Which(explain bool) error {
	if err := config.RequireInit(); err != nil {
		return err
	}

	if !explain {
		version, err := config.ResolveConcreteVersion()
		if err != nil {
			return err
		}
		binaryPath, err := config.GetBinaryPath(version)
		if err != nil {
			return err
		}
		fmt.Println(binaryPath)
		return nil
	}

	version, trace, err := config.ResolveVersionTraced()
	for _, step := range trace {
		marker := " "
		if step.Hit {
			marker = "*"
		}
		fmt.Fprintf(os.Stderr, "[%s] %s: %s\n", marker, step.Source, step.Detail)
	}
	if err != nil {
		return err
	}
	binaryPath, err := config.GetBinaryPath(version)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "resolved: %s\n", version)
	fmt.Fprintf(os.Stderr, "binary:   %s\n", binaryPath)
	fmt.Println(binaryPath)
	return nil
}
