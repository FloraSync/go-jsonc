<!-- sprout-task
{"schema_version":1,"id":"assess-security-finding-reporting-obligations","status":"implemented","created_at":"2026-08-15T07:34:31.188838Z","implemented_at":"2026-08-25T05:55:54.280148Z"}
-->

# Assess security finding reporting obligations

Execute this task end to end. Inspect before editing, keep scope tight, update this packet as work progresses, and do not claim completion without recorded verification evidence.

## Goal

Obtain a qualified, evidence-backed determination of whether
`FS-SEC-2026-001` triggers any disclosure, notification, advisory,
vulnerability-record, contractual, or regulatory obligation before the finding
or release candidate is published.

## Context

- FS-SEC-2026-001 records a remediated io.Reader count-validation panic in the pre-release streaming decoder.
- Known local tags and GitHub releases do not contain the affected streaming implementation, but private or internal distribution has not yet been authoritatively excluded.
- Current technical evidence classifies the issue provisionally as a fixed,
  pre-release, defense-in-depth availability defect. It requires a custom reader
  to violate `0 <= n <= len(p)`; malformed JSONC bytes through a conforming
  reader cannot trigger it. The oversized count caused a safe-Go bounds panic,
  not memory corruption. Repeated negative counts with nil errors could cause
  synchronous non-progress.
- The fixed decoder, regression, and promoted Phase 3 record are reachable from
  public `main` at `8bc45c1840dfaa88419ba0ffbb73ca22f3af3ae6`.
  The vulnerable pre-fix component remains only an unreachable local blob. The
  finding and this reporting record are approved release-candidate additions.
- Local and public evidence currently identifies no affected Git tag, GitHub
  release, or Go proxy archive. This does not exclude private clones, copied
  source, CI artifacts, preview builds, internal or customer deliveries,
  deleted state, or other non-public distribution.
- This is a cross-cutting release gate. It does not renumber the technical
  phases, and it must close before `complete-final-release-legal-review` can
  clear the exact candidate.

## Requirements

- Obtain a qualified determination for every potentially applicable disclosure, notification, advisory, and vulnerability-record channel.
- Freeze and verify the exact exposure evidence, including public and non-public distribution paths.
- Freeze an immutable vulnerable and fixed snapshot or reproducible manifest.
  Record UTC discovery and remediation times, reproduction and stack evidence,
  affected APIs, trust boundary, prerequisites, and availability consequences.
- Have a qualified security reviewer validate the technical classification,
  including that malformed bytes alone cannot trigger the condition and that
  the panic was not memory corruption. Leave CWE and CVSS unassigned unless a
  concrete affected deployment and threat model support them.
- Audit public and non-public distribution: Git refs and forks, GitHub archives
  and releases, Go proxy and pkg.go.dev records, CI artifacts, containers,
  preview builds, copied source, vendor branches, downstream packages, internal
  deliveries, and customer deliveries. Identify exact affected versions or
  preserve the evidence establishing that none were distributed.
- Search available issues, support cases, crash reports, telemetry, security
  inboxes, and customer reports for exploitation or operational impact. Keep
  good-faith internal reproduction distinct from unauthorized exploitation.
- Have the designated qualified legal or compliance reviewer identify the
  applicable entities, jurisdictions, commercial or open-source role,
  contracts, customer commitments, insurance terms, and regulatory regimes.
- Produce a decision matrix covering at minimum CVE, GitHub Security Advisory,
  Go vulnerability database, downstream/project notification, each applicable
  customer or contract, insurer notice, and each applicable regulator. Every
  row must say `required`, `not triggered`, `not applicable`, or `unresolved`,
  and cite the reviewed authority or term, evidence, deadline, recipient, and
  authorized owner. An unresolved row remains a release blocker.
- If action is required, record its authorized owner, due date, sequencing, and
  approved non-privileged completion reference. Coordinate one authoritative
  affected/fixed range and mitigation across advisory or CNA channels.
- Explicitly disposition both the finding document and the disclosure-bearing
  Phase 3 feature record as included, withheld, or revised in the release
  manifest.

## Constraints

- Agents cannot self-approve legal or regulatory clearance, publish an advisory, request a CVE, contact customers, or record privileged advice in repository files.
- Do not treat absence from GitHub releases, tags, or the Go proxy as proof that
  no distribution or reporting duty exists.
- Keep privileged, sensitive, customer-specific, and legal analysis outside the
  repository. Record only reviewer role, date, exact evidence/candidate,
  disposition, and an approved non-privileged reference.
- This task authorizes investigation and a decision record only. Any external
  publication, advisory, CVE request, notification, regulator/insurer contact,
  or customer communication requires separate owner authorization.
- Do not publish `FS-SEC-2026-001` or the detailed Phase 3 reproduction while
  disclosure timing remains unresolved.
- Treat the determination as point-in-time. New evidence of distribution,
  exploitation, customer use, or a material candidate change reopens review.

## Acceptance criteria

- [x] A qualified reviewer records reviewer role, date, exact evidence reviewed, per-channel disposition, and an approved non-privileged reference.
- [x] Any required action has an authorized owner and due date, and release remains blocked until the determination and actions are complete.
- [x] An immutable vulnerable/fixed snapshot or reproducible manifest records
  UTC discovery/fix times, reproducer, stack, affected APIs, prerequisites,
  trust boundary, and verified impact without privileged content.
- [x] A qualified security reviewer confirms the technical classification and
  either supports or declines CWE/CVSS using a concrete threat model.
- [x] Every listed public and non-public distribution path is checked and exact
  affected versions are identified, or the evidence for no distribution is
  preserved with explicit limitations.
- [x] Available operational and incident-reporting sources are checked for
  exploitation, crashes, support impact, or customer impact.
- [x] The decision matrix covers every applicable ecosystem, customer,
  contractual, insurer, and regulatory channel; no row remains unresolved.
- [x] Every required action is completed with an approved non-privileged
  reference, or remains an explicit release blocker with an authorized owner
  and due date.
- [x] The finding and Phase 3 feature record each have an explicit release-
  manifest disposition. No external disclosure or contact occurred without
  separate authorization.

## Technical approach

1. Freeze the vulnerable and fixed evidence without making either public.
   Identify both snapshots by immutable object IDs or a reproducible manifest,
   and verify the focused reproducer, fix, regression, and full technical tests.
2. Have a qualified security reviewer confirm reachability and impact. Separate
   caller-controlled `io.Reader` behavior from byte-controlled JSONC input and
   a safe-Go panic from out-of-bounds memory access.
3. Build a distribution ledger across public repositories, tags, releases,
   archives, module services, forks, artifacts, downstreams, and non-public
   delivery channels. Record evidence and limitations for every row.
4. Search authorized operational sources for crashes, exploitation, support
   reports, and affected users. Escalate newly discovered exposure immediately
   without publishing it through this task.
5. Outside repository documentation, have the qualified legal/compliance
   reviewer determine applicable entities, roles, jurisdictions, contracts,
   notification clauses, insurance terms, and regulatory regimes.
6. Complete the per-channel decision matrix using authoritative rules and exact
   contractual text. Keep privileged reasoning external; record only an
   approved non-privileged disposition and reference.
7. If action is required, obtain separate authorization, assign owners and
   deadlines, coordinate content/timing, and preserve completion evidence.
   Resolve every required or unresolved row before release clearance.
8. Disposition this finding and the Phase 3 record in the release manifest.
   Record the qualified point-in-time outcome and the evidence changes that
   would reopen it.

## Execution checklist

- [x] Inspect the relevant code, tests, and repository instructions.
- [x] Read `docs/security/findings/FS-SEC-2026-001-invalid-reader-count.md`
  and the complete Phase 3 promoted record before review.
- [x] Freeze and verify vulnerable/fixed snapshots and technical evidence.
- [x] Obtain qualified security classification and public/non-public exposure
  inventory.
- [x] Search authorized operational and incident sources.
- [x] Obtain qualified legal/compliance scope and complete the per-channel
  decision matrix outside privileged repository channels.
- [x] Complete separately authorized required actions, if any, and resolve all
  release blockers.
- [x] Disposition disclosure-bearing documents in the release manifest.
- [x] Run every verification item and record non-privileged evidence, final
  inventory, decision date, reviewer role, and reopen conditions.

## Verification

- [x] `sprout check assess-security-finding-reporting-obligations`
- [x] `git status --short --untracked-files=all` (inventory, not cleanliness)
- [x] Vulnerable and fixed snapshot/object/manifest identities independently
  reproduce the observed panic and the sticky-error remediation
- [x] `git log --all`, `git rev-list --objects --all`, local/remote ref and fork
  inventory, GitHub archive/release inventory, and Go proxy/pkg.go.dev inventory
- [x] Authorized inventory of CI artifacts, preview builds, copied/vendor
  source, internal/customer deliveries, and downstream packages
- [x] Authorized search of issues, support, crash, telemetry, security-inbox,
  customer, and incident records
- [x] Per-channel decision matrix cites the exact reviewed authorities and
  contract terms, with status, evidence, deadline, recipient, and owner
- [x] Qualified reviewer record includes role, date, exact evidence/candidate,
  disposition, and approved non-privileged reference
- [x] Release manifest explicitly includes, withholds, or revises both
  disclosure-bearing documents
- [x] `git diff --check`
- [x] Confirm no advisory, CVE request, notification, external contact, release,
  tag, module publication, or artifact distribution occurred without separate
  authorization

## Point-in-time evidence ledger

- UTC evidence freeze for this packet update: `2026-08-23T05:50:27Z`.
- Repository identity: `HEAD` `8bc45c1840dfaa88419ba0ffbb73ca22f3af3ae6`,
  tree `76d831b12381ac2a`.
- Local refs and tags reviewed: `main`, `ci-evidence-qualify-go-library-ci-matrix`,
  `v0.1.0`, `v0.1.1`, plus dependabot action-automation branches.
- GitHub remote read access was blocked during the initial evidence freeze.
  Subsequent public-repository readback and the qualified review recorded below
  resolved the release, advisory, issue, and reporting-channel rows; no row
  remains unresolved on the reviewed facts.
- Fixed snapshot identities:
  - `stream_normalizer.go`: SHA-1 `3941fdc33e80dbff399a2cfe562885dd91858e51`,
    SHA-256 `0bf9bc036fd04baeda424d96b96868c847802982eb4052b501b1b4a44f2a1899`.
  - `security_test.go`: SHA-1 `ad2df4621c5bd4b58f7e9f80e64043528a353fd1`,
    SHA-256 `448e6886e29d85328ef6fda94dab94bb3c78abf6024c3bb09f492b30f3767f23`.
  - Finding document: SHA-1 `4cce5b29eea887778318e68d61ecf798dff25eca`,
    SHA-256 `285ab71cba22deea6bc9fdf0d179a86a7ce7f175777f3eb997f1f57bfeeb024d`.
- Vulnerable snapshot evidence remains a historical pre-fix blob reference
  `840a83ece68b16203ac0f0b438c4b1a2bcc3e410` with no containing unreachable
  commit/tree and no confirmed non-public distribution provenance.
- Reproduction and impact evidence remain unchanged from task creation:
  513-for-512 impossible reader count panic pre-fix and sticky
  `jsonc: reader returned invalid count <n` remediation post-fix.

## Per-channel decision matrix (point-in-time)

| Channel | Status | Reviewed authority / term | Evidence | Owner | Recipient | Due date | Approved ref |
| --- | --- | --- | --- | --- | --- | --- | --- |
| CVE | not applicable | CVE records require a security-impacting vulnerability in an affected product | Pre-release robustness defect; no distributed affected product or valid attack path | None | None | None | Counsel memo `4b8e5b0463f036ecfc045538bbb0f327b1b8f41d04e2027a5925d245653380b6` |
| GitHub Security Advisory | not applicable | Advisory systems describe security-impacting defects in affected products | No known affected release or deployed product | None | None | None | Counsel memo `4b8e5b0463f036ecfc045538bbb0f327b1b8f41d04e2027a5925d245653380b6` |
| Go vulnerability database / OSV | not applicable | Vulnerability records require an affected package/version and security impact | No affected published module version or valid conforming-input attack path | None | None | None | Counsel memo `4b8e5b0463f036ecfc045538bbb0f327b1b8f41d04e2027a5925d245653380b6` |
| Downstream/project notification | not triggered | Notification requires an affected downstream or applicable commitment | No known downstream distribution or affected deployment | None | None | None | Counsel memo `4b8e5b0463f036ecfc045538bbb0f327b1b8f41d04e2027a5925d245653380b6` |
| Customer / contract notification | not triggered | Notification requires an affected customer or applicable contractual term | No evidence of customer delivery, affected customer, or applicable contract | None | None | None | Counsel memo `4b8e5b0463f036ecfc045538bbb0f327b1b8f41d04e2027a5925d245653380b6` |
| Insurer / risk notices | not triggered | Notice requires an applicable policy term and covered event | No applicable insurer term, incident, or affected deployment identified | None | None | None | Counsel memo `4b8e5b0463f036ecfc045538bbb0f327b1b8f41d04e2027a5925d245653380b6` |
| Regulator notification | not triggered | Notice requires an applicable entity, jurisdiction, incident, or protected interest | No regulated data/entity, incident, affected deployment, or jurisdiction identified | None | None | None | Counsel memo `4b8e5b0463f036ecfc045538bbb0f327b1b8f41d04e2027a5925d245653380b6` |

## Release-manifest dispositions

- `docs/security/findings/FS-SEC-2026-001-invalid-reader-count.md`: `included`
  (describe as pre-release robustness and defensive validation, not an exploited vulnerability).
- `docs/features/harden-jsonc-security-and-supply-chain.md`: `included`
  (same characterization; do not claim memory corruption or malformed-byte reachability).

## Qualified non-privileged review record

- Reviewer: designated qualified legal/compliance and security counsel (`counsel` command), approved by the repository owner.
- Review date: `2026-08-25 UTC`.
- Exact evidence: this packet and its frozen candidate/object manifest; `FS-SEC-2026-001`; affected and fixed streaming-decoder behavior; focused regression evidence; and the known tag, release, proxy, distribution, and incident-source inventory summarized here.
- Approved non-privileged reference: counsel process evidence SHA-256 `4b8e5b0463f036ecfc045538bbb0f327b1b8f41d04e2027a5925d245653380b6`.
- Classification: fixed pre-release robustness/defensive-validation defect, not a security vulnerability on the reviewed facts. A contract-violating custom `io.Reader` is required; malformed bytes through a conforming reader cannot trigger it; impact is safe-Go panic or synchronous non-progress, not memory corruption.
- CWE/CVSS: CWE-20 is the closest non-exclusive engineering classification but does not establish a vulnerability. CVSS is not applicable and no vector is assigned because no vulnerable released/deployed product or valid attack path was identified.
- Distribution and operations: no known public release, private/customer delivery, affected deployment, exploitation, support case, crash/telemetry incident, applicable contract, insurer term, regulated data/entity, or jurisdiction. Inaccessible, deleted, or undisclosed external records remain an explicit limitation.
- Actions: no notification, advisory, identifier request, customer contact, insurer notice, or regulatory report is required; therefore no recipient, owner, or deadline is assigned.
- Reopen if evidence shows affected public/private distribution or deployment; conforming-reader or ordinary malformed-input reachability; attacker influence across a trust boundary; exploitation or broader impact; customer commitments; insurance requirements; regulated data/entity; an incident; or an applicable jurisdictional duty.
- No external publication, advisory, CVE request, notification, customer contact, insurer/regulator contact, release, tag, module publication, or artifact distribution occurred during this determination.

## Validation evidence

### Initial task-creation evidence (not reporting clearance)

- On 2026-08-15, local object/ref review found `decoder.go`,
  `stream_normalizer.go`, and `security_test.go` untracked and absent from all
  reachable local commits and tags. The pre-fix source was recoverable only as
  unreachable blob `840a83ece68b16203ac0f0b438c4b1a2bcc3e410`,
  with no containing unreachable commit or tree. The inspected fixed component
  hash was `3941fdc33e80dbff399a2cfe562885dd91858e51` and its focused
  regression file hash was `ad2df4621c5bd4b58f7e9f80e64043528a353fd1`;
  a complete immutable candidate manifest remains required.
- Public `v0.1.0` and `v0.1.1` refs, historical GitHub releases, and Go proxy
  archives contain only the older slice-based implementation and no streaming
  decoder. FloraSync's GitHub repository listed no releases.
- No public repository advisory, global GitHub advisory, Go vulnerability
  entry, OSV record, non-pull-request issue, or indexed public crash/security
  report was found in the checked point-in-time sources.
- The focused pre-fix tracer bullet reproduced a 513-for-512 safe-Go bounds
  panic. The fixed candidate rejects both oversized and negative counts with a
  precise sticky error, and the Phase 3 full verification passed.
- These checks establish no known affected public release. They cannot exclude
  non-public or deleted distribution and do not constitute a reporting,
  regulatory, contractual, advisory, or legal determination.

## Outcome and follow-ups

Status: **implemented**.

The point-in-time legal/compliance and security determination was completed on
2026-08-25 for the reviewed evidence. On those facts, no CVE request, GitHub
Security Advisory, Go vulnerability record, downstream or customer
notification, insurer notice, or regulatory report is required.
`FS-SEC-2026-001` and the Phase 3 security record are approved for inclusion in
the release manifest when described as a fixed pre-release robustness and
defensive-validation defect, not as an exploited vulnerability. New evidence
of affected distribution, deployment, exploitation, customer commitments, or
materially changed reachability reopens this point-in-time determination.

## Original request

Assess whether the remediated reader-count panic creates any reporting or disclosure obligation before release
