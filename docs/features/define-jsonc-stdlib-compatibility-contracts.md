<!-- sprout-task
{"schema_version":1,"id":"define-jsonc-stdlib-compatibility-contracts","status":"implemented","created_at":"2026-08-15T04:40:48.326073Z","implemented_at":"2026-08-15T05:20:49.724436Z"}
-->

# Define JSONC and encoding/json compatibility contracts

This is the Phase 1 approval artifact for the FloraSync/go-jsonc hardening
program. It defines contracts only. No implementation, test, CI, module, or
benchmark change is authorized by this story.

## Goal

Freeze a reviewable, testable contract for a secure JSONC parser facade that:

- accepts the named FloraSync JSONC Profile v1;
- is source-compatible with Go's stable `encoding/json` package;
- delegates ordinary JSON semantics to `encoding/json` without preprocessing;
- uses only the Go standard library at build and runtime; and
- blocks all implementation work until the repository owner approves the
  decisions in this packet.

## Context

### Program working-state and release policy

The dependent implementation, security, and fuzz work packages intentionally
converge in a shared dirty worktree. A non-empty `git status` during that work
is expected and is used as inventory, not as a failure condition or an excuse
to reset, revert, stash, discard, or hide concurrent changes. This program note
does not alter the approved JSONC or `encoding/json` contracts below.

Closing an individual work package proves only its scoped technical outcome.
The assembled release candidate requires a separate full review by the
designated qualified legal reviewer under
`complete-final-release-legal-review`; no feature closure alone authorizes a
tag, module publication, artifact distribution, or public release.

### Problem framing and assumptions

- Artifact: a public Go library processing untrusted byte slices and streams.
- Intended module path: `github.com/FloraSync/go-jsonc`.
- Intended default package identifier: `json`, so replacing only the import
  string `"encoding/json"` with `"github.com/FloraSync/go-jsonc"` leaves
  existing `json.X` call sites compiling.
- Compatibility target: the stable, default, non-experimental
  `encoding/json` v1 surface and documented behavior on supported Go releases.
- Security boundary: arbitrary `[]byte` and arbitrary `io.Reader` chunking,
  including malformed, hostile, and binary input.
- "Never panic" means no panic caused by library parsing or normalization for
  any input. It does not suppress panics deliberately raised by caller-owned
  `io.Reader`, `io.Writer`, `Marshaler`, or `Unmarshaler` implementations, and
  it cannot promise recovery from process-level out-of-memory failure.

### Authoritative baselines

#### JSONC

JSONC.org is a mutable draft without a released version. FloraSync Profile v1
is pinned to upstream commit `84b09994253d4da0fa8e619cfbe9aa6f6229790d`,
reviewed on 2026-08-14:

- [Pinned JSONC draft](https://github.com/JSONC-org/JSONC/blob/84b09994253d4da0fa8e619cfbe9aa6f6229790d/index.md)
- [Pinned JSONC ABNF](https://github.com/JSONC-org/JSONC/blob/84b09994253d4da0fa8e619cfbe9aa6f6229790d/grammar/JSONC.abnf)
- [Rendered draft](https://jsonc.org/)
- [RFC 8259](https://www.rfc-editor.org/rfc/rfc8259)

Upstream changes SHALL NOT silently change library behavior. A later JSONC.org
revision requires a new Sprout story, a grammar/behavior diff, compatibility
tests, and explicit approval before adoption.

#### Go standard library

The v1 compatibility reference is default `encoding/json` on the minimum Go
1.26 line. Go 1.27 made `encoding/json/v2` and `encoding/json/jsontext` stable
and reimplemented the v1 package on the new engine while preserving v1
behavior. The go-jsonc v1 facade therefore remains bound to the v1 API and
semantics. Go 1.27 callers may explicitly compose `Sanitize` with
`encoding/json/v2` when they want v2 semantics; that interoperation is covered
by version-gated tests.

- [Go 1.26.6 stream API](https://go.googlesource.com/go/+/refs/tags/go1.26.6/src/encoding/json/stream.go)
- [Go 1.26.6 decode API](https://go.googlesource.com/go/+/refs/tags/go1.26.6/src/encoding/json/decode.go)
- [Go 1.26.6 encode API](https://go.googlesource.com/go/+/refs/tags/go1.26.6/src/encoding/json/encode.go)
- [Go 1.26.6 scanner and Valid](https://go.googlesource.com/go/+/refs/tags/go1.26.6/src/encoding/json/scanner.go)
- [Go 1.26.6 formatting API](https://go.googlesource.com/go/+/refs/tags/go1.26.6/src/encoding/json/indent.go)
- [Go 1.27 release notes](https://go.dev/doc/go1.27#json_v2)
- [encoding/json/v2 migration guide](https://go.dev/doc/jsonv2-migration)

Supported release policy: Go 1.26.0 is the minimum baseline and the module
SHALL declare `go 1.26.0`. Go 1.25 and earlier are deliberately unsupported;
the implementation must not carry compatibility code, build tags, CI jobs, or
dependency constraints for those legacy releases. Verification SHALL cover
the exact Go 1.26.0 baseline, the current patched Go 1.26 toolchain, and current
Go 1.27 across the supported CI operating systems. Future stable Go releases
are supported subject to the stable API compatibility gate.

### FloraSync JSONC Profile v1

The words MUST, MUST NOT, SHALL, SHALL NOT, SHOULD, SHOULD NOT, and MAY are
normative.

#### Base document and whitespace

1. A document contains exactly one RFC 8259 JSON value, with optional JSONC
   whitespace/comments before and after it. Top-level scalars are allowed.
2. Strict RFC 8259 JSON is a subset of the profile.
3. Empty, whitespace-only, and comment-only documents are invalid.
4. Insignificant whitespace is exactly space U+0020, tab U+0009, LF U+000A,
   and CR U+000D. Form feed, vertical tab, non-breaking space, and other
   Unicode whitespace are not accepted as whitespace.
5. A leading UTF-8 BOM is not added to the grammar. It is passed unchanged to
   `encoding/json`, which determines validity.

#### Comment placement

1. Comments are allowed only where ordinary JSON permits insignificant
   whitespace.
2. A comment MUST NOT split a token. Inputs such as `tr/*x*/ue`,
   `1/*x*/.5`, `nu//x\nll`, and a comment inside `\\uXXXX` are invalid.
3. Replacing a comment with whitespace MUST preserve the represented JSON
   value. A normalizer MUST NOT join tokens by deleting comment bytes.
4. Comment markers inside a quoted JSON string are string data and MUST NOT be
   interpreted as comments. Quote recognition MUST honor odd/even backslash
   parity.

#### Single-line comments

1. A single-line comment begins with the exact ASCII bytes `//` outside a
   string.
2. It ends immediately before LF, CRLF, lone CR, or at EOF. The line ending is
   not part of the comment and remains input whitespace.
3. Backslash has no escape or continuation meaning in a comment.
4. Every valid Unicode scalar, including NUL and other control characters, is
   permitted in its body except CR and LF.
5. Under the pinned draft, U+2028 and U+2029 are ordinary comment characters,
   not line endings. Their treatment is an upstream security watch item; see
   JSONC-org issue 23 and pull request 32.
6. `#`, shebang, HTML, and any other comment syntax are unsupported.

#### Block comments

1. A block comment begins with exact ASCII bytes `/*` outside a string and
   MUST end at the first subsequent exact ASCII bytes `*/`.
2. Empty and multi-line block comments are valid. Backslash does not escape a
   closing delimiter.
3. Block comments do not nest. An inner-looking `/*` is ordinary body text;
   the next `*/` closes the only active comment, and remaining bytes are parsed
   normally.
4. Every valid Unicode scalar, including line endings and control characters,
   is permitted in the body except for the terminating byte sequence.
5. An unterminated block comment and a stray `*/` outside a comment are
   invalid. In particular, an unterminated comment after an otherwise complete
   JSON value MUST NOT be silently accepted.

#### Strings and ordinary JSON tokens

All other syntax is RFC 8259 JSON. Strings require double quotes; only RFC 8259
escapes are accepted; unescaped U+0000 through U+001F are invalid; and literals
`true`, `false`, and `null` are lowercase. Single quotes, unquoted keys,
hexadecimal numbers, leading `+`, `NaN`, `Infinity`, elisions, and other JSON5
features are invalid. Literal `/`, escaped `\/`, comment-looking text, and
commas inside strings remain string data.

#### FloraSync trailing-comma extension

The pinned JSONC.org ABNF and reference parser reject trailing commas by
default, while the prose permits parsers to support them. FloraSync deliberately
enables that permitted extension. It SHALL be described as the "FloraSync
JSONC Profile v1", not as JSONC.org's default profile.

The extension is equivalent to:

```abnf
array  = begin-array [ value *(value-separator value) [value-separator] ] end-array
object = begin-object [ member *(value-separator member) [value-separator] ] end-object
```

Exactly one trailing comma is accepted after at least one complete array
element or object member. JSONC whitespace/comments may occur between that
comma and the matching closing delimiter. A trailing comma is not accepted at
the root, in an empty container, after another comma, or before the wrong
closing delimiter. Consequently `[1,]`, `{"a":1,}`, and
`[1, /* note */]` are valid, while `[,]`, `{,}`, `[1,,]`,
`{"a":1,,}`, and `[1,}` are invalid.

#### Encoding and malformed bytes

1. Strict JSON without extension tokens MUST go directly to `encoding/json`;
   no global UTF-8 prevalidation is allowed. This preserves v1 behavior that
   replaces malformed UTF-8 and malformed UTF-16 surrogate pairs inside JSON
   strings with U+FFFD.
2. JSON syntax bytes are recognized byte-wise because all delimiters are ASCII.
3. A comment body is defined in terms of Unicode source characters. Malformed
   UTF-8 within a comment body is rejected with `ErrInvalidUTF8`. Invalid bytes
   outside comments remain for `encoding/json` to accept or reject.
4. Arbitrary binary input MUST terminate deterministically with a result or
   error and MUST NOT panic or hang.

#### Required edge-case outcomes

| Input class | Required outcome |
| --- | --- |
| `{"url":"https://x"}` | valid; `//` is string data |
| `{"s":"/* x */"}` | valid; delimiters are string data |
| `// c\n{"a":1}` | valid |
| `{"a":1}// EOF` | valid |
| `/**/1` | valid |
| `/* outer /* inner */ 1` | valid; first `*/` closes the sole comment |
| `/* unterminated` | invalid JSONC-specific syntax error |
| `{"a":1} /* unterminated` | invalid; complete prefix does not hide the error |
| comment-only input | invalid because a JSON value is required |
| `tr/*x*/ue` | invalid; normalization cannot create `true` |
| `# comment\n{}` | invalid |
| `[1, /* c */]` | valid FloraSync trailing-comma extension |
| `[,]`, `{,}`, `[1,,]` | invalid |
| CR, LF, and CRLF after `//` | each terminates the comment |
| U+2028/U+2029 after `//` | remains comment body under the pinned draft |
| valid non-ASCII UTF-8 in comments | valid and ignored |
| malformed UTF-8 in comments | `ErrInvalidUTF8`; never panic |
| malformed UTF-8 in JSON strings | exact `encoding/json` v1 behavior |

### Normalization contract

`Sanitize(data []byte) ([]byte, error)` is a FloraSync extension and SHALL:

1. never mutate `data`;
2. perform a non-recursive, deterministic lexical/structural pass;
3. replace every non-line-ending byte of a recognized comment with ASCII space;
4. preserve CR and LF bytes inside or terminating comments;
5. replace an accepted trailing-comma byte with one ASCII space;
6. preserve total byte length on success, so downstream byte offsets continue
   to identify positions in the original input;
7. preserve every byte not belonging to a recognized extension token;
8. ensure invalid token-splitting comments and invalid comma forms cannot become
   valid JSON after normalization;
9. return `nil` plus a JSONC-specific error for an unterminated block comment or
   malformed UTF-8 in a comment; and
10. not otherwise claim to validate JSON. Full validity remains the standard
    parser's responsibility.

Whether a no-op result aliases the input backing array is intentionally not a
caller contract. Callers MUST NOT depend on aliasing. The wrapper fast path
does not call `Sanitize` when no JSONC extension token is present.

The existing extension `func HasCommentRunes(data []byte) bool` SHALL remain
available for upstream API compatibility. It returns true only when an actual
`//` or `/*` opener occurs outside a JSON string, regardless of whether the
surrounding document or comment is complete and valid. It ignores
comment-looking string content, does not detect trailing commas, performs no
allocation, and never panics for arbitrary bytes.

JSONC-specific lexical failures SHALL use these extension contracts:

```go
var ErrInvalidUTF8 error
var ErrUnterminatedBlockComment error

type JSONCSyntaxError struct {
    Offset int64 // one-based byte offset in the original input
    Err    error // one of the JSONC sentinels
}

func (e *JSONCSyntaxError) Error() string
func (e *JSONCSyntaxError) Unwrap() error
```

The offset for malformed UTF-8 identifies the first malformed byte. The offset
for an unterminated block comment identifies the opening `/`. Callers can use
`errors.Is` for the category and `errors.As` for offset detail.

### encoding/json facade contract

#### Definition of drop-in compatibility

For supported default Go toolchains, changing only an import string from
`"encoding/json"` to `"github.com/FloraSync/go-jsonc"` SHALL compile for every
documented stable `encoding/json` symbol. On strict JSON, public documented
behavior, output bytes, concrete errors, and byte offsets SHALL match the
active toolchain's `encoding/json`; direct delegation and type aliases are the
default mechanism.

This is source compatibility, not impossible cross-package identity for the
custom `Decoder`. Code that simultaneously requires assignment between
`*encoding/json.Decoder` and the JSONC-aware `*json.Decoder`, relies on
reflection package paths, or relies on undocumented reader call timing is
outside the guarantee. All documented `Decoder` behavior remains in scope.

The required signatures called out by the program are:

```go
func Unmarshal(data []byte, v any) error
func Marshal(v any) ([]byte, error)
func NewDecoder(r io.Reader) *Decoder
func NewEncoder(w io.Writer) *Encoder
func Valid(data []byte) bool
```

Those five functions alone are insufficient for import-only compatibility.
The complete stable surface that MUST be present is listed below.

#### Complete stable package surface

```go
func Compact(dst *bytes.Buffer, src []byte) error
func HTMLEscape(dst *bytes.Buffer, src []byte)
func Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error
func Marshal(v any) ([]byte, error)
func MarshalIndent(v any, prefix, indent string) ([]byte, error)
func NewDecoder(r io.Reader) *Decoder
func NewEncoder(w io.Writer) *Encoder
func Unmarshal(data []byte, v any) error
func Valid(data []byte) bool
```

```go
type Decoder struct { /* unexported */ }
func (dec *Decoder) Buffered() io.Reader
func (dec *Decoder) Decode(v any) error
func (dec *Decoder) DisallowUnknownFields()
func (dec *Decoder) InputOffset() int64
func (dec *Decoder) More() bool
func (dec *Decoder) Token() (Token, error)
func (dec *Decoder) UseNumber()

type Encoder struct { /* unexported */ }
func (enc *Encoder) Encode(v any) error
func (enc *Encoder) SetEscapeHTML(on bool)
func (enc *Encoder) SetIndent(prefix, indent string)
```

The following named types, interfaces, methods, error types, and public fields
MUST also be available. Type aliases to `encoding/json` are preferred wherever
the JSONC-aware streaming decoder does not prevent aliasing.

```go
type Delim rune
func (d Delim) String() string

type Token any

type Number string
func (n Number) Float64() (float64, error)
func (n Number) Int64() (int64, error)
func (n Number) String() string

type RawMessage []byte
func (m RawMessage) MarshalJSON() ([]byte, error)
func (m *RawMessage) UnmarshalJSON(data []byte) error

type Marshaler interface { MarshalJSON() ([]byte, error) }
type Unmarshaler interface { UnmarshalJSON([]byte) error }

type InvalidUTF8Error struct { S string }
type InvalidUnmarshalError struct { Type reflect.Type }
type MarshalerError struct { Type reflect.Type; Err error /* plus unexported state */ }
type SyntaxError struct { Offset int64 /* plus unexported state */ }
type UnmarshalFieldError struct {
    Key string
    Type reflect.Type
    Field reflect.StructField
}
type UnmarshalTypeError struct {
    Value string
    Type reflect.Type
    Offset int64
    Struct string
    Field string
}
type UnsupportedTypeError struct { Type reflect.Type }
type UnsupportedValueError struct { Value reflect.Value; Str string }
```

All standard error types retain their standard `Error` methods;
`MarshalerError` also retains `Unwrap`. Deprecated `InvalidUTF8Error` and
`UnmarshalFieldError` remain required for source compatibility. The package has
no standard exported constants or variables.

#### Top-level behavior

- `Marshal`, `MarshalIndent`, `NewEncoder`, and encoder methods SHALL directly
  preserve standard-library behavior. Encoding emits strict JSON, never
  comments or trailing commas. `Encode` appends one newline; indentation and
  HTML-escaping options persist; writer errors remain sticky.
- `Unmarshal` SHALL validate the entire normalized document before decoding and
  preserve all standard conversion, struct-tag, duplicate-key, custom-method,
  number, invalid-string-encoding, and error-precedence behavior.
- If no actual comment delimiter or accepted trailing comma occurs outside a
  string, `Unmarshal` SHALL pass the original byte slice directly to
  `encoding/json.Unmarshal` with no JSONC allocation or UTF-8 prevalidation.
- `Valid(data)` SHALL equal
  `Sanitize(data) succeeds && encoding/json.Valid(normalized)` for the agreed
  profile, while using direct `encoding/json.Valid(data)` on the no-extension
  fast path. It examines the whole slice and accepts exactly one value.
- `Compact` and `Indent` SHALL accept the FloraSync profile, normalize it, and
  then apply the corresponding standard function. Their behavior on strict
  JSON remains byte-for-byte standard behavior.
- `HTMLEscape` has no error return and therefore SHALL remain the exact
  standard operation on its documented strict-JSON input; it is not a JSONC
  validator.

For JSONC input, `RawMessage` destinations and caller-defined
`Unmarshaler.UnmarshalJSON` methods receive the length-preserving normalized
strict JSON view, never comment bytes. This prevents ignored comments from
becoming a second parser channel.

#### Decoder behavior

The JSONC-aware `Decoder` SHALL preserve the complete documented streaming
contract:

- `Decode` reads one value and permits later concatenated values; no eager
  `io.ReadAll` is allowed.
- JSONC tokens and delimiters may span arbitrary reader chunk boundaries,
  including `/`, `*`, `*/`, `//`, CRLF, UTF-8 sequences, escaped strings, and
  a trailing comma separated from its closing delimiter by comments.
- Comments may occur between streamed top-level values.
- `UseNumber` and `DisallowUnknownFields` persist for subsequent operations.
- `Token` elides commas/colons, enforces delimiter nesting, and returns the
  standard dynamic token types, including the replacement package aliases for
  `Delim` and `Number`.
- `InputOffset` reports the original input's byte offset. Length-preserving
  normalization makes normalized and original offsets identical.
- `Buffered` exposes the unread normalized view. It is byte-identical to the
  unread original input on the strict/no-extension path and is valid until the
  next `Decode`, matching the standard lifetime contract.
- `io.EOF`, truncated input, read errors, syntax errors, semantic errors, and
  sticky-error behavior match `encoding/json` wherever the standard contract
  applies. A JSONC lexical error poisons subsequent decode operations in the
  same manner as a standard syntax/read error.

### Performance and security invariants

1. Runtime/build dependency graph: no non-standard-library Go modules. Test
   assertions SHALL use the standard library. Alternative JSON backends and
   their build tags are out of scope for the zero-dependency facade.
2. Time complexity: O(n) over input bytes; no regular expressions, backtracking,
   quadratic rescans, or input-controlled recursion.
3. Space complexity: O(1) lexer state plus O(depth) structural state and at most
   one O(n) normalized buffer when a transformation is required. The no-change
   `Unmarshal` and `Valid` paths add no buffer allocation.
4. The library adds no arbitrary document-size limit that would reject input
   accepted by `encoding/json`. Structural depth SHALL not be more permissive
   than the standard parser.
5. Comment removal and comma handling MUST be fail-closed: malformed extension
   syntax cannot become a different valid JSON value.
6. Error offsets MUST remain original-input byte offsets.
7. Parser differentials are security-sensitive. Duplicate keys,
   case-insensitive struct matching, unknown-field handling, large-number
   precision, and invalid string encoding intentionally inherit
   `encoding/json` v1 behavior rather than inventing stricter defaults.

### Repository baseline findings

Read-only inspection of HEAD `2fa72da` found:

- `go.mod` still declares `github.com/marcozac/go-jsonc`; root/internal imports,
  README links, and examples use the upstream identity.
- The package is named `jsonc`, so a bare import-string swap from
  `encoding/json` does not currently compile.
- The only public implementation file exposes `ErrInvalidUTF8`, `Sanitize`,
  `Unmarshal`, and `HasCommentRunes`. Four of the five required facade functions
  and both facade types are absent.
- The module contains third-party requirements for goccy/go-json,
  json-iterator, testify, and their indirect modules. Alternate build tags can
  select non-standard runtime backends.
- There is no trailing-comma support.
- The fast-path detector is not string-aware and intentionally reports comment
  markers inside strings, causing unnecessary sanitization/allocation.
- The sanitizer deletes bytes instead of preserving whitespace/offsets, can
  join tokens, does not report unterminated block comments, mishandles some
  escape parity and repeated-star endings, and terminates line comments only on
  LF.
- Positive finding: the current sanitizer is linear and regex-free and does
  not index slices directly; no obvious regex-DoS or direct out-of-bounds path
  was found in this Phase 1 inspection.
- Tests lack trailing-comma, malformed-comment, CR-line-ending, escape-parity,
  exact-output, facade, streaming, differential, and fuzz coverage.
- CI covers Go 1.19/1.20 with race tests and alternate backends, but has no
  `govulncheck` or fuzz stage and uses several floating action/tool versions.

These are findings, not Phase 1 implementation authorization. Remediation is
owned by the dependent Phase 2 through Phase 4 stories.

### Approval decisions

Approval of this story accepts all of the following together:

1. **Profile:** adopt the pinned FloraSync JSONC Profile v1, including always-on
   trailing commas as a documented extension to JSONC.org's default grammar.
2. **Unicode:** reject malformed UTF-8 only inside comments; preserve standard
   v1 behavior elsewhere. Treat U+2028/U+2029 according to the current pinned
   grammar and monitor the upstream security discussion.
3. **Normalization:** use length-preserving ASCII-space substitution and expose
   normalized JSON to `RawMessage`, custom unmarshalers, and `Buffered`.
4. **Facade breadth:** mirror the entire stable `encoding/json` surface, not
   only the five initially named functions.
5. **Package identity:** rename the declared package to `json` for a literal
   import-string-only migration. Existing callers relying on the implicit
   identifier `jsonc` will need an import alias or call-site rename.
6. **Go support:** require Go 1.26.0 or newer, carry no pre-1.26 compatibility
   baggage, test the exact 1.26.0 baseline plus current Go 1.26 and Go 1.27,
   retain the v1 facade contract, and test explicit `encoding/json/v2`
   composition on Go 1.27 without implicitly changing semantic versions.
7. **Zero dependencies:** remove every non-standard-library Go module and all
   alternative backend build tags from the shipping module.
8. **Compatibility meaning:** guarantee documented source/behavior
   compatibility; acknowledge the unavoidable `*Decoder` cross-package type
   identity limitation.

## Requirements

- Define the supported JSONC grammar and every identified edge case.
- Define the complete stable `encoding/json` facade and behavioral contract.
- Define malformed-input, offset, normalization, streaming, performance, and
  dependency invariants.
- Record repository baseline findings without changing implementation.
- Keep Phase 2 blocked until explicit owner approval.

## Constraints

- Do not edit implementation code, tests, CI, module metadata, README, or
  benchmarks in Phase 1.
- Do not run implementation or mutation workflows.
- Do not begin any dependent story until the repository owner explicitly
  approves this packet.

## Acceptance criteria

- [x] Normative rules cover comment placement, both comment forms, strings,
  whitespace, Unicode, malformed input, and trailing commas.
- [x] An edge-case matrix states testable expected outcomes.
- [x] The five requested signatures and the full stable facade are recorded.
- [x] Strict-JSON, JSONC, streaming, error, offset, and callback behavior are
  specified.
- [x] Performance and security invariants are measurable.
- [x] Repository flaws and future story ownership are recorded.
- [x] The repository owner explicitly approved all eight decisions above on
  2026-08-14.

## Technical approach

Phase 1 used Sprout's Lay of the Land stage only:

1. inventory the repository, public API, dependencies, tests, and CI;
2. compare the current behavior with pinned JSONC.org grammar and RFC 8259;
3. inventory the complete stable `encoding/json` surface and documented
   semantics;
4. resolve specification ambiguities into proposed, testable profile rules;
5. create dependent stories but leave them blocked; and
6. request approval before any tracer-bullet implementation.

No tracer bullet or implementation is part of this story.

## Execution checklist

- [x] Inspect relevant source, tests, workflows, module metadata, and repository
  instructions.
- [x] Review pinned JSONC.org prose and ABNF plus RFC 8259.
- [x] Inventory stable `encoding/json` functions, methods, types, errors, and
  key documented behavior.
- [x] Define the FloraSync JSONC Profile v1 and compatibility boundaries.
- [x] Create blocked Sprout stories for Phases 2, 3, and 4.
- [x] Receive explicit repository-owner approval.
- [x] Designate Phase 2 as the next authorized story without starting its
  implementation during Phase 1 closure.

## Verification

- [x] `sprout check define-jsonc-stdlib-compatibility-contracts`
- [x] Confirm `git status --short --untracked-files=all` contains
  specification/Sprout artifacts only.
- [x] Confirm no `SPEC_LOG.md` exists.

## Validation evidence

- Repository inspected at HEAD `2fa72da` on branch `main`.
- Local toolchain used for API inventory: Go 1.25.12.
- Current stable Go 1.26.6 source and the pinned JSONC.org commit were reviewed
  from the authoritative upstream repositories on 2026-08-14.
- No tests, formatters, generators, dependency changes, or implementation
  commands were run in Phase 1.
- `sprout check define-jsonc-stdlib-compatibility-contracts` returned `ok`.
- `sprout doctor` passed workspace, repository-instruction, local/global skill,
  and available-agent checks; it also validated all four open task packets.
- `git status --short --untracked-files=all` listed only `AGENTS.md`, Sprout
  initialization files, and the four story packets. `SPEC_LOG.md` is absent.
- The repository owner replied `Approve Phase 1` on 2026-08-14, explicitly
  accepting all eight approval decisions in this packet.
- After approval, the repository owner amended decision 6 on 2026-08-14:
  Go 1.26.0 and above is the only supported line; pre-1.26 upgrade baggage is
  explicitly out of scope.

## Outcome and follow-ups

Status: **approved and complete as of 2026-08-14**.

All eight contract decisions are now binding for the dependent stories. Phase
2 is unblocked and is the next authorized story, but no Phase 2 implementation
was started while closing this Phase 1 packet.

Dependent stories:

- `align-encoding-json-facade-and-sanitizer` (Phase 2)
- `harden-jsonc-security-and-supply-chain` (Phase 3)
- `fuzz-jsonc-adversarial-inputs` (Phase 4)
- `qualify-go-library-ci-matrix` (Phase 5)
- `harden-initial-release-pipeline` (Phase 6)
- `complete-final-release-legal-review` (Phase 7 release gate, after the exact
  release candidate and release path are assembled)

After approval, Phase 2 must begin with a minimal end-to-end strict-JSON and
JSONC tracer bullet, then proceed through the Sprout hot path, safe passage, and
closing review gates. Phase 3 and Phase 4 remain separate reviewable work.
Those gates may close while the shared tree remains intentionally dirty; final
release remains blocked until Phase 7 classifies the complete candidate and
records qualified legal clearance.

## Original request

Refactor, harden, and audit FloraSync/go-jsonc into a secure, zero-dependency,
`encoding/json`-compatible JSONC facade supporting line comments, block
comments, and trailing commas, using Sprout and go-coder. Log specification
work as Sprout stories and do not touch implementation until contracts are
approved.
