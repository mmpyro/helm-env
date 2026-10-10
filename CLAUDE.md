# CLAUDE.md

This file guides Claude Code (claude.ai/code) when working in this repository.

## Overview

`helm-env` is a version manager for the `helm` CLI, in the style of tfenv and pyenv.
- Versions are installed under `$HELMENV_ROOT`, conventionally `~/.helmenv`.
- A generated POSIX shim at `$HELMENV_ROOT/shims/helm` picks the version on every call. Precedence: **shell** (`HELMENV_VERSION`) > **local** (`.helm-version`, searched upward from the current directory) > **global** (`$HELMENV_ROOT/version`).
- Release lists come from the GitHub `helm/helm` API. Tarballs download from `https://get.helm.sh`.

Go 1.24.4, **standard library only**: no third-party dependencies and no `go.sum`. Keep it that way unless asked.

## Commands

```sh
make build            # build/helm-env  (ldflags: -X main.Version=$(VERSION))
make build-all        # linux/darwin × amd64/arm64 into build/
make test             # go test -v -race ./...
make test-docker      # BATS integration suite in Docker (needs Docker + network)
make install          # go install ./cmd/helm-env
make clean
make docs-build       # mkdocs build --strict into site/ (creates .venv-docs from docs/requirements.txt)
make docs-serve       # local preview of the MkDocs site

go test -race -run 'TestName/subtest' -v ./internal/semver/   # single unit test
go vet ./... && golangci-lint run                             # lint (CI uses golangci-lint, default config)
```

There is no lint Makefile target; CI runs golangci-lint.

## Architecture

- **Dispatch is hand-rolled; there is no cobra.** `cmd/helm-env/main.go` runs a big `switch args[0]` that parses flags by hand for each command and calls `commands.X(...)`.
  - An error goes to stderr and the process exits 1.
  - `main.Version` is copied into `commands.Version`.

| Package | Role |
|---|---|
| `internal/commands` | One file per subcommand. Each exports `X(...) error` and `XHelp()`. `autoinstall.go` (`AutoInstallMode`, reads `HELMENV_AUTO_INSTALL`), `versions_shared.go` (`getRemoteVersions`: cache + GitHub), `completion.go` (bash/zsh/fish/powershell scripts, `subcommands` list; `autocompletion` is a hidden deprecated alias), `upgrade.go` (atomic self-upgrade from `mmpyro/helm-env`) |
| `internal/config` | Root, init checks, version resolution (`ResolveVersion`, `FindLocalVersionFrom`, `ReadGlobalVersion`). `resolve.go` provides `ResolveConcreteVersion` and `ResolveVersionTraced`, which `which --explain` uses |
| `internal/semver` | `Parse`, `Less`, `SortDescending`. `match.go` provides `Match`, `IsAlias` and `IsExact`. Accepted forms: exact, partial (`3.14`), aliases (`latest`, `stable`, `latest-stable`, `latest-installed`), `~`, `^`, comparators, and AND-ranges joined by `,` |
| `internal/github` | `Client{BaseURL, DownloadBaseURL, HTTPClient}`. Lists releases (with Link-header paging) and downloads with progress |
| `internal/cache` | JSON cache at `$HELMENV_ROOT/cache/releases.json`, TTL 1h (`HELMENV_CACHE_TTL`). `baseline.go` holds a hard-coded list of known versions. See `docs/caching.md` |
| `internal/platform` | OS/arch detection and archive and checksum paths. `selfurl.go` provides the URL for self-upgrade |
| `internal/shim` | `GenerateShimScript` (the `#!/bin/sh` shim) and `GenerateShellInit` (adds to PATH and defines the `helm-env()` shell function used for `helm-env shell`) |

Runtime layout:
- `$HELMENV_ROOT/versions/<ver>/helm`
- `$HELMENV_ROOT/shims/helm` (generated; do not edit)
- `$HELMENV_ROOT/version`
- `$HELMENV_ROOT/cache/releases.json`

### Adding or changing a command

1. Add a `case` in `cmd/helm-env/main.go`.
2. Add `X()` and `XHelp()` in `internal/commands/<cmd>.go`, plus `<cmd>_test.go`.
3. List it in `internal/commands/help.go`.
4. Add it to the `subcommands` list and descriptions in `completion.go`.
5. Update `docs/cli-reference.md` and the README command table.
6. Add BATS tests in `tests/integration.bats`.

## Testing

- **Unit tests** sit next to the code (`*_test.go`).
  - Use the standard `testing` package with `t.Run` subtests and `net/http/httptest`. Do not add assertion libraries.
  - Isolate each test with `t.Setenv("HELMENV_ROOT", t.TempDir())`.
  - Use the existing seams instead of hitting the network: `installWithClient`, `latestWithClient`, `listRemoteWithClient`, `doctorWithWriter`, `var installerFunc`, `var doctorHTTPClient`.
  - `internal/shim/shim_integration_test.go` runs the real generated shim. It has no build tag, so it runs as part of `go test ./...`.
- **BATS integration tests** live in `tests/integration.bats` (39 tests, bats-support and bats-assert).
  - They run only in Docker through `Dockerfile.test`, with `scripts/run-bats.sh` as the entrypoint.
  - They hit real GitHub and get.helm.sh.

### Every user-facing change needs BATS coverage

This covers any new or changed command, flag, env var, or resolution or shim behavior. Add or update `@test` cases in `tests/integration.bats`. Go unit tests alone are not enough for behavior the CLI shows to users.
- Follow the existing pattern. `setup()` gives a fresh `HELMENV_ROOT=$BATS_TMPDIR/helmenv` with the shims on PATH, and `teardown()` removes it.
- Call `helm-env init`, then `run helm-env ...`, then `assert_success` and `assert_output --partial "..."` (or `--regexp`).
- Cover the error path too: `assert_failure`.

```bash
@test "helm-env <cmd> <does what>" {
	helm-env init
	run helm-env <cmd> <args>
	assert_success
	assert_output --partial "<expected>"
}
```

## Definition of done: final verification

Run these in order before calling any change complete:
1. `go vet ./...`, plus `golangci-lint run` if it is installed.
2. `make test`: unit tests with `-race`.
3. `make docs-build`, if you touched `docs/` or `mkdocs.yml`.
4. `make test-docker`: the **full BATS suite**. This is **always the last step**, and it needs Docker running and network access.

Report the real pass/fail counts from the output. If Docker is unavailable, say so plainly; do not claim success.

To iterate on one BATS test:
```sh
docker build -f Dockerfile.test -t helm-env-test .
docker run --rm -e TERM=xterm --entrypoint bats helm-env-test --filter "<test name regex>" /src/tests/integration.bats
```
Always finish with a full `make test-docker` afterwards.

## CLI contract (shared with istioctl-env and vc-env)

These three sibling repos expose the same command surface. Keep them in sync when you change one of them.

- `resolve [<spec>] [--install] [-s|--silent]`: with no spec, uses the active spec. Matches installed versions first and goes remote only with `--install` or auto-install. stdout is only the bare version.
- `exec [--auto|--no-auto] <spec> [--] <cmd> [args...]`: strips one leading `--`. **Exits with the child's exit code**, via `childExitCode` in main.go.
- `completion [<shell>] [--shell <shell>]`: supports bash, zsh, fish, powershell and `pwsh`, falling back to `$SHELL`. `autocompletion` is a hidden alias that prints a deprecation warning to stderr.
- `-h/--help` short-circuits on **every** command, and must never run side effects; `upgrade -h` must not upgrade.
- `prune [--keep-last N] [--older-than DUR] [--dry-run] [--yes]`: a **dry run unless `--yes`**. `--keep` is a deprecated alias.
- `doctor [--fix]`: helm-env has no `--deep`; the other two repos do.

## Environment variables

| Variable | Purpose |
|---|---|
| `HELMENV_ROOT` | Required. Root directory, checked by `config.RequireInit` |
| `HELMENV_VERSION` | Shell-level version, set by `helm-env shell` |
| `HELMENV_AUTO_INSTALL` | `1`/`true`/`yes`/`on`: install missing versions automatically |
| `HELMENV_CACHE_TTL` | Overrides the 1h release-cache TTL |
| `SHELL` | Fallback shell detection for `completion` |

Shell setup: `eval "$(helm-env init)"` for bash and zsh, `helm-env init | source` for fish.

## Gotchas

- **The shim duplicates the resolution logic in POSIX sh** (`internal/shim/shim.go`). If you change the precedence or version-file handling, update both the Go code and the shim.
- `site/` and `.venv-docs/` are local MkDocs build artifacts. Don't commit them. `docs-build` is strict, so broken links or anchors fail the build.
- The module path is the placeholder `github.com/user/helm-env`. The real repo is `github.com/mmpyro/helm-env`.
- `resolve --concrete` is a hidden, accepted no-op. Shims generated by older versions still call it, so don't remove it.
- With `HELMENV_AUTO_INSTALL` set, `resolve` installs the version itself. The shim discards its stderr, so the "auto-installing" notice isn't shown on that path.

## CI and release

- `.github/workflows/ci.yml` runs on push or PR to `main`:
  1. golangci-lint (latest).
  2. `gotestsum -- -race ./...`, with the JUnit report published by `dorny/test-reporter`.
  3. `make test-docker`.
- `.github/workflows/release.yml` runs on `v*.*.*` tags:
  1. `make build-all VERSION=<tag>`.
  2. BATS against the prebuilt linux amd64 and arm64 binaries using `Dockerfile.ci` (QEMU).
  3. Creates a **draft** GitHub release with the 4 binaries. It does not use goreleaser.
- Dependabot updates gomod weekly.
