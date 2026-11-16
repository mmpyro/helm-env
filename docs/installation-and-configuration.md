# Installation and configuration

This guide covers the supported platforms, the install methods, the required setup steps, how `helm-env` finds its configuration, and common problems.

## Supported platforms

The project currently builds prebuilt binaries for:

- `darwin/amd64`
- `darwin/arm64`
- `linux/amd64`
- `linux/arm64`

If your platform is not in this list, install from source.

## Prerequisites

- A POSIX-like shell (for example, `bash` or `zsh`).
- Permission to create and write files in your chosen `HELMENV_ROOT` directory.
- Network access to GitHub:
  - `helm-env install`, `helm-env latest`, and `helm-env list-remote` fetch information from GitHub.

You do not need an existing `helm` installation. `helm-env` manages the `helm` binaries that it installs.

## Install methods

=== "Prebuilt binary (recommended)"

    1. Download the binary for your platform from the project's [GitHub releases](https://github.com/mmpyro/helm-env/releases).
    2. Make it executable and move it into a directory on your `PATH`.

    Example (Linux x86_64):

    ```sh
    curl -L -o helm-env https://github.com/mmpyro/helm-env/releases/download/v1.2.0/helm-env-linux-amd64
    chmod +x helm-env
    sudo mv helm-env /usr/local/bin/helm-env
    ```

=== "Build from source"

    ```sh
    git clone https://github.com/mmpyro/helm-env.git
    cd helm-env
    make build
    ```

    `make build` writes the binary to `build/helm-env`.

=== "Cross-compile for all platforms"

    ```sh
    make build-all
    ```

    This command builds binaries for `linux/amd64`, `linux/arm64`, `darwin/amd64`, and `darwin/arm64`.

## Initial setup

### 1) Choose and export `HELMENV_ROOT`

`HELMENV_ROOT` is required. It defines where `helm-env` stores installed versions, the shim, and the global version file.

Add this line to your shell profile (for example, `~/.bashrc` or `~/.zshrc`):

```sh
export HELMENV_ROOT="$HOME/.helmenv"
```

Required permissions:

- `helm-env init` creates directories under `$HELMENV_ROOT`.
- `helm-env install` writes `helm` binaries under `$HELMENV_ROOT/versions/<version>/helm`.
- `helm-env global` writes `$HELMENV_ROOT/version`.

### 2) Initialize shell integration

Add the following to your shell profile, after the `HELMENV_ROOT` export. Pick the tab for your shell:

=== "Bash"

    Add these lines to `~/.bashrc`:

    ```sh
    # helm-env setup
    export HELMENV_ROOT="$HOME/.helmenv"
    eval "$(helm-env init)"
    source <(helm-env autocompletion bash)
    ```

=== "Zsh"

    Add these lines to `~/.zshrc`:

    ```sh
    # helm-env setup
    export HELMENV_ROOT="$HOME/.helmenv"
    eval "$(helm-env init)"
    eval "$(helm-env autocompletion zsh)"
    ```

=== "Fish"

    Add these lines to `~/.config/fish/config.fish`:

    ```fish
    # helm-env setup
    set -gx HELMENV_ROOT "$HOME/.helmenv"
    helm-env init | source
    helm-env autocompletion fish | source
    ```

=== "PowerShell"

    Append the completion script to your profile once:

    ```powershell
    helm-env autocompletion powershell >> $PROFILE
    ```

What `eval "$(helm-env init)"` does:

- It adds `$HELMENV_ROOT/shims` to the start of your `PATH`, so `helm` resolves to the shim.
- It defines a `helm-env` shell function. This function lets `helm-env shell` change the current shell environment.

You can also enable shell completion for `bash`, `zsh`, `fish` or `powershell` (see the [CLI reference](cli-reference.md#completion)):

```sh
eval "$(helm-env completion zsh)"
```

### 3) Install an `helm` version

```sh
helm-env install 3.14.0
```

To install the latest stable version, run:

```sh
helm-env install
```

### 4) Configure which version to use

Pick one of these options:

- Global default (applies everywhere, unless another setting overrides it):

  ```sh
  helm-env global 3.14.0
  ```

- Per-directory (creates `.helm-version` in the current directory):

  ```sh
  helm-env local 3.14.0
  ```

- Per-shell session (sets `HELMENV_VERSION` and requires `eval "$(helm-env init)"`):

  ```sh
  helm-env shell 3.14.0
  ```

## Environment variables

| Variable | Purpose | Default |
|---|---|---|
| `HELMENV_ROOT` | **Required.** Where versions, shims, the global version file and the cache live. | none |
| `HELMENV_VERSION` | Shell-level version (highest priority). Usually `helm-env shell` sets it. | unset |
| `HELMENV_AUTO_INSTALL` | When truthy (`1`, `true`, `yes`, `on`, case-insensitive), the shim, `helm-env resolve` and `helm-env exec` install missing versions automatically. `exec --no-auto` overrides it for one call. | unset (disabled) |
| `HELMENV_CACHE_TTL` | How long the release-list cache stays fresh, as a Go duration (`30m`, `24h`). `0s` disables the cache. See [caching](caching.md). | `1h` |
| `SHELL` | `helm-env completion` reads it to pick a shell when you do not give one. | from your login shell |

Example: install the version that `.helm-version` asks for on first use.

```sh
export HELMENV_AUTO_INSTALL=1
cd my-chart    # contains .helm-version with "~3.14"
helm version   # installs the newest 3.14.x on first run, then runs it
```

## How configuration is discovered/loaded

When the `helm` shim runs, it selects a version in this order of priority:

1. `HELMENV_VERSION` (shell version)
2. `.helm-version` in the current directory or any parent directory (local version)
3. `$HELMENV_ROOT/version` (global version)

Notes:

- The `.helm-version` lookup goes up the directory tree to the filesystem root.
- `helm-env` treats all version values as strings and trims whitespace from them.

## Common troubleshooting

### `HELMENV_ROOT not set`

Symptoms:

- `helm-env init` prints instructions and exits with an error.

!!! note "Fix"
    Export `HELMENV_ROOT`. Then run the shell integration again:

    ```sh
    export HELMENV_ROOT="$HOME/.helmenv"
    eval "$(helm-env init)"
    ```

### `helm-env: no helm version configured`

The `helm` shim prints this error when no version source has a configured version.

!!! note "Fix (choose one)"
    ```sh
    helm-env global 3.14.0
    helm-env local 3.14.0
    helm-env shell 3.14.0
    ```

### `version <X> not installed`

Before `global`, `local` or `shell` sets a version, `helm-env` makes sure that the version is installed. The shim also makes sure that the installed binary is present.

!!! note "Fix"
    ```sh
    helm-env install <X>
    ```

### GitHub API rate limit exceeded

Some commands query the GitHub releases API. If GitHub returns `403`, `helm-env` reports a rate limit error.

!!! note "Fix"
    - Wait and retry later.
    - In CI or other heavily automated use, consider calling `list-remote` and `latest` less often.
