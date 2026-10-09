// Package commands implements all helm-env CLI commands.
package commands

import "fmt"

// Help prints the help message with all available commands.
func Help() {
	fmt.Println(`Usage: helm-env <command> [arguments]

Commands:
  help            Display this help message and all available commands
  list            List all installed versions of helm cli
  list-remote     List all available versions of helm cli from GitHub
  init            Initialize helm-env setup
  install         Install a specific version (or latest if not specified). Flags: -s, --silent
  uninstall       Uninstall a specific version
  shell           Set or show the shell version of helm cli
  local           Set or show the local version of helm cli
  global          Set or show the global version of helm cli
  latest          Print the latest available version of helm cli from GitHub releases
  which           Print the full path to the active helm binary (--explain for a trace)
  resolve         Resolve the active version against installed versions
  exec            Run a command using a specific helm version (--auto / --no-auto)
  prune           Remove old installed versions (--keep N / --older-than DUR / --dry-run)
  doctor          Run diagnostics against the helm-env installation
  status          Show current helm-env environment status
  upgrade         Upgrade helm-env to the latest version
  autocompletion  Generate shell autocompletion script
  version         Print the version of helm-env`)
}

// InstallHelp prints help for the install command.
func InstallHelp() {
	fmt.Println(`Usage: helm-env install [version] [flags]

Flags:
  -s, --silent    Do not display progress bar or checksum info`)
}
