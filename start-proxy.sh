#!/bin/bash

# Local development script for kubevirt-apiserver-proxy

set -euo pipefail

export APP_ENV=dev

KUBE_API_SERVER=$(oc config view --minify -o jsonpath='{.clusters[0].cluster.server}' | sed 's|https://||')

if [ -z "$KUBE_API_SERVER" ]; then
  echo "Error: Failed to get API server from oc config. Make sure you're logged in."
  exit 1
fi

export KUBE_API_SERVER

echo "Using API Server: $KUBE_API_SERVER"

WATCH=true
if [[ "${1:-}" == "--once" ]]; then
  WATCH=false
fi

run_once() {
  go build -o ./kubevirt-apiserver-proxy .
  ./kubevirt-apiserver-proxy
}

go_bin_dir() {
  local gobin
  gobin="$(go env GOBIN)"
  if [[ -n "$gobin" ]]; then
    echo "$gobin"
    return
  fi
  echo "$(go env GOPATH)/bin"
}

run_with_reload() {
  local bin_dir
  bin_dir="$(go_bin_dir)"

  AIR_BIN=""
  if command -v air >/dev/null 2>&1; then
    AIR_BIN="air"
  elif [[ -x "$bin_dir/air" ]]; then
    AIR_BIN="$bin_dir/air"
  else
    echo "Installing air for live reload..."
    if ! go install github.com/air-verse/air@latest; then
      echo "Failed to install air. Falling back to one-shot mode."
      run_once
      return
    fi
    if [[ -x "$bin_dir/air" ]]; then
      AIR_BIN="$bin_dir/air"
    fi
  fi

  if [[ -z "$AIR_BIN" ]]; then
    echo "Error: air is not on PATH. Add \"$(go_bin_dir)\" to your PATH."
    echo "Falling back to one-shot mode. Use ./start-proxy.sh --once explicitly next time."
    run_once
    return
  fi

  echo "Watching Go files for changes (Ctrl+C to stop). Use --once to disable reload."
  "$AIR_BIN"
}

if [[ "$WATCH" == "true" ]]; then
  run_with_reload
else
  run_once
fi
