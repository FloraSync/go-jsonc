#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
export GOWORK=off

minimum_tenths=${COVERAGE_MIN_TENTHS:-900}
if [[ ! "$minimum_tenths" =~ ^[0-9]+$ ]]; then
  printf 'COVERAGE_MIN_TENTHS must be a non-negative integer\n' >&2
  exit 2
fi

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/go-jsonc-coverage.XXXXXX")"
trap 'rm -rf "$work_dir"' EXIT
profile="$work_dir/cover.out"

go test -mod=readonly -count=1 ./... \
  -coverpkg=./... \
  -covermode=atomic \
  -coverprofile="$profile"

if [[ ! -s "$profile" ]]; then
  printf 'coverage profile is missing or empty: %s\n' "$profile" >&2
  exit 1
fi

IFS= read -r profile_header <"$profile"
if [[ "$profile_header" != 'mode: atomic' ]]; then
  printf 'coverage profile mode is not atomic: %s\n' "$profile_header" >&2
  exit 1
fi

awk -v minimum="$minimum_tenths" '
  NR == 1 { next }
  {
    if (NF != 3 || $2 !~ /^[0-9]+$/ || $3 !~ /^[0-9]+$/) {
      printf "malformed coverage record at line %d: %s\n", NR, $0 > "/dev/stderr"
      invalid = 1
      next
    }
    total += $2
    if ($3 > 0) {
      covered += $2
    }
  }
  END {
    if (invalid) {
      exit 1
    }
    if (total == 0) {
      print "coverage profile contains no statements" > "/dev/stderr"
      exit 1
    }
    thousandths = int(covered * 100000 / total)
    printf "coverage: %d/%d statements = %d.%03d%% (minimum %d.%d%%)\n", covered, total, int(thousandths / 1000), thousandths % 1000, int(minimum / 10), minimum % 10
    if (covered * 1000 < minimum * total) {
      exit 1
    }
  }
' "$profile"

output=${COVERAGE_OUTPUT:-cover.out}
cp "$profile" "$output"
test -s "$output"
