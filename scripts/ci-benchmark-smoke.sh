#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
export GOWORK=off

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/go-jsonc-benchmark.XXXXXX")"
trap 'rm -rf "$work_dir"' EXIT
output="$work_dir/benchmark.out"

go test -mod=readonly -run='^$' -bench='.' -benchtime=1x -count=1 . | tee "$output"
if ! grep -q '^Benchmark' "$output"; then
  printf 'benchmark smoke matched no benchmarks\n' >&2
  exit 1
fi
