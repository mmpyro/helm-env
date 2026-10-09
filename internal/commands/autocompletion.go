package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// subcommands is the central list of top-level helm-env commands used by all
// shell completion generators. The order matches the dispatch order in
// cmd/helm-env/main.go. Keep this slice in sync with the dispatch switch.
var subcommands = []string{
	"help",
	"version",
	"init",
	"list",
	"list-remote",
	"latest",
	"install",
	"uninstall",
	"shell",
	"local",
	"global",
	"which",
	"upgrade",
	"autocompletion",
	"status",
	"exec",
}

// supportedShells is the ordered list of shells for which helm-env can emit a
// completion script. Used for help text and error messages.
var supportedShells = []string{"bash", "zsh", "fish", "powershell"}

// versionCompletingSubcommands are the subcommands that accept an installed
// helm version as an argument. Their completion is populated by invoking
// `helm-env list` at completion time.
var versionCompletingSubcommands = []string{"uninstall", "shell", "local", "global", "exec"}

// prereleaseSubcommands are the subcommands that accept the --prerelease flag.
var prereleaseSubcommands = []string{"list-remote", "latest"}

// DetectShell returns the shell name derived from the $SHELL environment
// variable, as one of "bash", "zsh" or "fish". It returns an empty string
// when $SHELL is unset or is a shell helm-env cannot emit completions for.
//
// powershell is intentionally never returned: there is no $SHELL on POSIX for
// PowerShell, so it must always be requested explicitly.
func DetectShell() string {
	raw := os.Getenv("SHELL")
	if raw == "" {
		return ""
	}
	base := filepath.Base(raw)
	switch base {
	case "bash", "zsh", "fish":
		return base
	}
	return ""
}

// Autocompletion prints a shell completion script for helm-env to stdout.
//
// When shell is empty, Autocompletion detects the current shell from $SHELL
// (falling back to bash) and writes a short notice to stderr so the user still
// sees which script was generated when the output is piped to `source`.
//
// When shell is explicitly set it must be one of "bash", "zsh", "fish" or
// "powershell"; any other value returns an error.
func Autocompletion(shell string) error {
	autoDetected := false
	if shell == "" {
		autoDetected = true
		shell = DetectShell()
		if shell == "" {
			shell = "bash"
		}
	}

	shell = strings.ToLower(shell)

	if autoDetected {
		fmt.Fprintf(os.Stderr, "# helm-env: generating completion for %s\n", shell)
	}

	switch shell {
	case "bash":
		return printBashCompletion(os.Stdout)
	case "zsh":
		return printZshCompletion(os.Stdout)
	case "fish":
		return printFishCompletion(os.Stdout)
	case "powershell":
		return printPowershellCompletion(os.Stdout)
	default:
		return fmt.Errorf("unsupported shell %q: supported shells are %s", shell, strings.Join(supportedShells, ", "))
	}
}

// AutocompletionHelp prints help for the autocompletion command.
func AutocompletionHelp() {
	fmt.Printf(`Usage: helm-env autocompletion [SHELL]

Generate a shell completion script for helm-env.

SHELL may be one of: %s.

When SHELL is omitted, helm-env detects your current shell from the $SHELL
environment variable (falling back to bash). PowerShell is never auto-detected
and must always be requested explicitly.

Examples:
  # bash (one-shot, current session):
  source <(helm-env autocompletion bash)

  # zsh:
  eval "$(helm-env autocompletion zsh)"

  # fish:
  helm-env autocompletion fish | source

  # powershell (save to profile):
  helm-env autocompletion powershell >> $PROFILE
`, strings.Join(supportedShells, ", "))
}

// printBashCompletion writes a bash completion script for helm-env.
//
// The function name _helm_env_completions and the `complete -F` registration
// are preserved from the original implementation so previously-sourced
// scripts keep working without regression. _helm_env_complete is registered
// as an alias for backward compatibility with any external callers that may
// reference it.
func printBashCompletion(w io.Writer) error {
	cmds := strings.Join(subcommands, " ")
	versionCmds := joinBashCase(versionCompletingSubcommands)
	prereleaseCmds := joinBashCase(prereleaseSubcommands)

	script := `# bash completion for helm-env
_helm_env_completions() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="` + cmds + `"

    if [[ ${COMP_CWORD} -eq 1 ]] ; then
        COMPREPLY=( $(compgen -W "${opts}" -- "${cur}") )
        return 0
    fi

    case "${prev}" in
        ` + versionCmds + `)
            # Suggest installed versions for these commands
            local versions
            versions=$(helm-env list 2>/dev/null)
            COMPREPLY=( $(compgen -W "${versions}" -- "${cur}") )
            return 0
            ;;
        install)
            COMPREPLY=( $(compgen -W "-s --silent -h --help" -- "${cur}") )
            return 0
            ;;
        ` + prereleaseCmds + `)
            COMPREPLY=( $(compgen -W "--prerelease --help" -- "${cur}") )
            return 0
            ;;
    esac
}
complete -F _helm_env_completions helm-env
# Backward-compatible alias for the completion function.
_helm_env_complete() { _helm_env_completions "$@"; }
`
	_, err := fmt.Fprint(w, script)
	return err
}

// printZshCompletion writes a zsh completion script for helm-env.
func printZshCompletion(w io.Writer) error {
	var describe strings.Builder
	for _, c := range subcommands {
		describe.WriteString("    '")
		describe.WriteString(c)
		describe.WriteString(":")
		describe.WriteString(zshDescription(c))
		describe.WriteString("'\n")
	}

	script := `#compdef helm-env
# zsh completion for helm-env

_helm_env() {
  local -a _helm_env_subcommands
  _helm_env_subcommands=(
` + describe.String() + `  )

  local -a _helm_env_installed_versions
  _helm_env_list_versions() {
    local -a versions
    versions=(${(f)"$(helm-env list 2>/dev/null)"})
    _describe 'installed helm version' versions
  }

  _arguments -C \
    '1: :->command' \
    '*:: :->args'

  case $state in
    command)
      _describe 'helm-env command' _helm_env_subcommands
      ;;
    args)
      case $words[1] in
        uninstall|shell|local|global|exec)
          _helm_env_list_versions
          ;;
        install)
          _arguments \
            '(-s --silent)'{-s,--silent}'[do not display progress bar]' \
            '(-h --help)'{-h,--help}'[show help]'
          ;;
        list-remote|latest)
          _arguments \
            '--prerelease[include pre-release versions]' \
            '--help[show help]'
          ;;
      esac
      ;;
  esac
}

compdef _helm_env helm-env
`
	_, err := fmt.Fprint(w, script)
	return err
}

// printFishCompletion writes a fish completion script for helm-env.
func printFishCompletion(w io.Writer) error {
	var b strings.Builder
	b.WriteString("# fish completion for helm-env\n\n")
	b.WriteString("function __helm_env_installed_versions\n")
	b.WriteString("    helm-env list 2>/dev/null\n")
	b.WriteString("end\n\n")

	// Top-level subcommands. `-f` disables file completion so fish doesn't
	// suggest files alongside subcommand names.
	b.WriteString("# Subcommands\n")
	for _, c := range subcommands {
		fmt.Fprintf(&b,
			"complete -c helm-env -f -n '__fish_use_subcommand' -a %q -d %q\n",
			c, fishDescription(c),
		)
	}
	b.WriteString("\n")

	// Installed-version suggestions.
	b.WriteString("# Installed-version suggestions\n")
	fmt.Fprintf(&b,
		"complete -c helm-env -f -n '__fish_seen_subcommand_from %s' -a '(__helm_env_installed_versions)'\n\n",
		strings.Join(versionCompletingSubcommands, " "),
	)

	// install flags.
	b.WriteString("# install flags\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from install' -s s -l silent -d 'Do not display progress bar'\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from install' -s h -l help -d 'Show help'\n\n")

	// list-remote / latest flags.
	b.WriteString("# list-remote / latest flags\n")
	fmt.Fprintf(&b,
		"complete -c helm-env -f -n '__fish_seen_subcommand_from %s' -l prerelease -d 'Include pre-release versions'\n",
		strings.Join(prereleaseSubcommands, " "),
	)
	fmt.Fprintf(&b,
		"complete -c helm-env -f -n '__fish_seen_subcommand_from %s' -l help -d 'Show help'\n",
		strings.Join(prereleaseSubcommands, " "),
	)

	_, err := fmt.Fprint(w, b.String())
	return err
}

// printPowershellCompletion writes a PowerShell completion script for
// helm-env using Register-ArgumentCompleter.
func printPowershellCompletion(w io.Writer) error {
	cmds := `'` + strings.Join(subcommands, `','`) + `'`
	versionCmds := `'` + strings.Join(versionCompletingSubcommands, `','`) + `'`
	prereleaseCmds := `'` + strings.Join(prereleaseSubcommands, `','`) + `'`

	script := `# PowerShell completion for helm-env
Register-ArgumentCompleter -CommandName helm-env -Native -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    $commands = @(` + cmds + `)
    $versionSubcommands = @(` + versionCmds + `)
    $prereleaseSubcommands = @(` + prereleaseCmds + `)

    $elements = @($commandAst.CommandElements)

    # Completing the first positional (subcommand).
    if ($elements.Count -le 2) {
        $commands | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
        return
    }

    $subcommand = $elements[1].Value

    if ($versionSubcommands -contains $subcommand) {
        try {
            $versions = & helm-env list 2>$null
        } catch {
            $versions = @()
        }
        $versions | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
        return
    }

    if ($subcommand -eq 'install') {
        @('-s', '--silent', '-h', '--help') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterName', $_)
        }
        return
    }

    if ($prereleaseSubcommands -contains $subcommand) {
        @('--prerelease', '--help') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterName', $_)
        }
        return
    }
}
`
	_, err := fmt.Fprint(w, script)
	return err
}

// joinBashCase joins subcommands for a bash case pattern (e.g. "a|b|c").
func joinBashCase(cmds []string) string {
	return strings.Join(cmds, "|")
}

// zshDescription returns a short human-readable description for a subcommand
// suitable for use in a zsh _describe list.
func zshDescription(cmd string) string {
	return subcommandDescription(cmd)
}

// fishDescription returns the same text as zshDescription; kept as its own
// function for symmetry and to make it obvious at call sites which shell is
// consuming the text.
func fishDescription(cmd string) string {
	return subcommandDescription(cmd)
}

// subcommandDescription returns a short, single-line description for a
// subcommand. Kept intentionally terse so the generated completion scripts
// stay small and readable.
func subcommandDescription(cmd string) string {
	switch cmd {
	case "help":
		return "Display help and all available commands"
	case "version":
		return "Print helm-env version"
	case "init":
		return "Initialize helm-env setup"
	case "list":
		return "List installed helm versions"
	case "list-remote":
		return "List available helm versions from GitHub"
	case "latest":
		return "Print the latest available helm version"
	case "install":
		return "Install a helm version"
	case "uninstall":
		return "Uninstall a helm version"
	case "shell":
		return "Set or show the shell helm version"
	case "local":
		return "Set or show the local helm version"
	case "global":
		return "Set or show the global helm version"
	case "which":
		return "Print path to the active helm binary"
	case "upgrade":
		return "Upgrade helm-env to the latest version"
	case "autocompletion":
		return "Generate shell completion script"
	case "status":
		return "Show current helm-env environment status"
	case "exec":
		return "Run a command using a specific helm version"
	default:
		return cmd
	}
}
