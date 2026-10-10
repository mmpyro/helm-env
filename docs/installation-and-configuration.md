# Installation and configuration

This guide covers supported platforms, installation methods, required setup steps, how `helm-env` discovers configuration, and common troubleshooting.

## Supported platforms

Prebuilt binaries are currently produced for:

- `darwin/amd64`
- `darwin/arm64`
- `linux/amd64`
- `linux/arm64`

If your platform is not listed, install from source.

## Prerequisites

- A POSIX-like shell (e.g. `bash` or `zsh`).
- Permission to create and write files in your chosen `HELMENV_ROOT` directory.
- Network access to GitHub:
  - `helm-env install`, `helm-env latest`, and `helm-env list-remote` fetch information from GitHub.

No existing `helm` installation is required; `helm-env` manages the `helm` binaries it installs.

## Install methods

### Option A: Install a prebuilt binary (recommended)

1. Download the binary for your platform from the project's [GitHub releases](https://github.com/mmpyro/helm-env/releases).
2. Make it executable and move it into a directory on your `PATH`.

Example (Linux x86_64):

```sh
curl -L -o helm-env https://github.com/mmpyro/helm-env/releases/download/v1.0.0/helm-env-linux-amd64
chmod +x helm-env
sudo mv helm-env /usr/local/bin/helm-env
```

### Option B: Build from source

```sh
git clone https://github.com/mmpyro/helm-env.git
cd helm-env
make build
```

The binary will be available at `build/helm-env`.

### Option C: Cross-compile for all supported platforms

```sh
make build-all
```

## Initial setup

### 1) Choose and export `HELMENV_ROOT`

`HELMENV_ROOT` is required. It defines where `helm-env` stores installed versions, the shim, and the global version file.

Add this to your shell profile (e.g. `~/.bashrc` or `~/.zshrc`):

```sh
export HELMENV_ROOT="$HOME/.helmenv"
```

Required permissions:

- `helm-env init` creates directories under `$HELMENV_ROOT`.
- `helm-env install` writes `helm` binaries under `$HELMENV_ROOT/versions/<version>/helm`.
- `helm-env global` writes `$HELMENV_ROOT/version`.

### 2) Initialize shell integration

Add this after the `HELMENV_ROOT` export.
Add this to your shell profile (e.g. `~/.bashrc` or `~/.zshrc`):
```sh
eval "$(helm-env init)"
```

What this does:

- Prepends `$HELMENV_ROOT/shims` to your `PATH` so `helm` resolves to the shim.
- Defines an `helm-env` shell function that enables `helm-env shell` to affect the current shell environment.

Optionally, enable shell completion (`bash`, `zsh`, `fish` or `powershell`; see the [CLI reference](cli-reference.md#completion)):

```sh
eval "$(helm-env completion zsh)"
```

### 3) Install an `helm` version

```sh
helm-env install 3.14.0
```

Or install the latest stable version:

```sh
helm-env install
```

### 4) Configure which version to use

Pick one of:

- Global default (applies everywhere unless overridden):

  ```sh
  helm-env global 3.14.0
  ```

- Per-directory (creates `.helm-version` in the current directory):

  ```sh
  helm-env local 3.14.0
  ```

- Per-shell session (sets `HELMENV_VERSION`; requires `eval "$(helm-env init)"`):

  ```sh
  helm-env shell 3.14.0
  ```

## Environment variables

| Variable | Purpose | Default |
|---|---|---|
| `HELMENV_ROOT` | **Required.** Where versions, shims, the global version file and the cache live. | none |
| `HELMENV_VERSION` | Shell-level version (highest priority). Usually set by `helm-env shell`. | unset |
| `HELMENV_AUTO_INSTALL` | When truthy (`1`, `true`, `yes`, `on`, case-insensitive), missing versions are installed automatically by the shim, `helm-env resolve` and `helm-env exec`. `exec --no-auto` overrides it per call. | unset (disabled) |
| `HELMENV_CACHE_TTL` | Freshness of the release-list cache, as a Go duration (`30m`, `24h`, `0s` disables). See [caching](caching.md). | `1h` |
| `SHELL` | Used by `helm-env completion` to pick a shell when none is given. | from your login shell |

Example: install whatever `.helm-version` asks for on first use.

```sh
export HELMENV_AUTO_INSTALL=1
cd my-chart    # contains .helm-version with "~3.14"
helm version   # installs the newest 3.14.x on first run, then runs it
```

## How configuration is discovered/loaded

When the `helm` shim runs, it selects a version using this priority order:

1. `HELMENV_VERSION` (shell version)
2. `.helm-version` in the current directory or any parent directory (local version)
3. `$HELMENV_ROOT/version` (global version)

Notes:

- The `.helm-version` lookup walks upward until the filesystem root.
- All version values are treated as strings and trimmed for whitespace.

## Common troubleshooting

### `HELMENV_ROOT not set`

Symptoms:

- `helm-env init` prints instructions and exits with an error.

Fix:

```sh
export HELMENV_ROOT="$HOME/.helmenv"
```

Then ensure you also have:

```sh
eval "$(helm-env init)"
```

### `helm-env: no helm version configured`

This is emitted by the `helm` shim when none of the version sources are configured.

Fix (choose one):

```sh
helm-env global 3.14.0
helm-env local 3.14.0
helm-env shell 3.14.0
```

### `version <X> not installed`

`helm-env` validates that a version is installed before setting it via `global`, `local`, or `shell`, and the shim also verifies the installed binary is present.

Fix:

```sh
helm-env install <X>
```

### GitHub API rate limit exceeded

Some commands query the GitHub releases API. If GitHub returns `403`, `helm-env` reports a rate limit error.

Fixes:

- Wait and retry later.
- If running in CI or heavily automated use, consider reducing frequency of `list-remote` / `latest` calls.
