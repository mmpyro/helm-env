#!/usr/bin/env bats
# tests/integration.bats

bats_require_minimum_version 1.5.0

setup() {
	# Load helper libraries
	load '/usr/local/lib/bats-support/load'
	load '/usr/local/lib/bats-assert/load'

	# Set up a clean environment for each test
	export HELMENV_ROOT="${BATS_TMPDIR}/helmenv"
	export PATH="${HELMENV_ROOT}/shims:${PATH}"
	mkdir -p "${HELMENV_ROOT}"
}

teardown() {
	# Clean up after each test
	rm -rf "${HELMENV_ROOT}"
}

# fake_helm VERSION [EXIT_CODE] installs a stub helm binary that echoes its
# arguments and exits with EXIT_CODE. helm-env treats any versions/<v>/helm
# as installed, so this avoids extra downloads.
fake_helm() {
	local ver="$1" code="${2:-0}"
	mkdir -p "${HELMENV_ROOT}/versions/${ver}"
	printf '#!/bin/sh\necho "fake helm %s: $*"\nexit %s\n' "${ver}" "${code}" \
		>"${HELMENV_ROOT}/versions/${ver}/helm"
	chmod +x "${HELMENV_ROOT}/versions/${ver}/helm"
}

@test "helm-env init creates necessary directories and shims" {
	run helm-env init
	assert_success
	[ -d "${HELMENV_ROOT}/versions" ]
	[ -f "${HELMENV_ROOT}/shims/helm" ]
}

@test "helm-env list-remote returns a list of versions" {
	helm-env init
	run helm-env list-remote
	assert_success
	assert_output --regexp "[0-9]+\.[0-9]+\.[0-9]+"
}

@test "helm-env latest returns a stable version" {
	helm-env init
	run helm-env latest
	assert_success
	assert_output --regexp "[0-9]+\.[0-9]+\.[0-9]+$"
}

@test "helm-env latest --prerelease returns a version" {
	helm-env init
	run helm-env latest --prerelease
	assert_success
	assert_output --regexp "[0-9]+\.[0-9]+\.[0-9]+"
}

@test "helm-env latest --help shows help text" {
	run helm-env latest --help
	assert_success
	assert_output --partial "latest"
	assert_output --partial "prerelease"
}

@test "helm-env install/list/global/which flow" {
	helm-env init
	local test_ver="3.14.0"

	# Install
	run helm-env install "${test_ver}"
	assert_success
	[ -f "${HELMENV_ROOT}/versions/${test_ver}/helm" ]

	# List
	run helm-env list
	assert_success
	assert_output --partial "${test_ver}"

	# Global
	run helm-env global "${test_ver}"
	assert_success

	run helm-env global
	assert_success
	assert_output --partial "${test_ver}"

	# Which
	run helm-env which
	assert_success
	assert_output --partial "${HELMENV_ROOT}/versions/${test_ver}/helm"
}

@test "helm-env local sets directory-specific version" {
	helm-env init
	local test_ver="3.14.0"

	# Must install it first because setup() wipes HELMENV_ROOT
	helm-env install "${test_ver}"

	local work_dir="${BATS_TMPDIR}/work"
	mkdir -p "${work_dir}"

	pushd "${work_dir}"
	run helm-env local "${test_ver}"
	assert_success
	[ -f ".helm-version" ]
	grep -q "${test_ver}" ".helm-version"

	run helm-env local
	assert_success
	assert_output --partial "${test_ver}"
	popd

	rm -rf "${work_dir}"
}

@test "helm-env shell outputs export command" {
	helm-env init
	local test_ver="3.14.0"
	helm-env install "${test_ver}"
	run helm-env shell "${test_ver}"
	assert_success
	assert_output --partial "export HELMENV_VERSION=${test_ver}"
}

@test "helm-env uninstall removes version" {
	helm-env init
	local test_ver="3.14.0"
	# Pre-install
	run helm-env install "${test_ver}"
	assert_success

	# Uninstall
	run helm-env uninstall "${test_ver}"
	assert_success
	[ ! -d "${HELMENV_ROOT}/versions/${test_ver}" ]

	run helm-env list
	assert_success
	refute_output --partial "${test_ver}"
}

@test "helm-env version" {
	run helm-env version
	assert_success
	assert_output --regexp "[0-9]+\.[0-9]+\.[0-9]+"
}

@test "helm-env help" {
	run helm-env help
	assert_success
	assert_output --partial "Commands:"
}

@test "helm-env status shows environment information" {
	helm-env init
	local test_ver="3.14.0"
	helm-env install "${test_ver}"
	helm-env global "${test_ver}"

	run helm-env status
	assert_success
	assert_output --partial "HELMENV_ROOT"
	assert_output --regexp "Active version:[[:space:]]+${test_ver}"
	assert_output --partial "set by global version file"
	assert_output --partial "* ${test_ver}"
}

@test "helm-env exec runs specific version" {
	helm-env init
	local test_ver="3.14.0"
	helm-env install "${test_ver}"

	# Run a command via exec
	run helm-env exec "${test_ver}" version
	assert_success
	assert_output --regexp "([vV]ersion|[0-9]+\.[0-9]+\.[0-9])"
}

@test "helm shim end-to-end" {
	helm-env init
	local test_ver="3.14.0"

	helm-env install "${test_ver}"
	helm-env global "${test_ver}"

	# Ensure shim is in PATH
	run helm version
	assert_output --regexp "([vV]ersion|[0-9]+\.[0-9]+\.[0-9])"
}

@test "helm-env completion with no shell falls back to bash" {
	SHELL= run helm-env completion
	assert_success
	assert_output --partial "_helm_env_completions()"
	assert_output --partial "complete -F _helm_env_completions helm-env"
}

@test "helm-env completion generates every shell, positional and --shell" {
	local sh marker
	for sh in bash zsh fish powershell pwsh; do
		case "${sh}" in
		bash) marker="complete -F _helm_env_completions helm-env" ;;
		zsh) marker="#compdef helm-env" ;;
		fish) marker="complete -c helm-env" ;;
		powershell | pwsh) marker="Register-ArgumentCompleter" ;;
		esac

		run helm-env completion "${sh}"
		assert_success
		assert_output --partial "${marker}"

		run helm-env completion --shell "${sh}"
		assert_success
		assert_output --partial "${marker}"
	done
}

@test "helm-env completion scripts complete shell names after completion" {
	run helm-env completion bash
	assert_success
	assert_output --partial "bash zsh fish powershell pwsh --shell"
	refute_output --partial "autocompletion"
}

@test "helm-env completion rejects an unknown shell" {
	run helm-env completion tcsh
	assert_failure
	assert_output --partial "unsupported shell"
}

@test "helm-env completion --help shows help text" {
	run helm-env completion --help
	assert_success
	assert_output --partial "Usage: helm-env completion [SHELL] [--shell SHELL]"
	assert_output --partial "source <(helm-env completion bash)"
}

@test "helm-env autocompletion is a deprecated alias that warns on stderr" {
	run --separate-stderr helm-env autocompletion --shell zsh
	assert_success
	assert_output --partial "#compdef helm-env"
	[ "${stderr}" = "helm-env: 'autocompletion' is deprecated; use 'helm-env completion'" ]

	run helm-env help
	refute_output --partial "autocompletion"
}

@test "helm-env upgrade --help prints help and does not upgrade" {
	local bin before
	bin="$(command -v helm-env)"
	before="$(sha256sum "${bin}")"

	for flag in --help -h; do
		run helm-env upgrade "${flag}"
		assert_success
		assert_output --partial "Usage: helm-env upgrade"
		refute_output --partial "Downloading"
		refute_output --partial "Upgraded"
	done

	[ "$(sha256sum "${bin}")" = "${before}" ]
}

@test "helm-env -h short-circuits uninstall, shell, local, global and exec" {
	helm-env init
	cd "${BATS_TEST_TMPDIR}"
	local cmd
	for cmd in uninstall shell local global exec resolve prune doctor; do
		run helm-env "${cmd}" -h
		assert_success
		assert_output --partial "Usage: helm-env ${cmd}"
	done
	# -h must not have been taken as a version.
	[ ! -f .helm-version ]
	[ ! -f "${HELMENV_ROOT}/version" ]
}

@test "helm-env resolve with no argument prints the active global version" {
	helm-env init
	fake_helm 3.14.0
	helm-env global 3.14.0

	run --separate-stderr helm-env resolve
	assert_success
	assert_output "3.14.0"
}

@test "helm-env resolve resolves an explicit constraint against installed versions" {
	helm-env init
	fake_helm 3.13.2
	fake_helm 3.14.1
	fake_helm 3.14.3
	helm-env global 3.13.2

	run --separate-stderr helm-env resolve "~3.14"
	assert_success
	assert_output "3.14.3"

	# legacy --concrete flag is still accepted (older shims pass it)
	run --separate-stderr helm-env resolve --concrete
	assert_success
	assert_output "3.13.2"
}

@test "helm-env resolve fails for an unresolved spec" {
	helm-env init
	fake_helm 3.14.0
	HELMENV_AUTO_INSTALL= run helm-env resolve 9.9.9
	assert_failure
	assert_output --partial "9.9.9"
}

@test "helm-env resolve --install installs a missing version" {
	helm-env init
	run --separate-stderr helm-env resolve --install --silent 3.14.0
	assert_success
	assert_output "3.14.0"
	[ -x "${HELMENV_ROOT}/versions/3.14.0/helm" ]
}

@test "helm-env exec strips a leading -- and runs the real helm" {
	helm-env init
	helm-env install --silent 3.14.0

	run helm-env exec 3.14.0 -- version --short
	assert_success
	assert_output --partial "v3.14.0"
}

@test "helm-env exec resolves a constraint spec" {
	helm-env init
	fake_helm 3.14.1
	fake_helm 3.14.3

	run helm-env exec "~3.14" -- status x
	assert_success
	assert_output "fake helm 3.14.3: status x"
}

@test "helm-env exec propagates the child exit code" {
	helm-env init
	fake_helm 3.14.0 3

	run helm-env exec 3.14.0 version
	[ "${status}" -eq 3 ]
	assert_output --partial "fake helm 3.14.0: version"
	refute_output --partial "exit status"
}

@test "helm-env exec propagates a failing real helm subcommand" {
	helm-env init
	helm-env install --silent 3.14.0

	run helm-env exec 3.14.0 -- no-such-subcommand
	[ "${status}" -ne 0 ]
	refute_output --partial "exit status"
}

@test "helm-env exec -h shows help" {
	run helm-env exec -h
	assert_success
	assert_output --partial "Usage: helm-env exec [--auto|--no-auto] <spec> [--]"
}

@test "helm-env prune is a dry run by default" {
	helm-env init
	fake_helm 3.12.0
	fake_helm 3.13.0
	fake_helm 3.14.0
	helm-env global 3.14.0

	run helm-env prune --keep-last 1
	assert_success
	assert_output --partial "[dry-run] would prune 3.13.0"
	assert_output --partial "[dry-run] would prune 3.12.0"
	assert_output --partial "--yes"
	[ -d "${HELMENV_ROOT}/versions/3.12.0" ]
	[ -d "${HELMENV_ROOT}/versions/3.13.0" ]
}

@test "helm-env prune --yes removes versions" {
	helm-env init
	fake_helm 3.12.0
	fake_helm 3.13.0
	fake_helm 3.14.0
	helm-env global 3.14.0

	run helm-env prune --keep-last 1 --yes
	assert_success
	assert_output --partial "pruning 3.12.0"
	[ ! -d "${HELMENV_ROOT}/versions/3.12.0" ]
	[ ! -d "${HELMENV_ROOT}/versions/3.13.0" ]
	[ -d "${HELMENV_ROOT}/versions/3.14.0" ]
}

@test "helm-env prune --keep is a deprecated alias for --keep-last" {
	helm-env init
	fake_helm 3.13.0
	fake_helm 3.14.0
	helm-env global 3.14.0

	run helm-env prune --keep 1 --yes
	assert_success
	[ ! -d "${HELMENV_ROOT}/versions/3.13.0" ]
}

@test "helm-env prune keeps the version referenced by .helm-version" {
	helm-env init
	fake_helm 3.12.0
	fake_helm 3.13.0
	fake_helm 3.14.0
	helm-env global 3.14.0

	cd "${BATS_TEST_TMPDIR}"
	echo "3.12.0" >.helm-version

	run helm-env prune --keep-last 1 --yes
	assert_success
	assert_output --partial "refusing to prune active version 3.12.0"
	[ -d "${HELMENV_ROOT}/versions/3.12.0" ]
	[ ! -d "${HELMENV_ROOT}/versions/3.13.0" ]
}

@test "helm-env prune --older-than bogus fails" {
	helm-env init
	run helm-env prune --older-than bogus
	assert_failure
	assert_output --partial "invalid --older-than"
}

@test "helm-env doctor succeeds after init with shims on PATH" {
	helm-env init
	run helm-env doctor
	assert_success
	refute_output --partial "[FAIL]"
	assert_output --partial "[OK] \$HELMENV_ROOT/shims/helm exists and is executable"
}

@test "helm-env doctor fails when shims are not on PATH" {
	helm-env init
	local bindir
	bindir="$(dirname "$(command -v helm-env)")"
	run env PATH="${bindir}:/usr/bin:/bin" helm-env doctor
	assert_failure
	assert_output --partial "[FAIL]"
	assert_output --partial "not on PATH"
}

@test "helm-env doctor --fix regenerates a tampered shim" {
	helm-env init
	echo "tampered" >"${HELMENV_ROOT}/shims/helm"

	run helm-env doctor
	assert_failure
	assert_output --partial "[FAIL]"

	run helm-env doctor --fix
	assert_success
	assert_output --partial "[FIX] regenerated"
	run grep -q "tampered" "${HELMENV_ROOT}/shims/helm"
	assert_failure
}
