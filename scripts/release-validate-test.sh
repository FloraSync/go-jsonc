#!/bin/sh
set -eu

repo_root=$(CDPATH='' cd -- "$(dirname "$0")/.." && pwd)
validator=$repo_root/scripts/release-validate.sh
work_dir=$(mktemp -d "${TMPDIR:-/tmp}/go-jsonc-release-validate.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT HUP INT TERM

checks=0
expect_pass() {
  name=$1
  shift
  if output=$("$@" 2>&1); then
    checks=$((checks + 1))
    printf 'ok %d - %s\n' "$checks" "$name"
    return
  fi
  printf 'not ok - %s\n%s\n' "$name" "$output" >&2
  exit 1
}

expect_fail() {
  name=$1
  expected=$2
  shift 2
  if output=$("$@" 2>&1); then
    printf 'not ok - %s unexpectedly passed\n%s\n' "$name" "$output" >&2
    exit 1
  fi
  case "$output" in
    *"$expected"*) ;;
    *)
      printf 'not ok - %s returned the wrong failure\n%s\n' "$name" "$output" >&2
      exit 1
      ;;
  esac
  checks=$((checks + 1))
  printf 'ok %d - %s\n' "$checks" "$name"
}

origin=$work_dir/origin.git
seed=$work_dir/seed
candidate=$work_dir/candidate
fake_bin=$work_dir/bin

git init --bare -q "$origin"
git init -q "$seed"
git -C "$seed" config user.name 'Release Test'
git -C "$seed" config user.email 'release-test@example.invalid'
printf 'release validation fixture\n' >"$seed/fixture.txt"
mkdir -p "$seed/docs/releases"
for release_tag in v0.2.0 v0.2.0-rc.1 v0.3.0 v9.0.0; do
  printf '# %s test notes\n' "$release_tag" >"$seed/docs/releases/$release_tag.md"
done
git -C "$seed" add fixture.txt docs/releases
git -C "$seed" commit -q -m 'initial candidate'
git -C "$seed" branch -M main
main_sha=$(git -C "$seed" rev-parse HEAD)
git -C "$seed" tag v0.2.0
git -C "$seed" tag v0.2.0-rc.1
git -C "$seed" remote add origin "$origin"
git -C "$seed" push -q origin main refs/tags/v0.2.0 refs/tags/v0.2.0-rc.1
git --git-dir="$origin" symbolic-ref HEAD refs/heads/main

git -C "$seed" switch -q -c side
printf 'non-main candidate\n' >>"$seed/fixture.txt"
git -C "$seed" commit -q -am 'side candidate'
side_sha=$(git -C "$seed" rev-parse HEAD)
git -C "$seed" tag v0.3.0
git -C "$seed" push -q origin refs/tags/v0.3.0
git -C "$seed" switch -q main

git clone -q "$origin" "$candidate"
git -C "$candidate" checkout -q "$main_sha"

mkdir -p "$fake_bin"
fake_gh=$fake_bin/gh
# Keep parameter expansion literal while generating the disposable fake.
# shellcheck disable=SC2016
printf '%s\n' \
  '#!/bin/sh' \
  'status=${FAKE_GH_STATUS:-404}' \
  'printf "HTTP/2 %s test\\r\\n\\r\\n" "$status"' \
  '[ "$status" -lt 400 ]' >"$fake_gh"
chmod +x "$fake_gh"

validate_local() {
  test_tag=$1
  test_sha=$2
  (
    cd "$candidate"
    RELEASE_TAG=$test_tag RELEASE_SHA=$test_sha "$validator" local
  )
}

validate_remote() {
  test_tag=$1
  test_sha=$2
  clearance=$3
  status=$4
  (
    cd "$candidate"
    PATH="$fake_bin:$PATH" \
      RELEASE_TAG=$test_tag \
      RELEASE_SHA=$test_sha \
      LEGAL_CLEARANCE_REF=$clearance \
      GITHUB_REPOSITORY=FloraSync/go-jsonc \
      FAKE_GH_STATUS=$status \
      "$validator" remote
  )
}

expect_pass 'exact local tag and commit' validate_local v0.2.0 "$main_sha"
expect_pass 'canonical prerelease tag' validate_local v0.2.0-rc.1 "$main_sha"
expect_fail 'malformed tag' 'canonical v-prefixed SemVer' validate_local v01.2.3 "$main_sha"
expect_fail 'numeric prerelease leading zero' 'prerelease identifiers cannot contain leading zeroes' \
  validate_local v0.2.0-01 "$main_sha"
expect_fail 'build metadata' 'build metadata is not allowed' validate_local v0.2.0+build.1 "$main_sha"
expect_fail 'historical tag reuse' 'historical tag v0.1.0 cannot be reused' validate_local v0.1.0 "$main_sha"
expect_fail 'abbreviated commit' 'full lowercase commit SHA' validate_local v0.2.0 "${main_sha%????????}"
expect_fail 'cross-SHA checkout' "checked out $main_sha, expected $side_sha" validate_local v0.2.0 "$side_sha"
expect_fail 'missing tag' 'tag v9.0.0 resolves to nothing' validate_local v9.0.0 "$main_sha"
expect_fail 'missing release notes' 'release notes are missing or empty' validate_local v0.4.0 "$main_sha"
expect_pass 'unused remote tag on main' validate_remote v0.2.0 "$main_sha" clearance-123 404
expect_fail 'blank legal clearance' 'LEGAL_CLEARANCE_REF is required' validate_remote v0.2.0 "$main_sha" '   ' 404
expect_fail 'already-published release' 'release v0.2.0 already exists' validate_remote v0.2.0 "$main_sha" clearance-123 200
expect_fail 'indeterminate release lookup' 'could not prove release v0.2.0 is unused' validate_remote v0.2.0 "$main_sha" clearance-123 500

git -C "$candidate" checkout -q "$side_sha"
expect_fail 'tag outside main history' "$side_sha is not an ancestor of remote main $main_sha" \
  validate_remote v0.3.0 "$side_sha" clearance-123 404
git -C "$candidate" checkout -q "$main_sha"

git -C "$seed" tag -f v0.2.0 "$side_sha" >/dev/null
git -C "$seed" push -q --force origin refs/tags/v0.2.0
expect_fail 'moved remote tag' "remote tag v0.2.0 resolves to $side_sha" \
  validate_remote v0.2.0 "$main_sha" clearance-123 404
git -C "$seed" tag -f v0.2.0 "$main_sha" >/dev/null
git -C "$seed" push -q --force origin refs/tags/v0.2.0

before_refs=$(git --git-dir="$origin" show-ref)
expect_pass 'first verification retry' validate_remote v0.2.0 "$main_sha" clearance-123 404
expect_pass 'second verification retry' validate_remote v0.2.0 "$main_sha" clearance-123 404
after_refs=$(git --git-dir="$origin" show-ref)
if [ "$before_refs" != "$after_refs" ]; then
  echo 'verification retries changed remote refs' >&2
  exit 1
fi
checks=$((checks + 1))
printf 'ok %d - verification retries are read-only\n' "$checks"
printf 'verified %d release identity checks\n' "$checks"
