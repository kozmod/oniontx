#!/usr/bin/env bash

set -e

GOBIN="$(pwd)/bin/go"
PKGSITE_BIN="${GOBIN}/pkgsite"
PKGSITE_LOCAL_PORT="${PKGSITE_LOCAL_PORT:-:6060}"

function install_deps {
  local update="${1:-false}"

  if [[ "${update}" == "false" && -x "${PKGSITE_BIN}" ]]; then
    return
  fi

  echo "Installing latest 'pkgsite'..." >&2
  mkdir -p "${GOBIN}"
  GOBIN="${GOBIN}" go install "golang.org/x/pkgsite/cmd/pkgsite@latest"
}

function run_godoc {
  echo "Starting 'pkgsite' on [${PKGSITE_LOCAL_PORT}]"
  exec "${PKGSITE_BIN}" -http="${PKGSITE_LOCAL_PORT}" .
}

function main {
  local update=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --update)
        update=true
        ;;
      *)
        echo "Usage: $0 [update]" >&2
        return 1
        ;;
    esac
    shift
  done

  go version
  install_deps "${update}"
  run_godoc
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi
