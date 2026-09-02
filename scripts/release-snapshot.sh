#!/bin/sh
set -eu

export GOWORK=off
repo_root=$(CDPATH='' cd -- "$(dirname "$0")/.." && pwd)
cd "$repo_root"

if ! command -v goreleaser >/dev/null 2>&1; then
  echo "goreleaser v2.17.1 is required" >&2
  exit 1
fi
version=$(goreleaser --version 2>/dev/null | awk '/GitVersion:/ {print $2; exit}')
if [ "$version" != "v2.17.1" ]; then
  echo "expected goreleaser v2.17.1, found ${version:-unknown}" >&2
  exit 1
fi

"$repo_root/scripts/release-validate.sh" local
goreleaser release --snapshot --clean

# Adding root dotfiles to the source archive makes GoReleaser retain its
# pre-injection archive as an implementation-detail backup. It is not a release
# artifact, so remove it before validating the publishable output inventory.
find dist -maxdepth 1 -type f -name '*.bkp' -delete

if find dist -type f \( -name '*.exe' -o -name '*.dll' -o -name '*.so' -o -name '*.dylib' -o -perm -111 \) -print -quit | grep -q .; then
  echo "source-only release unexpectedly produced a binary" >&2
  exit 1
fi

metadata=dist/metadata.json
if [ ! -s "$metadata" ]; then
  echo "snapshot did not produce metadata" >&2
  exit 1
fi
artifacts=dist/artifacts.json
if [ ! -s "$artifacts" ]; then
  echo "snapshot did not produce an artifact manifest" >&2
  exit 1
fi
if ! grep -Eq '"type"[[:space:]]*:[[:space:]]*"Source"' "$artifacts"; then
  echo "snapshot did not record a source archive artifact" >&2
  exit 1
fi

source_archives=$(find dist -maxdepth 1 -type f -name '*.tar.gz' -print)
if [ "$(printf '%s\n' "$source_archives" | sed '/^$/d' | wc -l | tr -d ' ')" -ne 1 ]; then
  echo "snapshot must produce exactly one source archive" >&2
  printf '%s\n' "$source_archives" >&2
  exit 1
fi
source_archive=$source_archives

for output in dist/*; do
  if [ ! -f "$output" ]; then
    echo "snapshot produced an unexpected output: $output" >&2
    exit 1
  fi
  case "$output" in
    dist/artifacts.json | dist/checksums.txt | dist/config.yaml | dist/metadata.json | "$source_archive") ;;
    *) echo "snapshot produced an unexpected output: $output" >&2; exit 1 ;;
  esac
done

checksums=dist/checksums.txt
if [ ! -s "$checksums" ]; then
  echo "snapshot did not produce checksums" >&2
  exit 1
fi
if ! grep -F "  $(basename "$source_archive")" "$checksums" >/dev/null; then
  echo "source archive is missing from checksums" >&2
  exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  (cd dist && sha256sum --check checksums.txt)
elif command -v shasum >/dev/null 2>&1; then
  (cd dist && shasum -a 256 --check checksums.txt)
else
  echo "sha256sum or shasum is required" >&2
  exit 1
fi

work_dir=$(mktemp -d "${TMPDIR:-/tmp}/go-jsonc-release-snapshot.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT HUP INT TERM
tar -xzf "$source_archive" -C "$work_dir"
archive_roots=$(find "$work_dir" -mindepth 1 -maxdepth 1 -type d -print)
if [ "$(printf '%s\n' "$archive_roots" | sed '/^$/d' | wc -l | tr -d ' ')" -ne 1 ]; then
  echo "source archive must contain one top-level directory" >&2
  exit 1
fi
archive_root=$archive_roots
case "$(basename "$archive_root")" in
  go-jsonc-*) ;;
  *) echo "unexpected source archive prefix: $(basename "$archive_root")" >&2; exit 1 ;;
esac

actual_files=$work_dir/actual-files.txt
expected_files=$work_dir/expected-files.txt
find "$archive_root" -type f -print | sed "s#^$archive_root/##" | LC_ALL=C sort >"$actual_files"
git ls-tree -r --name-only HEAD | LC_ALL=C sort >"$expected_files"
if ! diff -u "$expected_files" "$actual_files"; then
  echo "source archive does not exactly match the tagged revision" >&2
  exit 1
fi

for path in LICENSE README.md go.mod; do
  if [ ! -f "$archive_root/$path" ]; then
    echo "required distribution file is missing: $path" >&2
    exit 1
  fi
done
if ! find "$archive_root/docs" -type f -name '*.md' -print -quit | grep -q .; then
  echo "source archive contains no documentation" >&2
  exit 1
fi

if [ "$(awk '$1 == "module" {print $2}' "$archive_root/go.mod")" != "github.com/FloraSync/go-jsonc" ]; then
  echo "unexpected module path" >&2
  exit 1
fi
if [ "$(awk '$1 == "go" {print $2}' "$archive_root/go.mod")" != "1.26.0" ]; then
  echo "unexpected Go baseline" >&2
  exit 1
fi

printf 'verified source archive %s against commit %s\n' \
  "$(basename "$source_archive")" "$(git rev-parse HEAD)"
find dist -type f -print | LC_ALL=C sort
