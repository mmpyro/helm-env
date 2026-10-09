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
helm-env which
```

Options/flags: none.

Environment variables:

- `HELMENV_ROOT` (required)
- `HELMENV_VERSION` (optional; highest priority if set)

Exit codes:

- `0` on success.
- `1` if not initialized or no version is configured.

Example:

```sh
helm-env which
```

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
helm-env exec <version> <command> [args...]
```

Options/flags: none.

Environment variables:

- `HELMENV_ROOT` (required)
- `HELMENV_VERSION` (set for the subprocess to match the requested version)

Exit codes:

- Exit code of the executed command.
- `1` if the version is not installed or initialization fails.

Example:

```sh
helm-env exec 3.14.0 version
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

### `autocompletion`

Purpose: Generate a shell completion script for `helm-env`.

Supported shells: `bash`, `zsh`, `fish`, `powershell`.

The script provides completion for subcommands and suggests installed versions
for commands that accept a version argument (`uninstall`, `shell`, `local`,
`global`, `exec`). For `install`, `-s`/`--silent` and `-h`/`--help` are
suggested; for `list-remote` and `latest`, `--prerelease` and `--help` are
suggested.

Syntax:

```text
helm-env autocompletion [SHELL]
```

When `SHELL` is omitted, `helm-env` detects your current shell from the
`$SHELL` environment variable (falling back to `bash`). A short notice is
written to stderr so you still see which script was generated when the output
is piped to `source`. PowerShell is never auto-detected and must be requested
explicitly.

Options/flags:
- `-h`, `--help`: show command help and exit

Environment variables:
- `$SHELL` (optional; read for shell auto-detection when `SHELL` arg is omitted)

Exit codes:
- `0` on success.
- `1` if an unsupported shell is requested.

Examples:

```sh
# bash — current session
source <(helm-env autocompletion bash)

# bash — persistent
echo 'source <(helm-env autocompletion bash)' >> ~/.bashrc
```

```sh
# zsh — current session (simplest)
eval "$(helm-env autocompletion zsh)"

# zsh — persistent via fpath + compinit
mkdir -p ~/.zsh/completions
helm-env autocompletion zsh > ~/.zsh/completions/_helm-env
# Then add to ~/.zshrc (before any `compinit` call):
#   fpath=(~/.zsh/completions $fpath)
#   autoload -Uz compinit && compinit
```

```sh
# fish — current session
helm-env autocompletion fish | source

# fish — persistent
helm-env autocompletion fish > ~/.config/fish/completions/helm-env.fish
```

```powershell
# powershell — persistent (append to your profile)
helm-env autocompletion powershell >> $PROFILE
```
