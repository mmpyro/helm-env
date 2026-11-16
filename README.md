# helm-env

A version manager for [helm](https://helm.sh), similar to [tfenv](https://github.com/tfutils/tfenv) and [pyenv](https://github.com/pyenv/pyenv).

Manage multiple versions of the `helm` CLI and switch between them.

## Documentation

- [Docs index](docs/index.md)
- [Installation and configuration](docs/installation-and-configuration.md)
- [CLI reference](docs/cli-reference.md)
- [Caching strategy](docs/caching.md)

## Downloads (latest tag v1.2.0)

The latest tag is **v1.2.0** (tag: `v1.2.0`).

| Platform | Asset | Download |
|---|---|---|
| macOS (Intel) | `helm-env-darwin-amd64` | [Download](https://github.com/mmpyro/helm-env/releases/download/v1.2.0/helm-env-darwin-amd64) |
| macOS (Apple Silicon) | `helm-env-darwin-arm64` | [Download](https://github.com/mmpyro/helm-env/releases/download/v1.2.0/helm-env-darwin-arm64) |
| Linux (x86_64) | `helm-env-linux-amd64` | [Download](https://github.com/mmpyro/helm-env/releases/download/v1.2.0/helm-env-linux-amd64) |
| Linux (ARM64) | `helm-env-linux-arm64` | [Download](https://github.com/mmpyro/helm-env/releases/download/v1.2.0/helm-env-linux-arm64) |

## Features

- Install and manage multiple `helm` CLI versions
- Switch versions automatically per project directory (`.helm-version`)
- Set the version per shell, per directory (local) or globally
- Version priority: shell > local > global
- Detect the OS and architecture automatically for binary downloads
- Forward `helm` commands transparently through a shim

## Installation

### From source

```sh
git clone https://github.com/mmpyro/helm-env.git
cd helm-env
make build
```

`make build` writes the binary to `build/helm-env`. Move it to a directory in your `PATH`:

```sh
sudo mv build/helm-env /usr/local/bin/
```

### Cross-compile for all platforms

```sh
make build-all
```

This command builds binaries for:
- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`

## Quick start

### 1. Set up HELMENV_ROOT

Add this line to your `~/.bashrc` or `~/.zshrc`:

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

This line adds:
- A `helm-env` shell function, so that `helm-env shell` works
- The helm shim directory at the start of your PATH

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

The shim resolves the correct version automatically and uses it.

## Command reference

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
| `helm-env exec [--auto\|--no-auto] SPEC [--] ARGS...` | Run helm with a specific version and exit with helm's exit code |
| `helm-env shell [VERSION]` | Set/show shell version (`HELMENV_VERSION`) |
| `helm-env local [VERSION]` | Set/show local version (`.helm-version`) |
| `helm-env global [VERSION]` | Set/show global version (`$HELMENV_ROOT/version`) |
| `helm-env which [--explain]` | Print path to active helm binary (optionally with resolution trace) |
| `helm-env resolve [SPEC] [--install] [-s]` | Resolve a spec (default: the active one) to a concrete version |
| `helm-env prune [--keep-last N] [--older-than DUR] [--dry-run] [--yes]` | Remove old installed versions (dry run unless `--yes`) |
| `helm-env doctor [--fix]` | Run diagnostics. `--fix` regenerates the shim |
| `helm-env completion [SHELL] [--shell SHELL]` | Print a completion script (bash, zsh, fish, powershell/pwsh) |
| `helm-env upgrade` | Upgrade helm-env itself to the latest GitHub release |
| `helm-env version` | Print helm-env version |

Every command accepts `-h`/`--help`.

## Version strings

Every version slot (`HELMENV_VERSION`, `.helm-version`, `$HELMENV_ROOT/version`, and every CLI `<version>` argument) accepts:

- Exact: `3.14.0`
- Partial: `3.14` (highest `3.14.x`) or `3` (highest `3.x.y`)
- Aliases: `latest`, `latest-stable` or `stable` (highest stable version), and `latest-installed` (highest installed version)
- Tilde: `~3.14.0`, `~3.14`
- Caret: `^3.14.0`
- Comparators: `>3.14.0`, `>=3.14.0`, `<4.0.0`, `<=3.14.0`, `=3.14.0`
- AND-combined ranges: `>=3.12.0,<4.0.0`

`install` and the `latest*` aliases resolve against the remote release list. Every other command resolves against the installed versions on disk.

## Auto-install

If you set `HELMENV_AUTO_INSTALL` to a truthy value (`1`, `true`, `yes`, `on`), `helm-env` installs missing helm versions transparently. It installs a version the first time the shim, `resolve` or `exec` resolves it. `helm-env resolve --install` does the same for one call. `helm-env exec` has two flags that override the setting for one call:

- `--auto` forces auto-install.
- `--no-auto` disables auto-install, even if the environment variable is set.

By default, auto-install is **disabled**, because a silent fallback is dangerous.

## Version priority

When you run `helm`, the shim resolves the version in this order:

1. Shell: the `HELMENV_VERSION` environment variable (set with `helm-env shell`)
2. Local: the `.helm-version` file in the current directory or a parent directory (set with `helm-env local`)
3. Global: the `$HELMENV_ROOT/version` file (set with `helm-env global`)

If you do not configure a version at any level, the command fails with an informative error.

## Shell setup

### Bash

Add the following to your `~/.bashrc`:

```sh
# helm-env setup
export HELMENV_ROOT="$HOME/.helmenv"
eval "$(helm-env init)"
source <(helm-env completion bash)
```

### Zsh

Add the following to your `~/.zshrc`:

```sh
# helm-env setup
export HELMENV_ROOT="$HOME/.helmenv"
eval "$(helm-env init)"
eval "$(helm-env completion zsh)"
```

### Fish

Add the following to your `~/.config/fish/config.fish`:

```fish
# helm-env setup
set -gx HELMENV_ROOT "$HOME/.helmenv"
helm-env init | source
helm-env completion fish | source
```

### PowerShell

Append the completion script to your profile once:

```powershell
helm-env completion powershell >> $PROFILE
```

`helm-env autocompletion` is a deprecated alias for `helm-env completion`. It still works, but it prints a deprecation warning to stderr.

## Release source

`helm-env` gets the list of available versions from the [`helm/helm`](https://github.com/helm/helm) GitHub repository. It downloads the matching binary tarballs from <https://get.helm.sh/>. This follows Helm's official install convention.

## Caching

`helm-env` uses a three-layer caching strategy to keep `list-remote` and `latest` fast and reliable. For details on how the cache works and how to configure it, see [docs/caching.md](docs/caching.md).

## Directory structure

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

### Run tests

```sh
make test
```

### Run Docker integration tests

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
