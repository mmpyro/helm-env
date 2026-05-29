#!/usr/bin/env bats
# tests/integration.bats

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

@test "helm-env autocompletion outputs bash script" {
	run helm-env autocompletion
	assert_success
	assert_output --partial "_helm_env_completions()"
	assert_output --partial "complete -F _helm_env_completions helm-env"
}

@test "helm-env autocompletion --help shows help text" {
	run helm-env autocompletion --help
	assert_success
	assert_output --partial "autocompletion"
	assert_output --partial "source <(helm-env autocompletion)"
}
