#!/usr/bin/env bash

set -e

GOBIN="$(pwd)/bin/go"
GOIMPORTS_BIN="${GOBIN}/goimports"
GOIMPORTS_URL="golang.org/x/tools/cmd/goimports"
GOIMPORTS_VERSION="${GOIMPORTS_VERSION:-v0.52.0}"

function install_deps {
  local installed_version=""

  if [[ -x "${GOIMPORTS_BIN}" ]]; then
    installed_version=$(
      go version -m "${GOIMPORTS_BIN}" |
        awk '$1 == "mod" && $2 == "golang.org/x/tools" {print $3}'
    )
  fi

  if [[ "${installed_version}" != "${GOIMPORTS_VERSION}" ]]; then
    echo "No 'goimports' ${GOIMPORTS_VERSION} tool. Installing..." >&2
    mkdir -p "${GOBIN}"
    GOBIN="${GOBIN}" go install "${GOIMPORTS_URL}@${GOIMPORTS_VERSION}"
  fi
}

function run_goimports {
  "${GOIMPORTS_BIN}" -w .
}

function main {
  install_deps
  run_goimports
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main
fi