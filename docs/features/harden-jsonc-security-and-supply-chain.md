<!-- sprout-task
{"schema_version":1,"id":"harden-jsonc-security-and-supply-chain","status":"implemented","created_at":"2026-08-15T04:41:08.399371Z","implemented_at":"2026-08-15T07:27:01.18599Z"}
-->

# Harden JSONC security and supply chain

Execute this task end to end. Inspect before editing, keep scope tight, update this packet as work progresses, and do not claim completion without recorded verification evidence.

## Goal

Harden JSONC security and supply chain

## Context

- Phase 3; depends on the approved contracts and Phase 2 sanitizer
- The coordinated work packages intentionally share a dirty worktree while the
  release candidate is assembled. Existing modifications, deletions, untracked
  files, and Sprout working artifacts are expected and are not a Phase 3
  failure merely because they keep `git status` non-empty.
- Phase 3 completion is a security milestone, not final release or legal
  clearance. Phase 4 fuzzing, Phase 5 CI qualification, Phase 6 release
  hardening, and Phase 7 `complete-final-release-legal-review` remain in the
  pre-release chain.

## Requirements

- Statically audit all untrusted-input state-machine paths
- Integrate govulncheck into repository automation
- Record security findings and remediation evidence in this story
- Inventory the Phase 3-owned changes separately from pre-existing and
  concurrent working-tree changes

## Constraints

- No regular-expression sanitizer
- No unbounded recursion or avoidable input amplification
- Do not reset, revert, stash, discard, or hide shared working-tree changes to
  manufacture a clean status
- Treat `git status` as inventory evidence, not a cleanliness acceptance gate

## Acceptance criteria

- [x] Malformed byte input cannot cause an out-of-bounds panic on either the
  slice or conforming-reader path, including at the exact 10,000-frame tracking
  boundary and immediately beyond it
- [x] Impossible `io.Reader` counts fail with a precise sticky error instead of
  panicking or looping, while a temporary `(0, nil)` read can resume normally
- [x] Terminal JSONC lexical failures release unresolved comma lookahead rather
  than retaining an attacker-sized uncommitted buffer
- [x] CI or an equivalent checked-in target runs govulncheck ./...
- [x] Phase 3-owned changes and remaining shared working state are inventoried
  without loss or accidental cleanup

## Technical approach

Keep the Phase 2 architecture intact: a non-recursive slice normalizer and a
stateful streaming normalizer feeding `encoding/json`. Harden only the audited
boundaries:

1. Add a tracer-bullet guard immediately after the source `Read` so counts
   outside `0 <= n <= len(p)` become a private sticky terminal error before any
   slice expression. This is defense in depth beyond the `io.Reader` contract
   and does not alter conforming-reader behavior.
2. Centralize JSONC lexical failure finalization so unresolved comma lookahead
   is discarded on terminal lexical errors. Preserve already committed output
   and preserve the existing data-before-source-error behavior.
3. Add deterministic security characterization tests for tiny/malformed input,
   impossible and temporarily empty reader results, pending slash/comment/UTF-8/
   comma transitions, sticky errors, large-lookahead release, and the exact
   nesting cutover. Do not add fuzz targets or a sustained fuzz campaign; those
   remain owned by Phase 4.
4. Install a pinned Go-native `govulncheck` release in the existing least-
   privilege CI lint job and run the literal text-mode `govulncheck ./...`
   command. Keep the tool outside `go.mod` so the shipping module graph remains
   standard-library-only and vulnerability findings fail CI.
5. Verify under official Go 1.26 Linux containers because the documented host
   code-signing problem still prevents reliable native execution. Run tests,
   race, vet, lint/config checks, the pinned vulnerability scan, formatting,
   module-graph checks, and Sprout completion validation.

## Execution checklist

- [x] Inspect the relevant code, tests, and repository instructions.
- [x] Capture the initial working-tree inventory and identify the Phase 3-owned
  scope without disturbing other changes.
- [x] Refine the technical approach and acceptance criteria in this task before editing.
- [x] Implement the smallest complete change.
- [x] Run every verification item and address failures.
- [x] Record concrete validation evidence, the final working-tree inventory,
  and completion notes in this task.

## Verification

- [x] Go 1.26.0: go test -mod=readonly -count=1 ./...
- [x] Go 1.26.6: go test -mod=readonly -race -count=1 ./...
- [x] go vet ./...
- [x] govulncheck ./...
- [x] golangci-lint v2.12.0 and actionlint
- [x] go mod tidy -diff and single-module go list -m all
- [x] gofmt -l ., git diff --check, and shell syntax checks
- [x] `git status --short --untracked-files=all` recorded as scope inventory;
  non-empty output is expected and is not itself a failure

## Validation evidence

### Static security findings and remediation

- All slice lookaheads, UTF-8 decoding, comment blanking, frame indexes, and
  pending-comma indexes were guarded. The stream frame/UTF-8 indexes were also
  guarded. Both paths are iterative O(n), structural state is capped at 10,000
  frames, and no regex, recursion, unsafe code, or shared mutable parser state
  is present.
- The tracer-bullet test reproduced one concrete defense-in-depth defect: a
  contract-violating reader returning 513 for a 512-byte destination caused
  `panic: runtime error: slice bounds out of range [:513]`. A reader repeatedly
  returning a negative count with a nil error could likewise keep the
  normalizer's internal read loop retrying without progress. Counts are now
  validated before slicing and impossible values become a precise sticky error.
  The focused regression passed after the fix. The separately tracked finding
  is `FS-SEC-2026-001`.
- Terminal malformed-comment errors could retain an arbitrarily large pending
  comma/trivia allocation. JSONC lexical failure now uses one cleanup path that
  preserves committed output but releases speculative comma state. A 1 MiB
  hostile-comment regression failed before this change and passed afterward.
- Deterministic tests cover nil/tiny input, lone delimiters, repeated stars,
  truncated comment UTF-8, binary bytes, depths 9,999/10,000/10,001, temporary
  `(0, nil)` reads while slash/UTF-8/comma state is pending, source-error versus
  lexical-error precedence, and sticky errors across `Token`, `Decode`,
  `Buffered`, and `InputOffset`.
- A temporary `(0, nil)` source result is returned without an internal
  normalizer spin and later data resumes normally. A caller-owned reader that
  never returns data or an error supplies no progress indefinitely; this stays
  at the standard `io.Reader` boundary rather than receiving an arbitrary
  document timeout.
- Unresolved comma lookahead remains necessarily O(n) when arbitrary trivia
  precedes the deciding byte. It uses one buffer rather than amplified
  duplicates. Callers needing an absolute stream-size policy can compose an
  `io.LimitReader` outside the decoder.

### Automation and command evidence

- Baseline before Phase 3 changes: official `golang:1.26.6` container
  `go test -mod=readonly -count=1 ./...` passed (`ok`, 0.021s), and `go vet`
  produced no output.
- Final official `golang:1.26.0` Linux/arm64 container:
  `go test -mod=readonly -count=1 ./...` passed (`ok`, 0.039s).
- Final official `golang:1.26.6` Linux/arm64 container:
  `go test -mod=readonly -race -count=1 ./...` passed (`ok`, 1.639s), and
  `go vet ./...` produced no output.
- `golangci-lint` v2.12.0 reported `0 issues`; actionlint v1.7.12 accepted the
  workflow with no output.
- CI installs `govulncheck@v1.7.0` outside the product module and runs literal
  text-mode `govulncheck ./...` under the supported minimum Go 1.26.0 toolchain.
  The exact install-and-scan path exited 0 with zero reachable vulnerabilities.
  The scanner also reported 3 package-level and 30 module-level known-
  vulnerability findings with no vulnerable symbols called. Under patched Go
  1.26.6 the same literal scan reported `No vulnerabilities found`.
- `go mod tidy -diff`, `gofmt -l .`, `git diff --check`, and shell syntax checks
  produced no output. `go list -m all` printed only
  `github.com/FloraSync/go-jsonc`. `sprout check` passed throughout the task.
- Native host execution remains unreliable because of the pre-existing macOS
  code-signing/toolchain hang documented by Phase 2. Official read-only Linux
  containers were used consistently; this is an environment limitation, not a
  repository test failure.

### Scope inventory

Phase 3 owns only:

- `stream_normalizer.go`: reader-count validation and centralized terminal
  lexical cleanup;
- `security_test.go`: deterministic security characterization and regression
  coverage;
- `.github/workflows/ci.yml`: the six-line pinned govulncheck install/scan hunk,
  layered on the pre-existing Phase 2 workflow rewrite; and
- this task packet and its Sprout-promoted feature record.

The initial inventory already contained the accumulated Phase 1/2 changes:
tracked workflow/lint/README/module/script/test/benchmark edits, backend and
legacy implementation deletions, and untracked facade, sanitizer, decoder,
documentation, repository-instruction, research, configuration, and task
artifacts. All were preserved. The final inventory remains intentionally
non-empty and additionally contains two concurrently created, untouched task
packets: `harden-initial-release-pipeline` and
`qualify-go-library-ci-matrix`. Neither is attributed to Phase 3.

## Outcome and follow-ups

Status: **implemented and technically verified on 2026-08-15**.

Before Phase 3, the parser architecture was already cohesive and non-recursive,
but its streaming adapter trusted impossible reader counts and terminal lexical
errors could retain speculative attacker-sized lookahead. CI had static
linters and race tests but no Go vulnerability-database gate. After Phase 3,
the same architecture has an explicit defensive reader boundary, one terminal
lexical cleanup transition, deterministic security characterization at the
depth/state/error boundaries, and a pinned failing vulnerability scan in CI.
No public API or conforming-reader behavior changed.

Phase 4 remains responsible for native fuzz harnesses, corpus design,
differential validity oracles, and sustained adversarial campaigns. The later
CI qualification and release-hardening packets remain responsible for their
broader pipeline scopes. The scanner pin requires deliberate periodic updates,
vulnerability results evolve with the database, and existing mutable major
action tags/release-tool versions are not hidden Phase 3 work.

The worktree remains intentionally dirty for coordinated package assembly.
This security milestone is not release or legal clearance. Phases 4 through 7
remain release blockers, and no release, tag, module publication, or artifact
distribution is authorized until the exact candidate passes
`complete-final-release-legal-review` with qualified external clearance.

## Original request

Audit and harden JSONC parsing and repository vulnerability checks
