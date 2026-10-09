# CLI reference

This reference covers every `helm-env` command implemented by the project.

Conventions:

- Commands return exit code `0` on success.
- On errors, commands typically print an error message to stderr and exit with code `1`.
- Some commands print help and exit `0`.

## Global usage

```text
helm-env <command> [arguments]
```

Top-level help is available via `helm-env help`, `helm-env --help`, or `helm-env -h`.

## Environment variables

### `HELMENV_ROOT`

Required. Path where `helm-env` stores installed versions and shims.

Used by most commands and required for initialization.

### `HELMENV_VERSION`

Optional. When set, it forces a particular `helm` version to be used (highest priority).

Typically set via `helm-env shell` after enabling shell integration with `eval "$(helm-env init)"`.

### `HELMENV_AUTO_INSTALL`

Optional. When set to a truthy value (`1`, `true`, `yes`, `on`, case-insensitive), `helm-env` will *automatically install* a missing helm version whenever the resolver (shim, `exec`, `which`, etc.) encounters a version/constraint that is not yet present under `$HELMENV_ROOT/versions/`.

The installation is silent (equivalent to `helm-env install --silent <version>`).
To override at the command line, see the `--auto` and `--no-auto` flags on `helm-env exec`.

## Version strings

Every command that accepts a `<version>` argument (and every version slot
read by `helm-env`: `HELMENV_VERSION`, `.helm-version`, and
`$HELMENV_ROOT/version`) accepts the following forms:

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
- Print shell initialization code to stdout (intended to be evaluated by your shell).

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

Purpose: List all available `helm` versions from GitHub tag v1.0.0 (newest to oldest).

This command does not require `HELMENV_ROOT` or initialization. If `HELMENV_ROOT` is set, results are persistently cached on disk (see [caching strategy](caching.md)).

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

Purpose: Print the latest available `helm` version from GitHub tag v1.0.0.

This command does not require `HELMENV_ROOT` or initialization. If `HELMENV_ROOT` is set, results are persistently cached on disk (see [caching strategy](caching.md)).

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

If `<version>` is omitted, `helm-env` installs the latest stable version.

The command displays a progress bar during the download and automatically verifies the integrity of the downloaded file using SHA256 checksums from the GitHub tag v1.0.0.

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
- `1` if not initialized, platform detection fails, download fails, checksum mismatch, or filesystem writes fail.

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

Options/flags: none.

Environment variables:

- `HELMENV_ROOT` (required)

Exit codes:

- `0` on success.
- `1` if `<version>` is missing, not initialized, the version is not installed, or filesystem removal fails.

Example:

```sh
helm-env uninstall 3.14.0
```

---

### `shell`

Purpose: Set or show the shell-level `helm` version.

Important: To *set* the version for your current shell session, you must have shell integration enabled via `eval "$(helm-env init)"`. Otherwise, you will only see the printed `export ...` line but your current shell will not be updated.

Syntax:

```text
helm-env shell            # show
helm-env shell <version>  # set
```

Options/flags: none.

Environment variables:

- `HELMENV_ROOT` (required)
- `HELMENV_VERSION` (read on show; written when you eval shell integration)

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

Setting writes a `.helm-version` file into the current directory.

Syntax:

```text
helm-env local            # show
helm-env local <version>  # set
```

Options/flags: none.

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

Setting writes `$HELMENV_ROOT/version`.

Syntax:

```text
helm-env global            # show
helm-env global <version>  # set
```

Options/flags: none.

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

Purpose: Print the full path to the active `helm` binary that would be used based on version resolution.

Syntax:

```text
helm-env which [flags]
```

Options/flags:

- `--explain`: additionally print a trace of the resolution process to stderr (which source — shell / local / global — won, which parent directories were searched for `.helm-version`, and the final concrete version + binary path).
- `-h`, `--help`: show command help and exit.

Environment variables:

- `HELMENV_ROOT` (required)
- `HELMENV_VERSION` (optional; highest priority if set)

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

Purpose: Print the concrete installed helm version produced by resolving the active version/constraint against `$HELMENV_ROOT/versions`.

Syntax:

```text
helm-env resolve [--concrete]
```

Options/flags:

- `--concrete`: print only the concrete installed version (default; accepted for forward-compatibility).
- `-h`, `--help`: show command help and exit.

Environment variables:

- `HELMENV_ROOT` (required)
- `HELMENV_VERSION` (optional)

Exit codes:

- `0` when the active constraint resolves to an installed version.
- `1` otherwise (not initialized, no version configured, or constraint resolves to a version that is not installed).

Example:

```sh
# .helm-version contains "~3.14"; the installed versions include 3.14.0 and 3.14.5
$ helm-env resolve
3.14.5
```

The shim uses `helm-env resolve --concrete` under the hood when the configured value is a constraint rather than an exact semver.

---

### `upgrade`

Purpose: Download the tag v1.0.0 of `helm-env` from GitHub and replace the current binary in-place.

Syntax:

```text
helm-env upgrade
```

Options/flags: none.

Environment variables: none (the binary path is auto-detected via `os.Executable()`).

Exit codes:

- `0` on success, or when already up to date.
- `1` on network/download errors, permission issues, or filesystem errors.

Notes:

- The command detects the current OS and CPU architecture automatically.
- The new binary is written atomically (temp file + rename) to avoid corruption.
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
helm-env exec [--auto|--no-auto] <version> <command> [args...]
```

Options/flags:

- `--auto`: enable auto-install for this invocation (installs the resolved version if it is not yet present on disk). Overrides `HELMENV_AUTO_INSTALL`.
- `--no-auto`: disable auto-install for this invocation even if `HELMENV_AUTO_INSTALL` is set.

Version resolution follows the general [version strings](#version-strings) rules: an exact installed version is used verbatim, otherwise the value is matched against the installed candidates (or, when auto-install is enabled, against the remote release list before installation).

Environment variables:

- `HELMENV_ROOT` (required)
- `HELMENV_VERSION` (set for the subprocess to match the resolved version)
- `HELMENV_AUTO_INSTALL` (optional)

Exit codes:

- Exit code of the executed command.
- `1` if the version is not installed (and auto-install is disabled) or initialization fails.

Example:

```sh
helm-env exec 3.14.0 version
helm-env exec ~3.14 version
HELMENV_AUTO_INSTALL=1 helm-env exec 3.14.0 version
helm-env exec --auto latest version
```

---

### `status`

Purpose: Display a comprehensive overview of the current `helm-env` environment.

Output includes:
- `HELMENV_ROOT` path.
- Currently active version and the source it was resolved from.
- Full path to the active `helm` binary.
- List of all installed versions (active one marked with `*`).

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

Purpose: Remove installed `helm` versions that are no longer needed.

Syntax:

```text
helm-env prune [--keep N] [--older-than DUR] [--dry-run]
```

Options/flags:

- `--keep N`: keep the N newest installed versions.
- `--older-than DUR`: remove installed versions whose helm binary mtime is older than `DUR`. Accepts Go duration strings (e.g. `30m`, `24h`) plus a `<N>d` suffix for whole days.
- `--dry-run`: print what would be removed without touching the disk.
- `-h`, `--help`: show command help and exit.

If both `--keep` and `--older-than` are supplied, a version must satisfy **BOTH** rules to be kept. The currently-resolved version is always protected (a warning is printed and the removal is skipped).

Environment variables:

- `HELMENV_ROOT` (required)

Exit codes:

- `0` on success (including dry-run and no-op).
- `1` on bad flags or when `helm-env` is not initialized.

Example:

```sh
helm-env prune --keep 3
helm-env prune --older-than 30d
helm-env prune --keep 2 --older-than 90d --dry-run
```

---

### `doctor`

Purpose: Run a set of diagnostic checks against the current `helm-env` installation. Each check prints one line prefixed with `[OK]`, `[WARN]`, or `[FAIL]`.

Syntax:

```text
helm-env doctor
```

Checks:

1. `HELMENV_ROOT` is set.
2. `$HELMENV_ROOT` exists and is writable.
3. `$HELMENV_ROOT/versions` exists.
4. `$HELMENV_ROOT/shims/helm` exists and is executable.
5. `$HELMENV_ROOT/shims` is on `$PATH` and appears before any other `helm` binary.
6. The cache file (if present) is parseable JSON.
7. Reachability of `https://github.com` and `https://get.helm.sh` (WARN on failure; never fatal).
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

### `autocompletion`

Purpose: Generate bash autocompletion script for `helm-env`.

The script provides completion for subcommands and suggests installed versions for commands that accept a version argument (`install`, `uninstall`, `shell`, `local`, `global`, `exec`).

Syntax:

```text
helm-env autocompletion
```

Options/flags:
- `-h`, `--help`: show command help and exit

Environment variables: none.

Exit codes:
- `0` on success.

Example:

```sh
# Enable autocompletion for the current session
source <(helm-env autocompletion)

# Enable autocompletion permanently
echo 'source <(helm-env autocompletion)' >> ~/.bashrc
```
