#!/usr/bin/env bash

set -e

GOBIN="$(pwd)/bin/go"
GOLANGCI_LINT_BIN="${GOBIN}/golangci-lint"
GOLANGCI_LINT_URL="github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
GOLANGCI_LINT_VERSION="${GOLANGCI_LINT_VERSION:-v2.15.0}"

function install_deps {
  local lint=""
  lint="$(which ${GOLANGCI_LINT_BIN})"

  if [[ -z "${lint}" || "v$(${lint} version --short)" != "${GOLANGCI_LINT_VERSION}" ]]; then
    echo "No 'golangci-lint' ${GOLANGCI_LINT_VERSION} tool. Installing..." >&2
    GOBIN=${GOBIN} go install "${GOLANGCI_LINT_URL}@${GOLANGCI_LINT_VERSION}"
  fi
}

function run_linters {
  ${GOLANGCI_LINT_BIN} --version
  ${GOLANGCI_LINT_BIN} run -v --color=always ./...
}

function main {
  go version
  install_deps
  run_linters
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main
fi
