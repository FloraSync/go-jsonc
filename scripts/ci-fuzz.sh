#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
export GOWORK=off

mode=${1:-verify}
fuzz_time=${2:-${FUZZ_TIME:-1m}}
inventory="$repo_root/testdata/fuzz-targets.txt"

case "$mode" in
  verify | run) ;;
  *)
    printf 'usage: %s [verify|run [fuzztime]]\n' "$0" >&2
    exit 2
    ;;
esac

if [[ ! -s "$inventory" ]]; then
  printf 'fuzz target inventory is missing or empty: %s\n' "$inventory" >&2
  exit 1
fi

while IFS= read -r target || [[ -n "$target" ]]; do
  if [[ ! "$target" =~ ^Fuzz[A-Za-z0-9_]+$ ]]; then
    printf 'invalid fuzz target inventory entry: %q\n' "$target" >&2
    exit 1
  fi
done <"$inventory"

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/go-jsonc-fuzz.XXXXXX")"
trap 'rm -rf "$work_dir"' EXIT
actual="$work_dir/actual-targets.txt"
expected="$work_dir/expected-targets.txt"
listing="$work_dir/go-test-list.txt"

printf '%s\n' \
  FuzzSanitize \
  FuzzFacadeDifferential \
  FuzzDecoder >"$expected"
if ! diff -u "$expected" "$inventory"; then
  printf 'fuzz target inventory does not match the approved Phase 4 targets\n' >&2
  exit 1
fi

go test -mod=readonly -run='^$' -list='^Fuzz' . >"$listing"
awk '/^Fuzz[A-Za-z0-9_]+$/ { print }' "$listing" >"$actual"
if [[ ! -s "$actual" ]]; then
  printf 'go test discovered no fuzz targets\n' >&2
  exit 1
fi
if ! diff -u "$inventory" "$actual"; then
  printf 'fuzz target inventory does not match go test discovery\n' >&2
  exit 1
fi

case "$mode" in
  verify)
    printf 'verified %d fuzz targets\n' "$(wc -l <"$inventory" | tr -d ' ')"
    ;;
  run)
    while IFS= read -r target; do
      printf 'fuzzing %s for %s\n' "$target" "$fuzz_time"
      go test -mod=readonly -run='^$' -fuzz="^${target}$" \
        -fuzztime="$fuzz_time" -fuzzminimizetime=30s -parallel=1 -timeout=20m .
    done <"$inventory"
    ;;
esac
