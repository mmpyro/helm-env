# CLI reference

This reference covers every `helm-env` command that the project implements.

!!! tip "Version arguments"
    Every `<version>` argument accepts exact versions, partials, aliases, tildes, carets, and comparator ranges. See [Version strings](#version-strings) for the full table.

Conventions:

- Commands return exit code `0` on success.
- If an error occurs, a command typically prints an error message to stderr and exits with code `1`.
- Every command accepts `-h`/`--help`. This flag prints the help for that command and exits `0`. It does nothing else (for example, `helm-env upgrade --help` never upgrades).
- `helm-env exec` exits with the exit code of the helm process it runs.

## Global usage

```text
helm-env <command> [arguments]
```

To see the top-level help, run `helm-env help`, `helm-env --help`, or `helm-env -h`. To see the help for one command, run `helm-env <command> --help`.

## Environment variables

### `HELMENV_ROOT`

Required. The path where `helm-env` stores installed versions and shims.

Most commands use it, and initialization requires it.

### `HELMENV_VERSION`

Optional. If it is set, it forces `helm-env` to use that `helm` version (highest priority).

Typically, `helm-env shell` sets it after you enable shell integration with `eval "$(helm-env init)"`.

### `HELMENV_AUTO_INSTALL`

Optional. If it is set to a truthy value (`1`, `true`, `yes`, `on`, case-insensitive), `helm-env` *automatically installs* missing helm versions. This happens when the resolver (shim, `exec`, `which`, etc.) finds a version or constraint that is not yet present under `$HELMENV_ROOT/versions/`.

The installation is silent (equivalent to `helm-env install --silent <version>`).
To override it on the command line, see the `--auto` and `--no-auto` flags on `helm-env exec`, and `--install` on `helm-env resolve`.

### `HELMENV_CACHE_TTL`

Optional. How long the cached release list (`$HELMENV_ROOT/cache/releases.json`) stays fresh, as a Go duration (`30m`, `24h`). `0s` disables the cache. Default: `1h`. See [caching strategy](caching.md).

### `SHELL`

Optional. Read by `helm-env completion` to choose a shell when none is given.

## Version strings

Every command that takes a `<version>` argument accepts the forms in this table.
Every version slot that `helm-env` reads (`HELMENV_VERSION`, `.helm-version`, and
`$HELMENV_ROOT/version`) accepts them too.

| Form | Example | Meaning |
|---|---|---|
| Exact | `3.14.0` | A specific release. |
| Partial | `3.14` | Highest `3.14.x` release (prereleases skipped). |
| Partial | `3` | Highest `3.x.y` release (prereleases skipped). |
| Alias | `latest`, `latest-stable`, `stable` | Highest stable release in the candidate set. |
| Alias | `latest-installed` | Highest installed version (prereleases allowed). |
| Tilde | `~3.14.0` | `>=3.14.0, <3.15.0`. |
| Tilde | `~3.14` | `>=3.14.0, <3.15.0`. |
| Caret | `^3.14.0` | `>=3.14.0, <4.0.0`. |
| Range | `>=3.12.0,<4.0.0` | AND-combined comparators. |
| Comparator | `>3.14.0`, `>=3.14.0`, `<4.0.0`, `<=3.14.0`, `=3.14.0` | Single comparator. |

The candidate pool depends on the command:

- `install` and the `latest*` aliases resolve against the **remote** release list (cached in `$HELMENV_ROOT/cache`).
- `uninstall`, `shell`, `local`, `global`, `exec`, `which`, and the shim resolve against **installed** versions on disk.

## Commands

### `help`

Purpose: Print usage and the list of available commands.

Syntax:

```text
helm-env help
helm-env --help
helm-env -h
```

Options/flags: none.

Environment variables: none.

Exit codes:

- `0` always.

Example:

```sh
helm-env help
```

---

### `version`

Purpose: Print the `helm-env` version.

Syntax:

```text
helm-env version
helm-env --version
helm-env -v
```

Options/flags: none.

Environment variables: none.

Exit codes:

- `0` on success.

Example:

```sh
helm-env version
```

---

### `init`

Purpose:

- Create required directories under `HELMENV_ROOT`.
- Generate the `helm` shim under `$HELMENV_ROOT/shims/helm`.
- Print shell initialization code to stdout, for your shell to evaluate.

Syntax:

```text
helm-env init
```

Options/flags: none.

Environment variables:

- `HELMENV_ROOT` (required)

Exit codes:

- `0` on success.
- `1` if `HELMENV_ROOT` is not set or filesystem operations fail.

Example:

```sh
export HELMENV_ROOT="$HOME/.helmenv"
eval "$(helm-env init)"
```

---

### `list`

Purpose: List installed `helm` versions (newest to oldest).

Syntax:

```text
helm-env list
```

Options/flags: none.

Environment variables:

- `HELMENV_ROOT` (required)

Exit codes:

- `0` on success.
- `1` if `helm-env` is not initialized.

Example:

```sh
helm-env list
```

---

### `list-remote`

Purpose: List all available `helm` versions from GitHub releases (newest to oldest).

This command does not require `HELMENV_ROOT` or initialization. If `HELMENV_ROOT` is set, `helm-env` keeps a persistent cache of the results on disk (see [caching strategy](caching.md)).

Syntax:

```text
helm-env list-remote [flags]
```

Options/flags:

- `--prerelease`: include pre-release versions (alpha, beta, rc)
- `-h`, `--help`: show command help and exit

Environment variables: none.

Exit codes:

- `0` on success, or when printing `--help`.
- `1` on GitHub/network errors (including rate limiting).

Example:

```sh
helm-env list-remote --prerelease
```

---

### `latest`

Purpose: Print the latest available `helm` version from GitHub releases.

This command does not require `HELMENV_ROOT` or initialization. If `HELMENV_ROOT` is set, `helm-env` keeps a persistent cache of the results on disk (see [caching strategy](caching.md)).

Syntax:

```text
helm-env latest [flags]
```

Options/flags:

- `--prerelease`: include pre-release versions when selecting the latest
- `-h`, `--help`: show command help and exit

Environment variables: none.

Exit codes:

- `0` on success, or when printing `--help`.
- `1` if no versions are found, or on GitHub/network errors.

Example:

```sh
helm-env latest
```

---

### `install`

Purpose: Download and install an `helm` version into `$HELMENV_ROOT/versions/<version>/helm`.

If you omit `<version>`, `helm-env` installs the latest stable version.

During the download, the command shows a progress bar. It also verifies the integrity of the downloaded file automatically. For this, it uses the SHA256 checksums that are published with the release.

Syntax:

```text
helm-env install [version] [flags]
```

Options/flags:

- `-s`, `--silent`: do not display the progress bar or checksum verification information
- `-h`, `--help`: show command help and exit

Environment variables:

- `HELMENV_ROOT` (required)

Exit codes:

- `0` on success.
- `1` if `helm-env` is not initialized, platform detection fails, the download fails, the checksum does not match, or a filesystem write fails.

Example:

```sh
helm-env install 3.14.0
helm-env install --silent
helm-env install
```

---

### `uninstall`

Purpose: Remove an installed `helm` version directory.

Syntax:

```text
helm-env uninstall <version>
```

Options/flags:

- `-h`, `--help`: show command help and exit.

Environment variables:

- `HELMENV_ROOT` (required)

Exit codes:

- `0` on success.
- `1` if `<version>` is missing, `helm-env` is not initialized, the version is not installed, or the removal from the filesystem fails.

Example:

```sh
helm-env uninstall 3.14.0
```

---

### `shell`

Purpose: Set or show the shell-level `helm` version.

Important: To *set* the version for your current shell session, you must enable shell integration with `eval "$(helm-env init)"`. Without it, you only see the printed `export ...` line, and your current shell does not change.

Syntax:

```text
helm-env shell            # show
helm-env shell <version>  # set
```

Options/flags:

- `-h`, `--help`: show command help and exit.

Environment variables:

- `HELMENV_ROOT` (required)
- `HELMENV_VERSION` (read on show, written when you eval shell integration)

Exit codes:

- `0` on success.
- `1` if not initialized, no shell version is configured (show), or the requested version is not installed (set).

Example:

```sh
eval "$(helm-env init)"
helm-env shell 3.14.0
helm version
```

---

### `local`

Purpose: Set or show the local (directory-level) `helm` version.

When you set a version, `helm-env` writes a `.helm-version` file into the current directory.

Syntax:

```text
helm-env local            # show
helm-env local <version>  # set
```

Options/flags:

- `-h`, `--help`: show command help and exit.

Environment variables:

- `HELMENV_ROOT` (required)

Exit codes:

- `0` on success.
- `1` if not initialized, no local version is configured for this directory (show), the requested version is not installed (set), or writing `.helm-version` fails.

Example:

```sh
helm-env install 3.14.0
helm-env local 3.14.0
helm version
```

---

### `global`

Purpose: Set or show the global default `helm` version.

When you set a version, `helm-env` writes `$HELMENV_ROOT/version`.

Syntax:

```text
helm-env global            # show
helm-env global <version>  # set
```

Options/flags:

- `-h`, `--help`: show command help and exit.

Environment variables:

- `HELMENV_ROOT` (required)

Exit codes:

- `0` on success.
- `1` if not initialized, no global version is configured (show), the requested version is not installed (set), or writing fails.

Example:

```sh
helm-env install 3.14.0
helm-env global 3.14.0
helm version
```

---

### `which`

Purpose: Print the full path to the active `helm` binary. This is the binary that version resolution selects.

Syntax:

```text
helm-env which [flags]
```

Options/flags:

- `--explain`: also print a trace of the resolution process to stderr. The trace shows which source won (shell, local, or global), which parent directories `helm-env` searched for `.helm-version`, and the final concrete version and binary path.
- `-h`, `--help`: show command help and exit.

Environment variables:

- `HELMENV_ROOT` (required)
- `HELMENV_VERSION` (optional, highest priority if set)

Exit codes:

- `0` on success.
- `1` if not initialized or no version is configured.

Example:

```sh
helm-env which
helm-env which --explain
```

---

### `resolve`

Purpose: Resolve a version spec to a concrete helm version and print only that version on stdout.

Syntax:

```text
helm-env resolve [<spec>] [--install] [-s|--silent]
```

- If you give no `<spec>`, the command uses the active spec (shell > local > global).
- `<spec>` accepts every [version string](#version-strings) form: exact, partial, alias or range.
- The command matches installed versions first. If no installed version matches and `--install` (or `HELMENV_AUTO_INSTALL`) is set, it installs the best remote match and prints it. Progress notices go to stderr, so stdout always holds only the version.

Options/flags:

- `--install`: install the best remote match when nothing installed matches.
- `-s`, `--silent`: suppress auto-install notices on stderr.
- `-h`, `--help`: show command help and exit.
- `--concrete`: legacy no-op. The command still accepts it because shims from older releases pass it. The output is always concrete.

Environment variables:

- `HELMENV_ROOT` (required)
- `HELMENV_VERSION` (optional, used when you give no `<spec>`)
- `HELMENV_AUTO_INSTALL` (optional, same as `--install`)

Exit codes:

- `0` when the spec resolves (or is installed with `--install`).
- `1` otherwise (not initialized, no version configured, nothing matches, or the install fails).

Example:

```sh
# .helm-version contains "~3.14"; the installed versions include 3.14.0 and 3.14.5
$ helm-env resolve
3.14.5

# explicit spec, ignoring the configured version
$ helm-env resolve 3.14.0
3.14.0

# install the newest 3.15.x if nothing installed matches
$ helm-env resolve --install --silent ~3.15
3.15.4
```

When the configured value is a constraint and not an installed exact version, the shim calls `helm-env resolve --concrete`.

---

### `upgrade`

Purpose: Upgrade `helm-env` itself. The command gets the latest release of [`mmpyro/helm-env`](https://github.com/mmpyro/helm-env/releases) from the GitHub API. It downloads the binary for the current OS and architecture. Then it replaces the running binary in place.

Syntax:

```text
helm-env upgrade
helm-env upgrade -h|--help
```

Options/flags:

- `-h`, `--help`: show command help and exit. This flag never starts an upgrade.

The command rejects any other argument with exit code `1`.

Environment variables: none. `upgrade` does not need `HELMENV_ROOT` and does not read any `HELMENV_*` variable. The command detects the binary path automatically with `os.Executable()` and follows symlinks.

Exit codes:

- `0` on success, or when already up to date.
- `1` on network or download errors, permission problems, or filesystem errors.

Notes:

- The command detects the current OS and CPU architecture automatically.
- The command writes the new binary atomically (temp file + rename) to prevent corruption.
- If the current version is a development build (`dev`), the upgrade always proceeds.

Example:

```sh
helm-env upgrade
```

---

### `exec`

Purpose: Run a specific version of `helm` for a single command without changing the active version (shell, local, or global).

Syntax:

```text
helm-env exec [--auto|--no-auto] <spec> [--] <helm-args...>
```

`helm-env` passes `<helm-args...>` to the resolved `helm` binary. It removes one leading `--` after `<spec>`, so `helm-env exec 3.14 -- version` runs `helm version`. Flags after `<spec>` belong to helm.

Options/flags:

- `--auto`: enable auto-install for this invocation (installs the resolved version if it is not yet present on disk). Overrides `HELMENV_AUTO_INSTALL`.
- `--no-auto`: disable auto-install for this invocation even if `HELMENV_AUTO_INSTALL` is set.
- `-h`, `--help` (before `<spec>`): show command help and exit.

Version resolution follows the general [version strings](#version-strings) rules. If the value is an exact installed version, `exec` uses it verbatim. Otherwise, `exec` matches the value against the installed candidates. When auto-install is enabled, it matches the value against the remote release list before the installation.

Environment variables:

- `HELMENV_ROOT` (required)
- `HELMENV_VERSION` (set for the subprocess to match the resolved version)
- `HELMENV_AUTO_INSTALL` (optional)

Exit codes:

- The exit code of `helm`, passed through unchanged (for example, `3` if helm exits `3`). `helm-env` prints no extra `exit status N` message.
- `1` if the version is not installed (and auto-install is disabled), you give no command, or initialization fails.

Example:

```sh
helm-env exec 3.14.0 version
helm-env exec ~3.14 -- version --short
HELMENV_AUTO_INSTALL=1 helm-env exec 3.14.0 version
helm-env exec --auto latest version
```

---

### `status`

Purpose: Show a full overview of the current `helm-env` environment.

The output includes:
- The `HELMENV_ROOT` path.
- The active version and the source that it was resolved from.
- Full path to the active `helm` binary.
- A list of all installed versions (the active one is marked with `*`).

Syntax:

```text
helm-env status
```

Options/flags: none.

Environment variables:

- `HELMENV_ROOT` (required)
- `HELMENV_VERSION` (read if set)

Exit codes:

- `0` on success.
- `1` on error.

Example:

```sh
helm-env status
```

---

### `prune`

Purpose: Remove installed `helm` versions that you no longer need.

Syntax:

```text
helm-env prune [--keep-last N] [--older-than DUR] [--dry-run] [--yes]
```

**`prune` is a dry run by default.** It prints the versions that it would remove. It deletes nothing until you pass `--yes`.

Options/flags:

- `--keep-last N`: keep the N newest installed versions.
- `--keep N`: deprecated alias for `--keep-last`.
- `--older-than DUR`: remove installed versions whose helm binary mtime is older than `DUR`. It accepts Go duration strings (for example, `30m`, `24h`) and a `<N>d` suffix for whole days.
- `--dry-run`: print what the command would remove, and do not touch the disk (the default). If you give both `--dry-run` and `--yes`, `--dry-run` wins.
- `--yes`: actually remove the selected versions.
- `-h`, `--help`: show command help and exit.

You must give at least one of `--keep-last` or `--older-than`. If you give both, `prune` keeps a version only if it satisfies **BOTH** rules. `prune` always protects the currently resolved version (shell > local > global, so this includes a `.helm-version` in the current directory). For that version, `prune` prints a warning and skips the removal.

`helm-env prune` selects versions by count and age only. Unlike `istioctl-env` and `vc-env`, it does not scan the disk for version files.

Environment variables:

- `HELMENV_ROOT` (required)

Exit codes:

- `0` on success (including dry-run and no-op).
- `1` on bad flags or when `helm-env` is not initialized.

Example:

```sh
helm-env prune --keep-last 3            # preview
helm-env prune --keep-last 3 --yes      # remove
helm-env prune --older-than 30d --yes
helm-env prune --keep-last 2 --older-than 90d
```

---

### `doctor`

Purpose: Run a set of diagnostic checks against the current `helm-env` installation. Each check prints one line that starts with `[OK]`, `[WARN]`, or `[FAIL]`.

Syntax:

```text
helm-env doctor [--fix]
```

Options/flags:

- `--fix`: before the checks run, create `$HELMENV_ROOT/versions` if it is missing, regenerate the shim, and make installed `helm` binaries executable. Each repair prints a `[FIX]` line. `--fix` never edits `PATH` or shell rc files.
- `-h`, `--help`: show command help and exit.

`helm-env doctor` has no `--deep` flag (unlike `istioctl-env`). `helm-env` verifies installed binaries against SHA256 checksums at install time only.

Checks:

1. `HELMENV_ROOT` is set.
2. `$HELMENV_ROOT` exists and is writable.
3. `$HELMENV_ROOT/versions` exists.
4. `$HELMENV_ROOT/shims/helm` exists, is executable and matches what this `helm-env` would generate (`[FAIL]` on drift, fix with `--fix`).
5. `$HELMENV_ROOT/shims` is on `$PATH` and appears before any other `helm` binary.
6. The cache file (if present) is parseable JSON.
7. Reachability of `https://github.com` and `https://get.helm.sh` (WARN on failure, never fatal).
8. At least one `helm` version is installed (WARN if none).
9. A resolvable active version exists (WARN if not).

Exit codes:

- `0` when no check reports `[FAIL]`.
- `1` when any check reports `[FAIL]`.

Example:

```sh
helm-env doctor
```

---

### `completion`

Purpose: Generate a shell completion script for `helm-env`.

Supported shells: `bash`, `zsh`, `fish`, `powershell` (`pwsh` is accepted as an alias).

The script completes subcommands and the flags of each command. For commands
that take a version (`uninstall`, `shell`, `local`, `global`, `exec`,
`resolve`), it suggests installed versions. After `completion`, it suggests the
shell names and `--shell`.

Syntax:

```text
helm-env completion [SHELL] [--shell SHELL]
```

If you omit `SHELL`, `helm-env` detects your current shell from the
`$SHELL` environment variable (the fallback is `bash`). `helm-env` also writes a
short notice to stderr. Thus, when you pipe the output to `source`, you still see
which script it generated. `helm-env` never detects PowerShell automatically.
You must request it explicitly.

Options/flags:
- `--shell SHELL`: same as the positional `SHELL`
- `-h`, `--help`: show command help and exit

Environment variables:
- `$SHELL` (optional, read for shell auto-detection when you give no shell)

Exit codes:
- `0` on success.
- `1` if an unsupported shell is requested.

Deprecated alias: `helm-env autocompletion` still works with the same
arguments. But it prints `helm-env: 'autocompletion' is deprecated; use 'helm-env completion'`
to stderr. The alias does not appear in help and completion lists. A future
release will remove it.

Examples:

```sh
# bash — current session
source <(helm-env completion bash)

# bash — persistent
echo 'source <(helm-env completion bash)' >> ~/.bashrc
```

```sh
# zsh — current session (simplest)
eval "$(helm-env completion zsh)"

# zsh — persistent via fpath + compinit
mkdir -p ~/.zsh/completions
helm-env completion zsh > ~/.zsh/completions/_helm-env
# Then add to ~/.zshrc (before any `compinit` call):
#   fpath=(~/.zsh/completions $fpath)
#   autoload -Uz compinit && compinit
```

```sh
# fish — current session
helm-env completion fish | source

# fish — persistent
helm-env completion fish > ~/.config/fish/completions/helm-env.fish
```

```powershell
# powershell — persistent (append to your profile)
helm-env completion --shell pwsh >> $PROFILE
```
