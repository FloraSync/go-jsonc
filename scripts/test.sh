#!/usr/bin/env bash
# This file was modified by FloraSync in 2026.

# Run the standard-library-only test suite with the race detector enabled.
set -euo pipefail

export GOWORK=off
test "$(go list -m all)" = "github.com/FloraSync/go-jsonc"
go test -v -race ./...
