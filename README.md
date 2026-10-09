# helm-env

A version manager for [helm](https://helm.sh), similar to [tfenv](https://github.com/tfutils/tfenv) and [pyenv](https://github.com/pyenv/pyenv).

Manage multiple versions of the `helm` CLI and switch between them seamlessly.

## Documentation

- [Docs index](docs/index.md)
- [Installation and configuration](docs/installation-and-configuration.md)
- [CLI reference](docs/cli-reference.md)
- [Caching strategy](docs/caching.md)

## Downloads (latest tag v1.0.0)

Latest tag v1.0.0: **v1.0.0** (tag: `v1.0.0`).

| Platform | Asset | Download |
|---|---|---|
| macOS (Intel) | `helm-env-darwin-amd64` | [Download](https://github.com/mmpyro/helm-env/releases/download/v1.0.0/helm-env-darwin-amd64) |
| macOS (Apple Silicon) | `helm-env-darwin-arm64` | [Download](https://github.com/mmpyro/helm-env/releases/download/v1.0.0/helm-env-darwin-arm64) |
| Linux (x86_64) | `helm-env-linux-amd64` | [Download](https://github.com/mmpyro/helm-env/releases/download/v1.0.0/helm-env-linux-amd64) |
| Linux (ARM64) | `helm-env-linux-arm64` | [Download](https://github.com/mmpyro/helm-env/releases/download/v1.0.0/helm-env-linux-arm64) |

## Features

- Install and manage multiple `helm` CLI versions
- Automatic version switching based on project directory (`.helm-version`)
- Shell-level, local (directory), and global version configuration
- Version priority: shell > local > global
- Auto-detection of OS and architecture for binary downloads
- Shim-based transparent proxying of `helm` commands

## Installation

### From Source

```sh
git clone https://github.com/mmpyro/helm-env.git
cd helm-env
make build
```

The binary will be at `build/helm-env`. Move it to a directory in your `PATH`:

```sh
sudo mv build/helm-env /usr/local/bin/
```

### Cross-compile for All Platforms

```sh
make build-all
```

This produces binaries for:
- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`

## Quick Start

### 1. Set up HELMENV_ROOT

Add to your `~/.bashrc` or `~/.zshrc`:

```sh
export HELMENV_ROOT="$HOME/.helmenv"
```

Reload your shell:

```sh
source ~/.bashrc  # or source ~/.zshrc
```

### 2. Initialize helm-env

Add the following to your `~/.bashrc` or `~/.zshrc` (after the `HELMENV_ROOT` export):

```sh
eval "$(helm-env init)"
```

This sets up:
- A `helm-env` shell function for `helm-env shell` support
- PATH prepend for the helm shim

### 3. Install a helm version

```sh
# Install a specific version
helm-env install 3.14.0

# Install the latest stable version
helm-env install
```

### 4. Set a version

```sh
# Set global default
helm-env global 3.14.0

# Set for current directory (creates .helm-version)
helm-env local 3.14.0

# Set for current shell session
helm-env shell 3.14.0
```

### 5. Use helm

```sh
helm version
```

The shim automatically resolves and uses the correct version.

## Command Reference

Full reference: [docs/cli-reference.md](docs/cli-reference.md)

| Command | Description |
|---------|-------------|
| `helm-env help` | Display help and all available commands |
| `helm-env list` | List all installed versions |
| `helm-env list-remote` | List all available helm versions from GitHub |
| `helm-env list-remote --prerelease` | Include pre-release helm versions |
| `helm-env latest` | Print the latest available version of helm from GitHub |
| `helm-env init` | Initialize helm-env setup |
| `helm-env status` | Show current environment status |
| `helm-env install [VERSION]` | Install a specific version (or latest) |
| `helm-env uninstall VERSION` | Uninstall a specific version |
| `helm-env exec [--auto\|--no-auto] VERSION CMD` | Run a command using a specific helm version |
| `helm-env shell [VERSION]` | Set/show shell version (`HELMENV_VERSION`) |
| `helm-env local [VERSION]` | Set/show local version (`.helm-version`) |
| `helm-env global [VERSION]` | Set/show global version (`$HELMENV_ROOT/version`) |
| `helm-env which [--explain]` | Print path to active helm binary (optionally with resolution trace) |
| `helm-env resolve` | Resolve the active constraint and print the concrete installed version |
| `helm-env prune [--keep N] [--older-than DUR] [--dry-run]` | Remove old installed versions |
| `helm-env doctor` | Run diagnostics against the helm-env installation |
| `helm-env version` | Print helm-env version |

## Version strings

Every version slot (`HELMENV_VERSION`, `.helm-version`, `$HELMENV_ROOT/version`, and every CLI `<version>` argument) accepts:

- Exact: `3.14.0`
- Partial: `3.14` → highest `3.14.x`; `3` → highest `3.x.y`
- Aliases: `latest` / `latest-stable` / `stable` (highest stable), `latest-installed` (highest installed)
- Tilde: `~3.14.0`, `~3.14`
- Caret: `^3.14.0`
- Comparators: `>3.14.0`, `>=3.14.0`, `<4.0.0`, `<=3.14.0`, `=3.14.0`
- AND-combined ranges: `>=3.12.0,<4.0.0`

`install` and the `latest*` aliases resolve against the remote release list; every other command resolves against installed versions on disk.

## Auto-install

Set `HELMENV_AUTO_INSTALL` to a truthy value (`1`, `true`, `yes`, `on`) and `helm-env` will transparently install missing helm versions when they are first resolved. Per-invocation overrides for `helm-env exec`:

- `--auto` → force auto-install for this call.
- `--no-auto` → disable auto-install for this call even if the env var is set.

Auto-install is **disabled** by default: a silent fallback is otherwise dangerous.

## Version Priority

When `helm` is invoked, the version is resolved in this order:

1. **Shell** — `HELMENV_VERSION` environment variable (set via `helm-env shell`)
2. **Local** — `.helm-version` file in the current or parent directories (set via `helm-env local`)
3. **Global** — `$HELMENV_ROOT/version` file (set via `helm-env global`)

If no version is configured at any level, the command fails with an informative error.

## Shell Setup

### Bash

Add the following to your `~/.bashrc`:

```sh
# helm-env setup
export HELMENV_ROOT="$HOME/.helmenv"
eval "$(helm-env init)"
source <(helm-env autocompletion bash)
```

### Zsh

Add the following to your `~/.zshrc`:

```sh
# helm-env setup
export HELMENV_ROOT="$HOME/.helmenv"
eval "$(helm-env init)"
eval "$(helm-env autocompletion zsh)"
```

### Fish

Add the following to your `~/.config/fish/config.fish`:

```fish
# helm-env setup
set -gx HELMENV_ROOT "$HOME/.helmenv"
helm-env init | source
helm-env autocompletion fish | source
```

### PowerShell

Append the completion script to your profile once:

```powershell
helm-env autocompletion powershell >> $PROFILE
```

## Release source

`helm-env` discovers available versions from the [`helm/helm`](https://github.com/helm/helm) GitHub repository, and downloads the corresponding binary tarballs from <https://get.helm.sh/>, mirroring Helm's official install convention.

## Caching

`helm-env` uses a three-layer caching strategy to keep `list-remote` and `latest` fast and reliable. For more details on how it works and how to configure it, see [docs/caching.md](docs/caching.md).

## Directory Structure

```
$HELMENV_ROOT/
├── versions/           # Installed helm versions
│   ├── 3.14.0/
│   │   └── helm        # helm binary
│   └── 3.20.0/
│       └── helm
├── shims/
│   └── helm            # Shim script (auto-generated)
└── version             # Global version file
```

## Development

### Run Tests

```sh
make test
```

### Run Docker Integration Tests

```sh
make test-docker
```

### Build

```sh
make build          # Current platform
make build-all      # All platforms
```

## License

See [LICENSE](LICENSE) for details.
