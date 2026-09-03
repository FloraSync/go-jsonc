<!-- sprout-task
{"schema_version":1,"id":"harden-initial-release-pipeline","status":"implemented","created_at":"2026-08-15T07:10:38.13924Z","implemented_at":"2026-09-02T23:09:37.508536Z"}
-->

# Harden the initial release pipeline

Execute this task end to end. Inspect before editing, keep scope tight, update this packet as work progresses, and do not claim completion without recorded verification evidence.

## Goal

Make the first FloraSync release path verify the exact tagged candidate, require
explicit protected authorization, and publish reproducibly with the smallest
possible token authority only after all technical and legal gates are satisfied.

## Context

- Phase 6; begins only after `qualify-go-library-ci-matrix` closes and provides
  stable required-check names plus a reusable exact-candidate verification path.
- Existing upstream tags `v0.1.0` and `v0.1.1` must remain immutable even though
  GitHub currently lists no published releases. Initial FloraSync versioning
  must select a new non-colliding SemVer tag; this task does not choose or push
  that tag without explicit owner approval.
- The current library-only GoReleaser configuration skips binary builds and the
  release workflow uses floating/major-only actions, `version: latest`, and no
  pre-release technical gate. The 2026-08-15 audit also found no repository
  rulesets or `main` branch protection.
- This task configures and dry-runs the release machinery but publishes nothing.
  `complete-final-release-legal-review` is bumped to Phase 7 and remains the
  final candidate-clearance gate. Actual publication still requires a separate
  explicit owner action after that clearance.
- The shared dirty-worktree policy remains active: inventory all material state;
  do not reset, revert, stash, discard, or hide concurrent changes.

## Requirements

- Make the tag-triggered or manually dispatched release workflow verify the
  exact selected tag revision through the Phase 5 reusable gates before any job
  receives write permission or can create a GitHub release.
- Refuse a missing, mutable, malformed, already-used, or non-ancestor-of-`main`
  tag. Preserve `v0.1.0` and `v0.1.1`; do not overwrite historical tags.
- Pin every release action to a verified full-length commit SHA and pin an exact
  GoReleaser version compatible with a checked-in, current-schema configuration.
  Eliminate `version: latest` and old action runtimes.
- Default workflow permissions to `contents: read`. Grant `contents: write` only
  to the final release job after verification and protected environment
  approval; grant `id-token: write` only if an implemented attestation step
  requires it.
- Configure a protected release environment with an owner-approved reviewer and
  no secrets exposed to untrusted pull-request code. Keep legal advice and
  privileged review material outside Actions logs and repository files.
- Configure repository rules so `main` requires the stable Phase 5 checks and
  pull-request review, force-push/deletion is denied, and release tags cannot be
  rewritten. Record the exact ruleset/protection API result as evidence.
- Validate the source-only library release contract: module path, Go baseline,
  LICENSE/README/documentation inclusion, generated release notes, and archive
  contents must match the exact tagged revision. Do not invent binary artifacts
  for a library-only module.
- Add deterministic pre-publication checks for the GoReleaser configuration and
  a no-publish snapshot/dry run. Validate checksums or provenance only for
  artifacts the chosen release design actually emits.
- Ensure release failure is fail-closed and idempotent: rerunning verification
  cannot create duplicate releases, move a tag, or partially publish under a
  different revision.

## Constraints

- Keep the change focused; avoid speculative abstractions and unrelated cleanup.
- Do not create or push a tag, publish a GitHub release, upload a public module,
  or mutate an existing release in this task.
- Do not weaken Phase 5 checks in the release caller or trust a prior run against
  a different SHA/ref.
- Do not store long-lived credentials in repository files or grant write
  permissions to lint/test jobs.
- Do not claim protected-branch/tag/environment completion from YAML alone;
  verify the live GitHub settings with read-back evidence.
- Repository-setting changes require explicit owner authorization and must be
  narrowly scoped and reversible without rewriting repository history.

## Acceptance criteria

- [x] Release verification runs against the exact tag SHA and must pass before
  the write-authorized job can start.
- [x] Every release action uses a verified full-length SHA, GoReleaser uses an
  exact version, actionlint is clean, and the release configuration passes its
  pinned checker.
- [x] Workflow/job permissions are read-only by default; only the final protected
  release job receives the minimal write scope.
- [x] A protected release environment requires an approved human reviewer, and
  no untrusted event can reach release credentials or write permissions.
- [x] Live repository read-back proves required checks/review on `main`, denied
  force-push/deletion, and immutable protected release tags.
- [x] Existing `v0.1.0` and `v0.1.1` remain unchanged, and a candidate initial
  FloraSync tag is validated as a new SemVer tag on the tested `main` history.
- [x] Pinned GoReleaser check and no-publish snapshot/dry run pass for the exact
  source-only library contents, with no unexpected files or binary artifacts.
- [x] Failure/retry tests prove no duplicate release, moved tag, partial publish,
  or cross-SHA verification path is possible.
- [x] No repository or remote release or tag is created during Phase 6, and Phase 7 legal clearance
  plus explicit owner authorization remain documented hard blockers.

## Technical approach

1. Inspect the Phase 5 workflow interface, current release YAML, GoReleaser
   schema, existing tags, live repository rules, and source-archive contents.
   Record immutable baseline identifiers before editing.
2. Extract or call the Phase 5 verification workflow from the release workflow
   so the release candidate is checked at the tag's exact SHA. Keep the release
   job downstream and unable to acquire write permission before success.
3. Add fail-closed SemVer, tag existence/ancestry, candidate identity, and
   already-published checks. Preserve historical tags and reject tag movement.
4. Pin actions by full SHA and GoReleaser by exact version; migrate the config to
   the pinned schema, then prove `check` and snapshot/no-publish behavior in a
   disposable local/container environment.
5. Minimize permissions and bind the write job to a protected release
   environment. Configure the smallest live branch/tag ruleset after explicit
   owner authorization, then read it back through GitHub's API.
6. Exercise safe failure paths and a complete dry run without creating a tag or
   release. Record exact candidate SHA, versions, action SHAs, ruleset IDs,
   workflow evidence, archive inventory, and remaining legal gate.

## Execution checklist

- [x] Inspect the relevant code, tests, and repository instructions.
- [x] Confirm Phase 5 is closed and capture its reusable workflow contract,
  stable required-check names, hosted run URL, and exact candidate revision.
- [x] Inventory existing tags/releases, current live protections/environments,
  release configuration, source archive inputs, and shared working state.
- [x] Refine the technical approach and acceptance criteria in this task before editing.
- [x] Implement exact-tag verification and a downstream least-privilege release job.
- [x] Pin/migrate release tooling and validate the source-only library release path.
- [x] Obtain explicit owner authorization, configure minimal live protections,
  and verify them by read-back without publishing.
- [x] Exercise no-publish success, failure, retry, and cross-SHA rejection paths.
- [x] Run every verification item and address failures.
- [x] Record exact dry-run, repository-settings, final inventory, and blocker evidence.

## Verification

- [x] `sprout check harden-initial-release-pipeline`
- [x] `actionlint`
- [x] Pinned GoReleaser configuration check
- [x] Pinned GoReleaser snapshot/no-publish run against a disposable exact candidate
- [x] Release workflow exact-tag, ancestry, already-used-tag, and cross-SHA tests
- [x] Source archive/module/LICENSE/README/documentation inventory
- [x] GitHub API read-back of required `main` checks/reviews, force-push/deletion
  denial, protected release tags, and protected release environment
- [x] Workflow permission review proves only the final approved job has write access
- [x] Existing `v0.1.0` and `v0.1.1` object IDs unchanged
- [x] `git diff --check`
- [x] `git status --short --untracked-files=all` recorded as scope inventory;
  non-empty output is expected and is not itself a failure
- [x] Confirm no shared-repository tag, release, or public artifact was created

## Validation evidence

### Earlier local architecture and tracer-bullet evidence on 2026-08-22

- Phase 5 remains open pending an exact hosted candidate run, so this Phase 6
  pass deliberately did not claim closure or mutate any live GitHub setting.
  It implemented the repository-side release path that can be reviewed in
  parallel while Phase 5's agent completes hosted qualification.
- Replaced automatic tag-triggered publication with an explicitly dispatched
  workflow requiring an existing tag, full candidate SHA, and approved
  non-privileged legal-clearance reference. The identify and reusable verify
  jobs are read-only; only the downstream `release` environment job receives
  `contents: write` after both gates pass.
- Added `.github/workflows/release-verify.yml`, a reusable exact-SHA gate that
  invokes the Phase 5 deterministic scripts for preflight, ordinary tests,
  race x10, atomic coverage, six-target cross-compilation, exact fuzz inventory,
  and benchmark smoke before checking and snapshotting the release config.
- Added `scripts/release-validate.sh`. It rejects malformed/noncanonical tags,
  build metadata, historical `v0.1.0`/`v0.1.1`, abbreviated SHAs, checkout/tag
  mismatches, tags not on remote `main`, existing releases, and API results
  other than the expected 404. The publish job repeats validation immediately
  before obtaining release authority, closing the verification/publication
  time-of-check gap without moving or force-fetching tags.
- Migrated `.goreleaser.yaml` to schema v2 and pinned GoReleaser v2.12.7.
  `scripts/release-snapshot.sh` requires that exact version, runs a no-publish
  clean snapshot, rejects binary output, and validates the module path, Go
  1.26.0 baseline, LICENSE, README, and metadata inventory.
- `shellcheck scripts/release-validate.sh scripts/release-snapshot.sh`,
  `git diff --check`, and `GOWORK=off go test -mod=readonly -count=1 ./...`
  passed locally. Every mutation also passed mandatory `go test ./...`.
- No tag, release, artifact publication, live repository setting, or external
  authorization was created or changed. Local actionlint and pinned
  GoReleaser execution remain pending because those binaries are unavailable
  in the offline host; they are mandatory in the reusable hosted gate.

### Safe Passage continuation on 2026-08-23

- Phase 5 is closed. Its hosted `CI` run
  `https://github.com/FloraSync/go-jsonc/actions/runs/32616686680` succeeded at
  exact revision `8bc45c1840dfaa88419ba0ffbb73ca22f3af3ae6`. The stable required
  check names are `preflight`, `runtime (linux-go-1.26.0)`,
  `runtime (linux-go-1.26.x)`, `runtime (macos-go-1.26.x)`,
  `runtime (windows-go-1.26.x)`, `race-coverage`, `cross-compile`, and
  `coverage-report`.
- Official tag-ref read-back verified the full action pins already used by the
  workflows: checkout `de0fac2e4500dabe0009e67214ff5f5447ce83dd`, setup-go
  `4b73464bb391d4059bd26b0524d20df3927bd417`, upload-artifact
  `ea165f8d65b6e75b540449e92b4886f43607fa02`, download-artifact
  `d3f86a106a0bac45b974a628896c90dbdf5c8093`, and Codecov
  `671740ac38dd9b0130fbe1cec585b89eea48d3de`.
- Upgraded GoReleaser from v2.12.7 to current stable v2.17.1, whose verified
  upstream tag peels to `83f4c19a5c5c0b9efef6bf2aedc6805bbcb9dfe2` and includes the
  v2.17 security dependency updates. The v2.17.1 checker exposed stale
  `changelog.skip` syntax; removing it and enabling the documented source
  archive made the checked-in schema pass.
- Hardened `scripts/release-snapshot.sh` to require the exact tool version and
  local tag/SHA identity, verify the source artifact type and checksum, unpack
  exactly one archive root, compare its complete file inventory with the tagged
  Git tree, reject binary output, and validate module path, Go 1.26.0 baseline,
  LICENSE, README, and checked-in documentation from the archive itself.
- Added `scripts/release-validate-test.sh`. Its 18 deterministic checks cover a
  valid stable and prerelease tag; malformed, build-metadata, numeric-leading-
  zero, historical, missing, moved, and non-main tags; abbreviated and cross-SHA
  identities; blank legal evidence; used and indeterminate GitHub release
  responses; and two read-only retries with unchanged remote refs. The tracer
  bullet found and fixed missing-ref ambiguity by using
  `git rev-parse --verify`.
- The final no-publish run materialized `HEAD`
  `8bc45c1840dfaa88419ba0ffbb73ca22f3af3ae6` plus only the Phase 6-owned
  overlay into a disposable repository with deterministic commit
  `9b603086e24ee1887582ace2c2aec6506dc3deaa` and disposable tag `v0.2.0`.
  GoReleaser produced only `artifacts.json`, `checksums.txt`, `config.yaml`,
  `metadata.json`, and `go-jsonc-0.2.0-SNAPSHOT-9b60308.tar.gz`; the SHA-256
  check and exact tagged-tree comparison passed. The disposable repository and
  tag were deleted after validation and never touched shared or remote refs.
- Exact pinned Safe Passage passed: actionlint v1.7.12, golangci-lint v2.12.0
  (`0 issues`), govulncheck v1.7.0 (`No vulnerabilities found`, database updated
  2026-08-21), GoReleaser v2.17.1 schema check, shellcheck and shell syntax,
  ordinary tests, race detector with ten repetitions, atomic coverage
  `473/491 = 96.334%`, six no-CGO cross-compiles, three-target fuzz inventory,
  benchmark smoke, `go vet`, module verification, and `git diff --check`.
- Point-in-time public GitHub read-back at `2026-08-23T06:11:47Z` found
  `main` at the exact Phase 5 SHA with `protected: false`, no repository
  rulesets, no environments, no releases, and only historical tags `v0.1.0`
  (`0936c967f8aa2920b03aeee2edfa6af433ee479b`) and `v0.1.1`
  (`0f0d9d7e45597f3f8977a2acd4e7e8cd6c329edb`). Both configured `gh`
  accounts have expired credentials, so authenticated setting changes and
  protected-setting read-back could not proceed.
- Phase 6 owns only `.goreleaser.yaml`, `.github/workflows/release.yml`,
  `.github/workflows/release-verify.yml`, `scripts/ci-preflight.sh`,
  `scripts/release-validate.sh`, new `scripts/release-validate-test.sh`,
  `scripts/release-snapshot.sh`, and this packet. The pre-existing Phase 5 task
  deletion, promoted record, security/legal packets, and internal task,
  research, plan, skill, and roadmap artifacts remain untouched and
  unattributed to Phase 6.

### Final live-protection and closing verification on 2026-09-02

- The owner explicitly authorized the live safeguards. Environment `release`
  has ID `21125415814`; unauthenticated REST read-back at
  `2026-09-02T23:08:44Z` returned `can_admins_bypass: false`, required-reviewer
  rule ID `64465097`, `prevent_self_review: true`, two required reviewers, and
  deployment policy
  `protected_branches: true`, `custom_branch_policies: false`. Browser
  read-back also proved the environment has no secrets or variables.
- Active branch ruleset `Protect main` has ID `22151153`, targets
  `~DEFAULT_BRANCH`, has an empty bypass list, denies deletion and non-fast-
  forward updates, requires one approval, dismisses stale approvals, requires
  approval of the latest reviewable push, and requires an up-to-date branch
  with all eight GitHub Actions checks: `preflight`, both Linux runtime checks,
  the macOS and Windows runtime checks, `race-coverage`, `cross-compile`, and
  `coverage-report`. Public REST read-back returned all eight with GitHub
  Actions integration ID `15368` and strict status-check enforcement.
- GitHub's environment UI reported that rulesets alone did not satisfy its
  `Protected branches only` selector. A matching classic `main` protection rule
  (ID `82655557`) was therefore added as the narrow compatibility bridge. UI
  read-back proved the same eight GitHub Actions checks, one approval, stale-
  approval dismissal, latest-push approval, up-to-date-branch enforcement, and
  disabled force-push/deletion. Reloading the environment then reported that
  the policy applies to exactly one protected branch: `main`.
- Active tag ruleset `Protect release tags` has ID `22151294`, matches
  `refs/tags/v*`, has an empty bypass list, permits new tag creation, and denies
  update, deletion, and non-fast-forward changes. Public REST read-back
  returned rule types `update`, `deletion`, and `non_fast_forward`.
- Public REST read-back kept `main` at the Phase 5 revision
  `8bc45c1840dfaa88419ba0ffbb73ca22f3af3ae6` with `protected: true`, found no
  GitHub releases, and found only the unchanged historical tags `v0.1.0`
  (`0936c967f8aa2920b03aeee2edfa6af433ee479b`) and `v0.1.1`
  (`0f0d9d7e45597f3f8977a2acd4e7e8cd6c329edb`). No shared or remote tag,
  release, package, or other public artifact was created.
- A fresh isolated snapshot exposed a release-candidate integrity defect:
  GoReleaser omitted `.editorconfig`, `.gitattributes`, and `.gitignore` while
  the verifier correctly required an exact tagged-tree inventory. The source
  configuration now includes those files, and the snapshot wrapper removes
  GoReleaser's implementation-only backup file before enforcing the exact five
  expected outputs. Disposable candidate
  `ba06c194ef52667ad0156e631df115f50791b20f` at local-only tag `v0.2.0`
  then passed checksum and exact tagged-tree comparison with only
  `artifacts.json`, `checksums.txt`, `config.yaml`, `metadata.json`, and the
  source archive. The disposable repository and tag were removed.
- Closing Safe Passage passed `sprout check`, shell syntax and ShellCheck,
  all 18 release-validation cases, ordinary tests, the race detector with ten
  repetitions, `go vet`, atomic coverage `473/491 = 96.334%`, six no-CGO
  cross-compiles, the exact three-target fuzz inventory, benchmark smoke,
  module verification, actionlint v1.7.12, golangci-lint v2.12.0 with zero
  issues, govulncheck v1.7.0 with no vulnerabilities, GoReleaser v2.17.1
  configuration check, no-publish snapshot, and `git diff --check`.
- The task-owned change inventory remains `.goreleaser.yaml`,
  `.github/workflows/release.yml`, `.github/workflows/release-verify.yml`,
  `scripts/ci-preflight.sh`, `scripts/release-validate.sh`, new
  `scripts/release-validate-test.sh`, `scripts/release-snapshot.sh`, and this
  packet. All other shared dirty-worktree entries remain preserved and outside
  Phase 6 ownership.

## Outcome and follow-ups

Status: **Phase 6 release pipeline and live repository safeguards are complete
and verified; no release was published**.

The technical path is ready for the final candidate gate. A real initial tag
remains unselected; `v0.2.0` was used only inside the deleted disposable
verification repository. Phase 7 must record the designated qualified legal
reviewer's clearance of the exact release candidate, and the owner must then
separately authorize publication. Agents cannot self-approve that legal gate.

## Original request

Harden the FloraSync initial-release pipeline before final legal review.
