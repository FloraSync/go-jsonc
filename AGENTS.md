# Repository agent instructions

Follow the repository's established conventions and keep changes focused.

## Shared working-tree and release policy

- The coordinated Sprout work packages intentionally accumulate in one dirty
  worktree until the final release candidate is assembled. A dirty worktree is
  expected and is not, by itself, a task failure.
- Treat `git status --short --untracked-files=all` as an inventory and scope-
  control tool, not as a clean-tree gate. Preserve pre-existing, user-owned,
  and concurrent changes; do not reset, revert, stash, discard, or hide them
  merely to make the tree appear clean.
- Before changing files, identify the task-owned portion of the shared change
  set. At closing review, record what the task changed and any remaining
  working-state artifacts or follow-ups.
- No public release, tag, module publication, or artifact distribution is
  cleared until `complete-final-release-legal-review` records review of the
  exact release candidate by the designated qualified legal reviewer. Agents
  cannot self-approve legal clearance, and privileged or sensitive advice must
  not be copied into repository documentation.

When executing a task from .sprout/tasks/:

- Read the entire task packet before editing.
- Lay of the Land: Inspect relevant code, tests, and repository instructions.
- For `align-encoding-json-facade-and-sanitizer`,
  `harden-jsonc-security-and-supply-chain`, and
  `fuzz-jsonc-adversarial-inputs`, read
  `.sprout/research/node-jsonc-parser-implementation-map.md` during Lay of the
  Land. Treat it as non-normative working research; do not promote it into
  product documentation, and keep the approved contract and active task packet
  authoritative.
- Tracer Bullet: First prove feasibility via the shortest possible end-to-end path.
- Hot Path Exploration: Build out the primary execution path and core logic.
- Safe Passage: Harden the implementation, execute all verification items, and resolve any regressions.
- Closing Review: Document final technical approach, record validation evidence,
  inventory the task-owned changes, and prepare the packet for sprout close.
  Closing one task does not require the shared worktree to be clean and does not
  imply final legal or release clearance.
