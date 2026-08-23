#!/bin/sh
set -eu

export GOWORK=off

if ! command -v goreleaser >/dev/null 2>&1; then
  echo "goreleaser v2.12.7 is required" >&2
  exit 1
fi
version=$(goreleaser --version 2>/dev/null | awk '/GitVersion:/ {print $2; exit}')
if [ "$version" != "2.12.7" ]; then
  echo "expected goreleaser 2.12.7, found ${version:-unknown}" >&2
  exit 1
fi

rm -rf dist
goreleaser release --snapshot --clean --skip=publish

if find dist -type f \( -name '*.exe' -o -name '*.dll' -o -name '*.so' -o -name '*.dylib' -o -perm -111 \) -print -quit | grep -q .; then
  echo "source-only release unexpectedly produced a binary" >&2
  exit 1
fi

metadata=dist/metadata.json
if [ ! -s "$metadata" ]; then
  echo "snapshot did not produce metadata" >&2
  exit 1
fi

for path in LICENSE README.md go.mod; do
  if [ ! -f "$path" ]; then
    echo "required distribution file is missing: $path" >&2
    exit 1
  fi
done

if [ "$(awk '$1 == "module" {print $2}' go.mod)" != "github.com/FloraSync/go-jsonc" ]; then
  echo "unexpected module path" >&2
  exit 1
fi
if [ "$(awk '$1 == "go" {print $2}' go.mod)" != "1.26.0" ]; then
  echo "unexpected Go baseline" >&2
  exit 1
fi

find dist -type f -print | LC_ALL=C sort
