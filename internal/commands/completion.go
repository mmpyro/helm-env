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
//
// The deprecated `autocompletion` alias is intentionally absent: it is still
// dispatched by main.go but hidden from help and completion lists.
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
	"completion",
	"status",
	"resolve",
	"prune",
	"doctor",
	"exec",
}

// supportedShells is the ordered list of shells for which helm-env can emit a
// completion script. Used for help text, error messages and the shell-name
// completions offered after `helm-env completion`.
var supportedShells = []string{"bash", "zsh", "fish", "powershell"}

// AutocompletionDeprecation is printed to stderr when the deprecated
// `autocompletion` alias is used instead of `completion`.
const AutocompletionDeprecation = "helm-env: 'autocompletion' is deprecated; use 'helm-env completion'"

// versionCompletingSubcommands are the subcommands that accept an installed
// helm version as an argument. Their completion is populated by invoking
// `helm-env list` at completion time.
//
// `exec` and `resolve` are intentionally omitted here: they have their own
// completion cases that combine installed versions with their flags.
var versionCompletingSubcommands = []string{"uninstall", "shell", "local", "global"}

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

// ParseCompletionArgs parses the arguments of `helm-env completion` (and the
// deprecated `autocompletion` alias): an optional positional SHELL and/or
// `--shell SHELL` / `--shell=SHELL`. help reports whether -h/--help was
// seen, in which case the caller must print CompletionHelp and stop.
func ParseCompletionArgs(args []string) (shell string, help bool, err error) {
	set := func(v string) error {
		if shell != "" && !strings.EqualFold(shell, v) {
			return fmt.Errorf("conflicting shells %q and %q", shell, v)
		}
		shell = v
		return nil
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return "", true, nil
		case arg == "--shell":
			if i+1 >= len(args) {
				return "", false, fmt.Errorf("--shell requires a value")
			}
			i++
			if err := set(args[i]); err != nil {
				return "", false, err
			}
		case strings.HasPrefix(arg, "--shell="):
			if err := set(strings.TrimPrefix(arg, "--shell=")); err != nil {
				return "", false, err
			}
		case strings.HasPrefix(arg, "-"):
			return "", false, fmt.Errorf("unknown flag %q for completion", arg)
		default:
			if err := set(arg); err != nil {
				return "", false, err
			}
		}
	}
	return shell, false, nil
}

// Completion prints a shell completion script for helm-env to stdout.
//
// When shell is empty, Completion detects the current shell from $SHELL
// (falling back to bash) and writes a short notice to stderr so the user still
// sees which script was generated when the output is piped to `source`.
//
// When shell is explicitly set it must be one of "bash", "zsh", "fish",
// "powershell" or the "pwsh" alias; any other value returns an error.
func Completion(shell string) error {
	autoDetected := false
	if shell == "" {
		autoDetected = true
		shell = DetectShell()
		if shell == "" {
			shell = "bash"
		}
	}

	shell = strings.ToLower(shell)
	if shell == "pwsh" {
		shell = "powershell"
	}

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
		return fmt.Errorf("unsupported shell %q: supported shells are %s (or pwsh)", shell, strings.Join(supportedShells, ", "))
	}
}

// CompletionHelp prints help for the completion command.
func CompletionHelp() {
	fmt.Printf(`Usage: helm-env completion [SHELL] [--shell SHELL]

Generate a shell completion script for helm-env.

SHELL may be one of: %s (pwsh is accepted as an alias for
powershell). It can be given positionally or with --shell.

When SHELL is omitted, helm-env detects your current shell from the $SHELL
environment variable (falling back to bash). PowerShell is never auto-detected
and must always be requested explicitly.

Flags:
  --shell SHELL  Shell to generate the script for.
  -h, --help     Show this help message.

Examples:
  # bash (one-shot, current session):
  source <(helm-env completion bash)

  # zsh:
  eval "$(helm-env completion zsh)"

  # fish:
  helm-env completion fish | source

  # powershell (save to profile):
  helm-env completion powershell >> $PROFILE
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
	shells := completionShellNames()

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
        exec)
            # exec takes an installed version plus optional --auto / --no-auto
            local versions
            versions=$(helm-env list 2>/dev/null)
            COMPREPLY=( $(compgen -W "${versions} --auto --no-auto -h --help" -- "${cur}") )
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
        which)
            COMPREPLY=( $(compgen -W "--explain -h --help" -- "${cur}") )
            return 0
            ;;
        prune)
            COMPREPLY=( $(compgen -W "--keep-last --older-than --dry-run --yes -h --help" -- "${cur}") )
            return 0
            ;;
        resolve)
            # resolve takes an optional version spec plus its flags
            local versions
            versions=$(helm-env list 2>/dev/null)
            COMPREPLY=( $(compgen -W "${versions} --install -s --silent -h --help" -- "${cur}") )
            return 0
            ;;
        completion|--shell)
            COMPREPLY=( $(compgen -W "` + shells + ` --shell -h --help" -- "${cur}") )
            return 0
            ;;
        doctor)
            COMPREPLY=( $(compgen -W "--fix -h --help" -- "${cur}") )
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
	shells := completionShellNames()
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
        uninstall|shell|local|global)
          _helm_env_list_versions
          ;;
        exec)
          _helm_env_list_versions
          _arguments \
            '--auto[force auto-install of missing versions]' \
            '--no-auto[disable auto-install even when HELMENV_AUTO_INSTALL is set]' \
            '(-h --help)'{-h,--help}'[show help]'
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
        which)
          _arguments \
            '--explain[print the resolution trace]' \
            '(-h --help)'{-h,--help}'[show help]'
          ;;
        prune)
          _arguments \
            '--keep-last[keep the N newest installed versions]:count:' \
            '--older-than[prune versions older than DURATION (e.g. 180d, 24h)]:duration:' \
            '--dry-run[print what would be pruned without removing (default)]' \
            '--yes[actually remove the selected versions]' \
            '(-h --help)'{-h,--help}'[show help]'
          ;;
        resolve)
          _helm_env_list_versions
          _arguments \
            '--install[install the best remote match when nothing installed matches]' \
            '(-s --silent)'{-s,--silent}'[suppress auto-install notices]' \
            '(-h --help)'{-h,--help}'[show help]'
          ;;
        completion)
          _arguments \
            '1:shell:(` + shells + `)' \
            '--shell[shell to generate the script for]:shell:(` + shells + `)' \
            '(-h --help)'{-h,--help}'[show help]'
          ;;
        doctor)
          _arguments \
            '--fix[regenerate the shim and repair what can be repaired]' \
            '(-h --help)'{-h,--help}'[show help]'
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

	// exec takes installed versions AND --auto / --no-auto flags.
	b.WriteString("# exec: installed versions + auto-install flags\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from exec' -a '(__helm_env_installed_versions)'\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from exec' -l auto -d 'Force auto-install of missing versions'\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from exec' -l no-auto -d 'Disable auto-install'\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from exec' -s h -l help -d 'Show help'\n\n")

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
		"complete -c helm-env -f -n '__fish_seen_subcommand_from %s' -l help -d 'Show help'\n\n",
		strings.Join(prereleaseSubcommands, " "),
	)

	// which flags.
	b.WriteString("# which flags\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from which' -l explain -d 'Print the resolution trace'\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from which' -s h -l help -d 'Show help'\n\n")

	// prune flags.
	b.WriteString("# prune flags\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from prune' -l keep-last -d 'Keep the N newest installed versions'\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from prune' -l older-than -d 'Prune versions older than DURATION'\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from prune' -l dry-run -d 'Print what would be pruned without removing (default)'\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from prune' -l yes -d 'Actually remove the selected versions'\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from prune' -s h -l help -d 'Show help'\n\n")

	// resolve flags.
	b.WriteString("# resolve flags\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from resolve' -a '(__helm_env_installed_versions)'\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from resolve' -l install -d 'Install the best remote match when nothing installed matches'\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from resolve' -s s -l silent -d 'Suppress auto-install notices'\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from resolve' -s h -l help -d 'Show help'\n\n")

	// completion: shell names + flags.
	b.WriteString("# completion shells and flags\n")
	fmt.Fprintf(&b,
		"complete -c helm-env -f -n '__fish_seen_subcommand_from completion' -a %q\n",
		completionShellNames(),
	)
	fmt.Fprintf(&b,
		"complete -c helm-env -f -n '__fish_seen_subcommand_from completion' -l shell -x -a %q -d 'Shell to generate the script for'\n",
		completionShellNames(),
	)
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from completion' -s h -l help -d 'Show help'\n\n")

	// doctor flags.
	b.WriteString("# doctor flags\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from doctor' -l fix -d 'Regenerate the shim and repair what can be repaired'\n")
	b.WriteString("complete -c helm-env -f -n '__fish_seen_subcommand_from doctor' -s h -l help -d 'Show help'\n")

	_, err := fmt.Fprint(w, b.String())
	return err
}

// printPowershellCompletion writes a PowerShell completion script for
// helm-env using Register-ArgumentCompleter.
func printPowershellCompletion(w io.Writer) error {
	cmds := `'` + strings.Join(subcommands, `','`) + `'`
	versionCmds := `'` + strings.Join(versionCompletingSubcommands, `','`) + `'`
	prereleaseCmds := `'` + strings.Join(prereleaseSubcommands, `','`) + `'`
	shells := `'` + strings.Join(strings.Fields(completionShellNames()), `','`) + `'`

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

    if ($subcommand -eq 'exec') {
        # exec takes an installed version plus --auto / --no-auto.
        try {
            $versions = & helm-env list 2>$null
        } catch {
            $versions = @()
        }
        $candidates = @($versions) + @('--auto', '--no-auto', '-h', '--help')
        $candidates | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
        return
    }

    if ($subcommand -eq 'which') {
        @('--explain', '-h', '--help') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterName', $_)
        }
        return
    }

    if ($subcommand -eq 'prune') {
        @('--keep-last', '--older-than', '--dry-run', '--yes', '-h', '--help') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterName', $_)
        }
        return
    }

    if ($subcommand -eq 'resolve') {
        # resolve takes an optional version spec plus its flags.
        try {
            $versions = & helm-env list 2>$null
        } catch {
            $versions = @()
        }
        $candidates = @($versions) + @('--install', '-s', '--silent', '-h', '--help')
        $candidates | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
        return
    }

    if ($subcommand -eq 'completion') {
        @(` + shells + `, '--shell', '-h', '--help') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
        return
    }

    if ($subcommand -eq 'doctor') {
        @('--fix', '-h', '--help') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterName', $_)
        }
        return
    }
}
`
	_, err := fmt.Fprint(w, script)
	return err
}

// completionShellNames returns the space-separated shell names offered after
// `helm-env completion`: every supported shell plus the pwsh alias.
func completionShellNames() string {
	return strings.Join(supportedShells, " ") + " pwsh"
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
	case "completion":
		return "Generate shell completion script"
	case "status":
		return "Show current helm-env environment status"
	case "resolve":
		return "Resolve a version constraint to a concrete installed version"
	case "prune":
		return "Remove old installed helm versions"
	case "doctor":
		return "Diagnose helm-env setup and environment"
	case "exec":
		return "Run a command using a specific helm version"
	default:
		return cmd
	}
}
