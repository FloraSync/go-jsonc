<!-- sprout-task
{"schema_version":1,"id":"align-encoding-json-facade-and-sanitizer","status":"implemented","created_at":"2026-08-15T04:41:06.360404Z","implemented_at":"2026-08-15T06:40:16.660478Z"}
-->

# Align encoding/json facade and JSONC sanitizer

Execute this task end to end. Inspect before editing, keep scope tight, update this packet as work progresses, and do not claim completion without recorded verification evidence.

## Goal

Deliver the smallest complete Go 1.26+ implementation of the approved
FloraSync JSONC Profile v1 and stable `encoding/json` facade.

## Context

- Phase 1 was approved on 2026-08-14 and promoted to
  `docs/features/define-jsonc-stdlib-compatibility-contracts.md`.
- The owner subsequently fixed the minimum supported toolchain at Go 1.26.0.
  Go 1.25 and earlier are intentionally unsupported.
- The ambient `/Users/shoe/Code/FloraSync/go.work` does not include this module;
  all module-local Go commands must run with `GOWORK=off`.
- Sprout files are working-state artifacts and are not intended for the product
  commit.
- The coordinated hardening program intentionally accumulates changes in a
  shared dirty worktree until the final release candidate is assembled. A
  non-empty `git status` was therefore expected and was not a Phase 2 failure
  or a reason to discard concurrent work.
- Phase 2 closure records the verified implementation milestone; it does not
  imply a clean tree, legal clearance, or permission to publish. Final release
  remains gated by `complete-final-release-legal-review`.

## Requirements

- Change the module path to `github.com/FloraSync/go-jsonc`, the package name to
  `json`, and the module directive to `go 1.26.0`.
- Mirror the complete approved stable default `encoding/json` surface with
  aliases/direct delegation wherever JSONC behavior does not require wrapping.
- Implement the approved line-comment, block-comment, trailing-comma, Unicode,
  typed-error, and length-preserving normalization contracts.
- Use a lexical no-transform fast path that ignores comment-looking content in
  strings and adds no normalization allocation for ordinary JSON.
- Implement a JSONC-aware streaming `Decoder` without eager `io.ReadAll`,
  including arbitrary reader chunk boundaries and all standard methods.
- Ensure `RawMessage`, custom unmarshalers, error offsets, `Buffered`, and
  `InputOffset` observe the approved normalized/original-offset contracts.
- Interpret `Buffered` exactly at the decoder boundary: it exposes the standard
  decoder's unread bytes followed by normalized bytes already committed by the
  upstream normalizer. Future-dependent comma lookahead is not committed or
  exposed until a later significant byte determines whether that comma is a
  trailing comma. This keeps `Buffered` non-blocking and matches the standard
  rule that an upstream reader's internal lookahead is not decoder-buffer data.
- Retain the existing `Sanitize`, `HasCommentRunes`, and `ErrInvalidUTF8`
  extensions under their approved Phase 1 semantics.
- Remove all non-standard-library Go modules, alternative JSON backends, and
  their build tags.
- Update package documentation, examples, scripts, CI toolchain versions, and
  tests to the FloraSync identity and Go 1.26+ policy.

## Constraints

- Use only the Go standard library in source, tests, and benchmarks.
- Do not add compatibility code, files, build tags, or CI coverage for Go 1.25
  or earlier.
- Do not implement the Phase 3 `govulncheck` integration or Phase 4 fuzz
  campaign in this story.
- Do not commit Sprout working-state artifacts.
- Preserve Apache-2.0 attribution and repository history.

## Acceptance criteria

- [x] A caller using every stable default `encoding/json` symbol compiles after
  changing only the import string.
- [x] The declared module/package identity is
  `github.com/FloraSync/go-jsonc` / `json` with minimum Go 1.26.0.
- [x] All approved JSONC examples and malformed cases produce the specified
  results without changing input bytes.
- [x] Normalized output is byte-length preserving and cannot join tokens or
  turn invalid comma forms into valid JSON.
- [x] Ordinary JSON delegates with no normalization allocation and preserves
  standard outputs, errors, offsets, and decoding behavior.
- [x] Streaming decode handles one-byte readers and delimiters/comments split
  at every boundary while preserving all Decoder options and methods.
- [x] The module graph contains no third-party Go modules and no alternative
  backend source/build tags remain.
- [x] Documentation and examples describe the FloraSync profile, trailing
  comma extension, Go 1.26+ support, and migration caveats accurately.

## Technical approach

Use a small facade around `encoding/json`:

1. Alias stable standard types and directly delegate encoding-only operations.
2. Build a non-recursive byte lexer/normalizer that lazily copies only when it
   sees a real comment or eligible trailing comma outside strings. Comments
   become same-length whitespace; accepted trailing commas become one space.
3. Track the previous significant byte and look ahead only across JSONC
   whitespace/comments so empty/repeated comma forms remain invalid.
4. Validate UTF-8 only in comment bodies and return `JSONCSyntaxError` with
   one-based original byte offsets for JSONC lexical failures. If an
   unterminated block comment also contains malformed UTF-8, report the first
   malformed byte; otherwise report the opening slash as the unterminated
   comment offset. This deterministic precedence is shared by slice and stream
   paths.
5. Put the normalizer in front of `encoding/json.Decoder` as a stateful
   streaming reader, preserving byte count and exposing the normalized buffered
   view. Keep unresolved comma lookahead inside that upstream reader until it is
   committed; do not make `Buffered` block to resolve future input. Do not copy
   or fork standard-library decoder internals.
6. Prove the tracer bullet first with `Sanitize`, `Unmarshal`, `Valid`, and the
   direct Marshal/Encoder facade; then add the full streaming method surface.
7. Replace third-party assertions with table-driven standard-library tests and
   add compile-time/full-surface parity tests.

## Execution checklist

- [x] Inspect the relevant code, tests, workflows, module graph, approved
  contract, and repository instructions.
- [x] Refine the technical approach and acceptance criteria before editing.
- [x] Implement and validate the tracer-bullet normalization/facade path.
- [x] Implement the full streaming Decoder path.
- [x] Remove legacy module/backend/package identity baggage and update docs.
- [x] Run every verification item and address failures.
- [x] Record concrete validation evidence and completion notes.

## Verification

- [x] Official `golang:1.26.0` container: `go test -mod=readonly -count=1
  ./...`.
- [x] Official `golang:1.26.6` container: `go test -mod=readonly -count=1
  ./...`.
- [x] Official `golang:1.26.6` container: `go test -mod=readonly -race
  -count=1 ./...`.
- [x] Official `golang:1.26.6` container: `go vet ./...`.
- [x] Go 1.26.6 `go mod tidy -diff` is empty and `go list -m all` lists only
  the main module. CI and `scripts/test.sh` enforce that single-line graph.
- [x] `golangci-lint` v2.12.0 reports `0 issues` with the migrated v2 config.
- [x] Focused sanitizer benchmarks record strict fast-path and transformed
  JSONC allocations.

## Validation evidence

- Before edits, plain `go test ./...` failed because the enclosing `go.work`
  excludes this module.
- Baseline `env GOWORK=off go test ./...` passed under local Go 1.25.12 before
  the minimum-version migration.
- Host-native execution became unavailable during closing verification:
  freshly built one-line Go 1.25 and 1.26 executables both hung before `main`,
  and macOS `spctl` reported an internal Code Signing subsystem error. This was
  isolated from repository behavior by moving execution to official read-only
  Linux containers.
- Exact baseline result: official `golang:1.26.0` passed the uncached full suite
  after the final streaming-memory fix (`ok`, 0.138s).
- Current patch result: official `golang:1.26.6` passed the uncached full suite
  (`ok`, 0.210s), race suite (`ok`, 2.872s), and `go vet` with no output.
- The strict benchmark path measured `0 B/op, 0 allocs/op`; the commented and
  trailing-comma paths each measured one normalization allocation. These are
  Linux/arm64 directional results, not cross-machine performance promises.
- Test coverage is 92.5% of statements under Go 1.26.6.
- `gofmt -l .`, `git diff --check`, and `bash -n scripts/test.sh
  scripts/benchmark.sh` all produced no output.
- Module identity is `github.com/FloraSync/go-jsonc` with package identifier
  `json`; `go.sum` and the alternative backend directory are absent.

## Outcome and follow-ups

Implemented the Go 1.26-only standard-library facade, a shared deterministic
JSONC normalizer, and a stateful streaming decoder. Normalization preserves
byte length, rejects malformed comment Unicode, fails closed on token splitting
and invalid comma forms, and adds no allocation on strict slice input. The
stream path retains one persistent standard decoder, supports arbitrary chunk
boundaries, preserves offsets/callback bytes/options, and transfers large
trailing-comma lookahead storage instead of retaining duplicate attacker-sized
buffers.

The module/package identity, full stable Go 1.26 API surface, documentation,
examples, scripts, CI, lint configuration, tests, and benchmarks now use the
FloraSync/Go 1.26 policy. All third-party Go modules and alternative JSON
backends were removed, and CI enforces the zero-dependency graph.

Phase 3 remains responsible for `govulncheck` and the broader security/supply-
chain audit. Phase 4 remains responsible for native fuzz harnesses and sustained
adversarial fuzzing. Phase 5 qualifies the bounded hosted CI matrix and Phase 6
hardens the exact-tag release path. No Sprout artifact is intended for the
product commit.

The repository intentionally remains dirty while those coordinated work
packages converge. Each phase uses `git status` as change inventory rather than
a clean-tree gate and must preserve pre-existing or concurrent work. After the
final release candidate and release path are assembled,
`complete-final-release-legal-review` (Phase 7)
must classify the complete change set and record qualified, non-privileged
legal clearance for the exact candidate before any release.

## Original request

Refactor the package into an encoding/json-compatible JSONC facade
