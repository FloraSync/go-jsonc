# FS-SEC-2026-001: Streaming decoder trusted impossible reader counts

> Public non-privileged engineering record. Approved for release-candidate
> inclusion under the completed point-in-time reporting determination. This
> record describes a fixed pre-release robustness and defensive-validation
> defect, not an exploited vulnerability.

## Finding status

| Field | Value |
| --- | --- |
| Finding ID | `FS-SEC-2026-001` |
| Status | Fixed before the first FloraSync release |
| Discovery and remediation date | 2026-08-15 |
| Component | `streamNormalizer.Read`, reached through `NewDecoder` |
| Trust boundary | Caller-supplied `io.Reader` implementation |
| Technical impact | Availability only; panic or synchronous non-progress |
| Known affected releases | None identified on reviewed evidence |
| CWE | Not assigned; CWE-20 noted only as a non-exclusive engineering classification |
| CVSS | Not applicable on reviewed facts |
| Reporting determination | No reporting or notification action required as of 2026-08-25; reopen conditions apply |
| Release-manifest disposition | Included |

## Summary

Before remediation, the streaming normalizer trusted the count returned by its
source reader and sliced an internal buffer with that value. A custom reader
returning a count greater than the supplied buffer length caused a Go runtime
slice-bounds panic. A reader repeatedly returning a negative count with a nil
error could keep the synchronous normalizer read loop retrying without progress
and consume CPU.

The Go [`io.Reader`](https://pkg.go.dev/io#Reader) contract requires
`0 <= n <= len(p)`. The trigger therefore requires a defective or hostile
caller-supplied reader. Malformed JSONC bytes delivered through a conforming
reader cannot select an impossible count and cannot trigger this defect. The
slice-based APIs are not affected.

Go's bounds check stopped the oversized slice operation. No out-of-bounds
memory access, memory corruption, confidentiality loss, or integrity loss was
identified. Depending on where callers recover panics, the availability effect
could be limited to one request or goroutine, or could terminate a process.

## Preconditions and observed behavior

Two contract violations reached the unchecked boundary:

- `n > len(p)`: the normalizer evaluated `scratch[:n]` and panicked before
  processing input;
- repeated `n < 0` with `err == nil`: the normalizer neither processed bytes,
  returned no-progress, nor terminated, so the synchronous read loop retried.

A single negative result did not by itself prove an indefinite loop; continued
non-progress required the reader to repeat the invalid result. No evidence of
exploitation or operational impact has been identified in the repository
review performed so far.

## Root cause and remediation

The adapter enforced neither bound of the reader's returned byte quantity at
its trust boundary. It now validates the count immediately after `Read` and
before any slice expression:

- counts below zero or above the supplied buffer length terminate with the
  precise error `jsonc: reader returned invalid count <n>`;
- that terminal error is sticky across later decoder calls; and
- committed output semantics remain unchanged for conforming readers.

The fix is in `stream_normalizer.go`. The focused regression is
`TestDecoderRejectsInvalidReaderCounts` in `security_test.go`.

## Reproduction and verification evidence

The pre-fix tracer bullet used `NewDecoder` with a custom reader that reported
513 bytes for a 512-byte destination. It reproduced:

```text
panic: runtime error: slice bounds out of range [:513] with capacity 512
```

The local vulnerable component snapshot hashes to Git blob
`840a83ece68b16203ac0f0b438c4b1a2bcc3e410`. The inspected fixed
`stream_normalizer.go` hashes to
`3941fdc33e80dbff399a2cfe562885dd91858e51`; its focused regression file
hashes to `ad2df4621c5bd4b58f7e9f80e64043528a353fd1`. These component IDs preserve
the point-in-time technical comparison, but they are not a complete release-
candidate manifest.

After the guard was added, focused tests for oversized and negative counts
returned the expected sticky errors. The complete pre-release candidate also
passed the following checks:

- Go 1.26.0: `go test -mod=readonly -count=1 ./...`;
- Go 1.26.6: `go test -mod=readonly -race -count=1 ./...` and `go vet ./...`;
- golangci-lint v2.12.0 with zero findings;
- govulncheck v1.7.0 with zero reachable vulnerabilities; and
- formatting, module-graph, actionlint, diff, and shell-syntax checks.

These results establish remediation in the inspected candidate. The completed
point-in-time reporting determination is recorded below.

## Preliminary exposure evidence

This is the technical inventory reviewed by the completed point-in-time
reporting determination. It is not final release-candidate legal clearance.

- The fixed `decoder.go`, `stream_normalizer.go`, and `security_test.go` are
  reachable from public `main` at
  `8bc45c1840dfaa88419ba0ffbb73ca22f3af3ae6`.
- The vulnerable pre-fix source survives locally only as an unreachable blob;
  no unreachable commit or tree contains it. It is not reachable from a known
  public Git ref.
- Public tags `v0.1.0` and `v0.1.1` contain the older slice-only implementation,
  no `NewDecoder` or streaming-normalizer API, and no affected code path.
- The FloraSync GitHub repository has no GitHub releases. The historical
  upstream releases and Go proxy archives for `v0.1.0` and `v0.1.1` likewise
  contain only the older implementation.
- The point-in-time review found no public repository advisory, global GitHub
  advisory, Go vulnerability entry, OSV record, non-pull-request issue, or
  indexed public crash/security report for either module path.
- No known public release is therefore established as affected by this finding.

The repository evidence cannot exclude private clones or mirrors, copied source,
CI artifacts, preview builds, internal or customer deliveries, vendor branches,
deleted remote state, or other non-public distribution. It also does not decide
contractual, regulatory, insurer, customer, advisory, CVE, or ecosystem duties.

## Reporting determination and reopen conditions

The completed
[`assess-security-finding-reporting-obligations`](../../features/assess-security-finding-reporting-obligations.md)
review determined that no advisory, CVE request, Go vulnerability record,
downstream or customer notification, insurer notice, or regulatory report is
required on the evidence reviewed as of 2026-08-25. New evidence of affected
distribution or deployment, conforming-reader reachability, exploitation,
customer commitments, insurance requirements, regulated interests, or a
material candidate change reopens that point-in-time determination.

The final release gate
[`complete-final-release-legal-review`](../../features/complete-final-release-legal-review.md)
also requires this reporting review to close and all required actions to be
resolved. Privileged or sensitive advice stays outside repository files; only
an approved non-privileged disposition may be recorded here.

## Release-manifest disposition note

- `FS-SEC-2026-001` is **included** in the release manifest as a fixed
  pre-release robustness and defensive-validation defect.
- The Phase 3 disclosure-bearing feature record
  [`harden-jsonc-security-and-supply-chain`](../../features/harden-jsonc-security-and-supply-chain.md)
  is included under the same characterization.

## Related records

- [`harden-jsonc-security-and-supply-chain`](../../features/harden-jsonc-security-and-supply-chain.md)
- [`assess-security-finding-reporting-obligations`](../../features/assess-security-finding-reporting-obligations.md)
- [`complete-final-release-legal-review`](../../features/complete-final-release-legal-review.md)
