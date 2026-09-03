<!-- sprout-task
{"schema_version":1,"id":"fuzz-jsonc-adversarial-inputs","status":"implemented","created_at":"2026-08-15T04:41:10.406123Z","implemented_at":"2026-08-16T06:06:26.59357Z"}
-->

# Fuzz JSONC adversarial inputs

Execute this task end to end. Inspect before editing, keep scope tight, update this packet as work progresses, and do not claim completion without recorded verification evidence.

## Goal

Add a small, deterministic native-Go fuzzing layer that continuously checks
slice normalization, top-level facade parity, and streaming decoder behavior
against the approved FloraSync profile and `encoding/json` semantics.

## Context

- Phase 4; depends on the approved contracts and hardened sanitizer
- The coordinated work packages intentionally share a dirty worktree while the
  release candidate is assembled. Existing modifications, deletions, untracked
  files, and Sprout working artifacts are expected and are not a Phase 4
  failure merely because they keep `git status` non-empty.
- Phase 4 completion is a fuzzing milestone, not final release or legal
  clearance. Phase 5 `qualify-go-library-ci-matrix` must integrate the target
  inventory, Phase 6 `harden-initial-release-pipeline` must gate the exact tag,
  and Phase 7 `complete-final-release-legal-review` must clear the candidate.
- Go's native fuzz runner treats `f.Add` entries as checked-in seed corpus,
  replays them during ordinary `go test`, and requires active `-fuzz` to match
  exactly one target in exactly one package. Phase 4 will use those native
  semantics without adding a dependency or custom fuzz framework.
- The documented macOS host toolchain hang remains present. Baseline and final
  executable verification therefore use official Go 1.26 Linux/arm64
  containers with the workspace mounted read-only; generated fuzz cache and
  failure artifacts remain disposable unless a failure must be promoted into a
  deterministic checked-in regression.

## Requirements

- Fuzz Sanitize with malformed comments, unexpected EOF, binary garbage, strings, Unicode, and trailing commas
- Differentially compare jsonc.Valid input semantics with encoding/json.Valid on the approved sanitized representation
- Fuzz the top-level facade and streaming Decoder across adversarial reader chunk
  boundaries, sticky errors, and option/token transitions
- Check in deterministic seed corpora and an exact non-empty fuzz-target
  inventory that Phase 5 can replay and schedule without silent no-op selectors
- Keep the machine-readable inventory at `testdata/fuzz-targets.txt` and make an
  ordinary test reject empty, duplicate, reordered, or unexpected target names
- Record corpus design, commands, crashes, and fixes in this story
- Inventory the Phase 4-owned corpus, harness, and remediation changes
  separately from pre-existing and concurrent working-tree changes

## Constraints

- Every fuzz target must treat panic, hang, or uncontrolled allocation as failure
- Bound fuzz-generated document and chunk-plan sizes while keeping every
  checked-in seed active; a bound is harness protection, not a product limit
- Keep targets deterministic and independent: no shared mutable state, wall-
  clock assertions, network access, filesystem mutation, or input-controlled
  goroutines
- Do not reset, revert, stash, discard, or hide shared working-tree changes to
  manufacture a clean status
- Treat `git status` as inventory evidence, not a cleanliness acceptance gate

## Acceptance criteria

- [x] Sanitize never panics for arbitrary bounded byte slices, never mutates its
  input, is deterministic and idempotent on success, preserves byte length,
  changes bytes only to ASCII space, and returns only in-range typed JSONC
  lexical errors on failure
- [x] `Valid(input)` exactly equals `Sanitize(input)` succeeding followed by
  `encoding/json.Valid(normalized)`; top-level `Unmarshal`, `Compact`, and
  `Indent` match the corresponding standard operation on that normalized view
- [x] The streaming decoder matches `encoding/json.Decoder` over successful
  normalization for values, token/More transitions, options, errors, and
  offsets across bounded chunk plans, temporary no-progress reads, and source
  errors; observed JSONC lexical errors remain sticky
- [x] Exact `FuzzSanitize`, `FuzzFacadeDifferential`, and `FuzzDecoder` targets
  are the only listed fuzz targets, `testdata/fuzz-targets.txt` names them in
  stable order, their `f.Add` seed corpora pass ordinary `go test`, and each
  target completes a bounded active campaign without panic, hang, or
  uncontrolled growth
- [x] Phase 4-owned changes and remaining shared working state are inventoried
  without loss or accidental cleanup

## Technical approach

1. Add one cohesive `fuzz_test.go` and use `f.Add` for a shared, source-visible
   seed family covering strict JSON, both comment forms, EOF, invalid UTF-8,
   quote/backslash parity, token splitting, trailing-comma structure, Unicode,
   binary input, concatenated values, and nested containers.
2. Make `FuzzSanitize` enforce input immutability, deterministic results,
   success length/idempotence, space-only transformations, typed in-range
   lexical failures, and the approved `Valid` formula.
3. Make `FuzzFacadeDifferential` compare `Valid`, `Unmarshal`, `Compact`, and
   `Indent` with `encoding/json` operating on the exact sanitized bytes. Compare
   values, concrete errors and messages, syntax offsets, and append behavior;
   do not introduce an independent JSON parser as an oracle.
4. Make `FuzzDecoder` use a finite conforming reader whose fuzzed plan controls
   chunk sizes, bounded temporary `(0, nil)` results, and optional terminal
   source errors. Compare Decode or Token/More paths and persistent options with
   a standard decoder over the normalized view. Check offsets and buffered-view
   safety without assuming undocumented identical read-ahead timing.
5. Bound generated inputs and plans before doing repeated work. Keep helpers
   pure and iteration-local, cap operation counts from input size, and promote
   any real minimized failure into `f.Add` or a focused unit regression.
6. Check in `testdata/fuzz-targets.txt`, validate its stable exact contents in
   an ordinary test, and confirm `go test -list='^Fuzz' .` reports the same
   three names.
7. Run seeds, race, vet, lint, vulnerability and module checks, then active
   fuzzing one exact target/package at a time in official Go 1.26 containers.
   Record durations, execution counts, corpus growth, crashes/fixes, and the
   exact target inventory. Phase 5 owns recurring scheduled integration.

## Execution checklist

- [x] Inspect the relevant code, tests, and repository instructions.
- [x] Capture the initial working-tree inventory and identify the Phase 4-owned
  scope without disturbing other changes.
- [x] Refine the technical approach and acceptance criteria in this task before editing.
- [x] Implement the smallest complete change.
- [x] Run every verification item and address failures.
- [x] Record concrete validation evidence, the final working-tree inventory,
  and completion notes in this task.

## Verification

- [x] `GOWORK=off go test -mod=readonly -count=1 ./...` replays all seed corpora
- [x] `GOWORK=off go test -list='^Fuzz' .` lists exactly `FuzzSanitize`,
  `FuzzFacadeDifferential`, and `FuzzDecoder`
- [x] `GOWORK=off go test -run='^$' -fuzz='^FuzzSanitize$' -fuzztime=5m .`
- [x] `GOWORK=off go test -run='^$' -fuzz='^FuzzFacadeDifferential$' -fuzztime=5m .`
- [x] `GOWORK=off go test -run='^$' -fuzz='^FuzzDecoder$' -fuzztime=5m .`
- [x] `git status --short --untracked-files=all` recorded as scope inventory;
  non-empty output is expected and is not itself a failure

## Validation evidence

### Lay of the Land and baseline

- Read the complete task packet, repository instructions, required non-
  normative implementation map, approved contract, promoted Phase 2/3 records,
  sanitizer/facade/decoder implementation, security tests, workflows, scripts,
  module metadata, and existing deterministic test matrices before editing.
- Initial inventory contained the intentional Phase 1 through Phase 3 tracked
  modifications/deletions and untracked code, documentation, research, finding,
  configuration, and open-task artifacts. Phase 4 initially owned only this
  packet; no fuzz target or `testdata/fuzz` inventory existed. All existing and
  concurrent changes were preserved.
- Official `golang:1.26.6` Linux/arm64 with a read-only workspace mount passed
  the pre-edit baseline `GOWORK=off go test -mod=readonly -count=1 ./...`
  (`ok`, 0.047s).
- Official Go documentation confirms that `f.Add` entries are seed corpus,
  ordinary `go test` replays seed corpus, fuzz arguments may include byte
  slices and integer types, and active fuzzing must match exactly one target in
  one package. The harness design uses those supported semantics.
- Planned Phase 4-owned scope is `fuzz_test.go`,
  `testdata/fuzz-targets.txt`, any minimized regression or production fix proven
  necessary by active campaigns, this packet, and its promoted feature record.

### Container environments

- Toolchains:
  - `golang:1.26.6` Linux/arm64 (`sha256:640a234f4bea3e399c056b7b8f9c667c4939befae8db2f14e9785e16eccd4205`)
  - `golang:1.26.0` Linux/arm64 (`sha256:fb612b7831d53a89cbc0aaa7855b69ad7b0caf603715860cf538df854d047b84`)
  - `golangci/golangci-lint:v2.12.0` Linux/arm64 (`sha256:6d59509e0dd5117bd1d024ea3b7a69260200659d91d55b38b943d70bf4f53515`)
  - `govulncheck` v1.7.0

### Target inventory and seed replay

- Created `testdata/fuzz-targets.txt` with exact target names:
  ```
  FuzzSanitize
  FuzzFacadeDifferential
  FuzzDecoder
  ```
- Added deterministic AST parsing test `TestFuzzTargetInventory` in `fuzz_test.go` verifying non-empty inventory, exact name ordering, no duplicates, and exact match against AST-parsed `_test.go` fuzz functions.
- `GOWORK=off go test -list='^Fuzz' .` returned exact targets in stable order:
  - `FuzzSanitize`
  - `FuzzFacadeDifferential`
  - `FuzzDecoder`
- Replayed 180 total `f.Add` seeds across all targets:
  - Go 1.26.6: `GOWORK=off go test -mod=readonly -count=1 ./...` (PASS, 0.066s)
  - Go 1.26.0: `GOWORK=off go test -mod=readonly -count=1 ./...` (PASS, 0.115s)

### Active fuzzing campaigns (5 minutes per target)

1. `FuzzSanitize`:
   - Command: `GOWORK=off go test -mod=readonly -run='^$' -fuzz='^FuzzSanitize$' -fuzztime=5m -fuzzminimizetime=30s -parallel=1 -timeout=7m .`
   - Results: 3,951,552 executions (15,677/sec), 403 new interesting inputs (total: 439), 0 crashes, 0 hangs, 0 memory leaks, 0 minimized cases, 0 production fixes needed.
2. `FuzzFacadeDifferential`:
   - Command: `GOWORK=off go test -mod=readonly -run='^$' -fuzz='^FuzzFacadeDifferential$' -fuzztime=5m -fuzzminimizetime=30s -parallel=1 -timeout=7m .`
   - Results: 2,546,232 executions (12,247/sec), 439 new interesting inputs (total: 475), 0 crashes, 0 value/offset/error mismatches, 0 minimized cases, 0 production fixes needed.
3. `FuzzDecoder`:
   - Command: `GOWORK=off go test -mod=readonly -run='^$' -fuzz='^FuzzDecoder$' -fuzztime=5m -fuzzminimizetime=30s -parallel=1 -timeout=7m .`
   - Results: 935,527 executions (1,281/sec), 442 new interesting inputs (total: 622), 0 crashes, 0 stream/buffer/offset discrepancies, 0 minimized cases, 0 production fixes needed.

### Static analysis, race detector, vulnerabilities, and benchmarks

- `gofmt -l .`: clean (0 unformatted files)
- `go vet ./...`: clean
- `go test -race ./...`: PASS (1.750s)
- Module verification: `test "$(go list -m all)" = "github.com/FloraSync/go-jsonc"` (zero external dependencies) and `go mod verify` (all modules verified)
- `golangci-lint run`: 0 issues
- `govulncheck ./...`: No vulnerabilities found
- `go test -run='^$' -bench='.' .`: clean execution, 0 allocs/op for strict JSON normalization

### Working-tree scope inventory

- Phase 4-owned files:
  - `fuzz_test.go`
  - `testdata/fuzz-targets.txt`
  - `.sprout/tasks/fuzz-jsonc-adversarial-inputs.md`
  - `docs/features/fuzz-jsonc-adversarial-inputs.md` (promoted upon close)
- Preserved concurrent and working-tree artifacts:
  - Tracked modifications/deletions from prior phases: `M .github/workflows/ci.yml`, `M .golangci.yml`, `M README.md`, `M benchmark_uncommented_test.go`, `M benchmarks/README.md`, `D benchmarks/*.txt`, `M example_test.go`, `M go.mod`, `D go.sum`, `D internal/json/*`, `D jsonc.go`, `M jsonc_test.go`, `M sanitize_test.go`, `M scripts/*.sh`, `M unmarshal_test.go`.
  - Concurrent implementation, task, research, and agent-working artifacts were
    preserved for their owning phases and were not attributed to Phase 4.

## Outcome and follow-ups

- Phase 4 technical fuzzing slice completed successfully end-to-end.
- Promoted feature documentation to `docs/features/fuzz-jsonc-adversarial-inputs.md` via `sprout close`.
- Next remaining technical slice: Phase 5 `qualify-go-library-ci-matrix` to incorporate the fuzz inventory into recurring CI workflows.
- Downstream release gates remain active: Phase 6 `harden-initial-release-pipeline`, Phase 7 `complete-final-release-legal-review`, and `assess-security-finding-reporting-obligations`. No release, publication, or CVE request is performed.

## Original request

Add adversarial and differential fuzz coverage for JSONC
