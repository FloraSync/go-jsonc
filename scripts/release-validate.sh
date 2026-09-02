#!/bin/sh
set -eu

mode=${1:-}
case "$mode" in
  local|remote) ;;
  *) echo "usage: $0 local|remote" >&2; exit 2 ;;
esac

tag=${RELEASE_TAG:-}
sha=${RELEASE_SHA:-}

case "$tag" in
  v0.1.0|v0.1.1)
    echo "historical tag $tag cannot be reused" >&2
    exit 1
    ;;
  *+*)
    echo "build metadata is not allowed in a release tag" >&2
    exit 1
    ;;
esac

if ! printf '%s\n' "$tag" | grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$'; then
  echo "RELEASE_TAG must be a canonical v-prefixed SemVer tag" >&2
  exit 1
fi
prerelease=${tag#*-}
if [ "$prerelease" != "$tag" ]; then
  remaining=$prerelease
  while :; do
    identifier=${remaining%%.*}
    case "$identifier" in
      *[!0-9]* | 0 | [1-9]*) ;;
      *)
        echo "numeric SemVer prerelease identifiers cannot contain leading zeroes" >&2
        exit 1
        ;;
    esac
    case "$remaining" in
      *.*) remaining=${remaining#*.} ;;
      *) break ;;
    esac
  done
fi

if ! printf '%s\n' "$sha" | grep -Eq '^[0-9a-f]{40}$'; then
  echo "RELEASE_SHA must be a full lowercase commit SHA" >&2
  exit 1
fi

head_sha=$(git rev-parse HEAD)
if [ "$head_sha" != "$sha" ]; then
  echo "checked out $head_sha, expected $sha" >&2
  exit 1
fi

tag_object=$(git rev-parse --verify "refs/tags/$tag^{commit}" 2>/dev/null || true)
if [ "$tag_object" != "$sha" ]; then
  echo "tag $tag resolves to ${tag_object:-nothing}, expected $sha" >&2
  exit 1
fi

if [ "$mode" = remote ]; then
  clearance=${LEGAL_CLEARANCE_REF:-}
  case "$clearance" in
    *[![:space:]]*) ;;
    *) echo "LEGAL_CLEARANCE_REF is required" >&2; exit 1 ;;
  esac

  main_sha=$(git ls-remote --exit-code origin refs/heads/main | awk '{print $1}')
  remote_tag=$(git ls-remote --exit-code origin "refs/tags/$tag" | awk '{print $1}')
  remote_peeled=$(git ls-remote origin "refs/tags/$tag^{}" | awk '{print $1}')
  if [ -n "$remote_peeled" ]; then
    remote_tag=$remote_peeled
  fi
  if [ "$remote_tag" != "$sha" ]; then
    echo "remote tag $tag resolves to $remote_tag, expected $sha" >&2
    exit 1
  fi

  git fetch --no-tags origin "$main_sha"
  if ! git merge-base --is-ancestor "$sha" "$main_sha"; then
    echo "$sha is not an ancestor of remote main $main_sha" >&2
    exit 1
  fi

  if ! command -v gh >/dev/null 2>&1; then
    echo "gh is required for remote validation" >&2
    exit 1
  fi
  response=$(mktemp)
  trap 'rm -f "$response"' EXIT HUP INT TERM
  if gh api --silent --include \
    -H 'Accept: application/vnd.github+json' \
    "repos/${GITHUB_REPOSITORY:?}/releases/tags/$tag" >"$response" 2>/dev/null; then
    :
  else
    : # The HTTP status below decides whether a 404 is the expected failure.
  fi
  status=$(awk 'toupper($1) ~ /^HTTP\// {code=$2} END {print code}' "$response")
  case "$status" in
    200) echo "release $tag already exists" >&2; exit 1 ;;
    404) ;;
    *) echo "could not prove release $tag is unused (HTTP ${status:-unknown})" >&2; exit 1 ;;
  esac
fi

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  printf 'tag=%s\nsha=%s\n' "$tag" "$sha" >> "$GITHUB_OUTPUT"
fi

printf 'validated release candidate tag=%s sha=%s mode=%s\n' "$tag" "$sha" "$mode"
