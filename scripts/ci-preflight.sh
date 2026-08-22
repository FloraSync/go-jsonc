#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
export GOWORK=off

for tool in actionlint golangci-lint govulncheck; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    printf 'required CI tool is missing: %s\n' "$tool" >&2
    exit 1
  fi
done

actionlint_version="$(actionlint -version 2>&1)"
if [[ "${actionlint_version%%$'\n'*}" != 'v1.7.12' ]]; then
  printf 'actionlint version mismatch; require v1.7.12, got:\n%s\n' "$actionlint_version" >&2
  exit 1
fi
printf '%s\n' "$actionlint_version"

golangci_version="$(golangci-lint version 2>&1)"
if [[ "$golangci_version" != *'version 2.12.0 '* ]]; then
  printf 'golangci-lint version mismatch; require 2.12.0, got:\n%s\n' "$golangci_version" >&2
  exit 1
fi
printf '%s\n' "$golangci_version"

govulncheck_version="$(govulncheck -version 2>&1)"
if [[ "$govulncheck_version" != *'Scanner: govulncheck@v1.7.0'* ]]; then
  printf 'govulncheck version mismatch; require v1.7.0, got:\n%s\n' "$govulncheck_version" >&2
  exit 1
fi
printf '%s\n' "$govulncheck_version"

unformatted="$(gofmt -l .)"
if [[ -n "$unformatted" ]]; then
  printf 'gofmt required for:\n%s\n' "$unformatted" >&2
  exit 1
fi

actionlint .github/workflows/ci.yml
go vet ./...
golangci-lint run
govulncheck ./...
go mod tidy -diff
go mod verify

module_files=''
legacy_backend_paths=''
legacy_backend_refs=''
while IFS= read -r path; do
  [[ -f "$path" ]] || continue
  case "$path" in
    go.mod | */go.mod)
      module_files+="${module_files:+$'\n'}$path"
      ;;
  esac
  case "$path" in
    internal/json/go_json.go | internal/json/jsoniter.go | internal/json/std.go)
      legacy_backend_paths+="${legacy_backend_paths:+$'\n'}$path"
      ;;
  esac
  case "$path" in
    *.go | go.mod | go.sum | .github/workflows/*.yml | .github/workflows/*.yaml)
      matches="$(grep -nE 'jsoniter|go_json|github\.com/json-iterator/go|github\.com/goccy/go-json' "$path" || true)"
      if [[ -n "$matches" ]]; then
        legacy_backend_refs+="${legacy_backend_refs:+$'\n'}$path:$matches"
      fi
      ;;
  esac
done < <(git ls-files --cached --others --exclude-standard)

if [[ "$module_files" != 'go.mod' ]]; then
  printf 'expected exactly one root go.mod, found:\n%s\n' "${module_files:-<none>}" >&2
  exit 1
fi
if [[ -n "$legacy_backend_paths" || -n "$legacy_backend_refs" ]]; then
  printf 'retired JSON backend source, tags, axes, or dependencies detected:\n%s\n%s\n' \
    "$legacy_backend_paths" "$legacy_backend_refs" >&2
  exit 1
fi

modules="$(go list -m all)"
if [[ "$modules" != 'github.com/FloraSync/go-jsonc' ]]; then
  printf 'unexpected module graph:\n%s\n' "$modules" >&2
  exit 1
fi

"$repo_root/scripts/ci-fuzz.sh" verify
test_inventory="$(go test -mod=readonly -run='^$' -list='^(Test|Example)' .)"
test_count=0
while IFS= read -r name; do
  case "$name" in
    Test* | Example*)
      ((test_count += 1))
      ;;
  esac
done <<<"$test_inventory"
if ((test_count == 0)); then
  printf 'ordinary test inventory is empty\n' >&2
  exit 1
fi
printf 'verified %d ordinary tests/examples\n' "$test_count"
go test -mod=readonly -count=1 ./...
