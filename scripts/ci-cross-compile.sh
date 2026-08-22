#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
export GOWORK=off

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/go-jsonc-cross.XXXXXX")"
trap 'rm -rf "$work_dir"' EXIT

targets=(
  linux/amd64
  linux/arm64
  darwin/amd64
  darwin/arm64
  windows/amd64
  windows/arm64
)
if [[ ${#targets[@]} -ne 6 ]]; then
  printf 'cross-compile target inventory must contain exactly six targets\n' >&2
  exit 1
fi

mapfile_support=false
if builtin help mapfile >/dev/null 2>&1; then
  mapfile_support=true
fi
if [[ "$mapfile_support" == true ]]; then
  mapfile -t packages < <(go list ./...)
else
  packages=()
  while IFS= read -r package; do
    packages+=("$package")
  done < <(go list ./...)
fi
if [[ ${#packages[@]} -eq 0 ]]; then
  printf 'go list discovered no packages\n' >&2
  exit 1
fi

for target in "${targets[@]}"; do
  goos=${target%/*}
  goarch=${target#*/}
  for index in "${!packages[@]}"; do
    suffix=''
    if [[ "$goos" == windows ]]; then
      suffix='.exe'
    fi
    output="$work_dir/${goos}-${goarch}-${index}.test${suffix}"
    printf 'compiling %s for %s/%s with CGO_ENABLED=0\n' "${packages[$index]}" "$goos" "$goarch"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
      go test -mod=readonly -run='^$' -c -o "$output" "${packages[$index]}"
    test -s "$output"
  done
done
