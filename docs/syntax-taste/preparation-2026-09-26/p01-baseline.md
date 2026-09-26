# P01 baseline, authority and scope — 2026-09-26 round

## P01.1 Current revision and working tree

- Preparation executed at `2cb1bc35435ab5075d51f9f945adfb0e1f9dfe64`
  (2026-09-26, "UP26/UP27 post-upgrade completion docs and qualification evidence").
- Reviewed revision in
  [full-language-review-2cb1bc3-2026-09-26.md](../full-language-review-2cb1bc3-2026-09-26.md):
  `2cb1bc35435ab5075d51f9f945adfb0e1f9dfe64`. **Baseline equals reviewed revision;
  zero code drift.**
- Working tree at preparation start (`git status --short`): clean except three
  untracked documentation inputs, all part of this round's source material:
  - `docs/syntax-taste/can-design-preparation-2026-09-26.md` (this checklist)
  - `docs/syntax-taste/full-language-review-2cb1bc3-2026-09-26.md` (source review)
  - `docs/syntax-taste/evidence/2026-09-26/full-review-2cb1bc3/` (review evidence)
- No compiler, runtime, catalogue, example, or test source differs from the
  reviewed revision. Historical evidence stays tied to its original version:
  - Sept 24 preparation and T01–T27 records describe pre-`fbd2a56`/`961f921` states.
  - Post-upgrade reconciliation (`fbd2a56` vs `961f921`) and UP26/UP27
    qualification (`be95d009` → `2cb1bc35`) are the bridge to the current tree.
- This round's preparation outputs live under
  `docs/syntax-taste/preparation-2026-09-26/` and do not modify production sources.

## P01.2 Reconciliation: decisions vs delivered vs proposed

Authoritative inputs (in precedence order):

1. [decisions.md](../decisions.md) + incorporated specs (`technical-spec.md`,
   `coordination-spec.md`, `ai-io-spec.md`, `platform-testing-spec.md`) and the
   Sept 22 ledger/acceptance records. User choices here override review proposals.
2. [post-upgrade-dispositions-2026-09-24.md](../post-upgrade-dispositions-2026-09-24.md):
   11 Fix (all now delivered per UP27 audit), 9 Retain, 19 Defer.
3. [post-upgrade-completion-audit-2026-09-26.md](../post-upgrade-completion-audit-2026-09-26.md)
   + [fix evidence](../post-upgrade-fix-evidence-2026-09-26.md): delivered behavior at this baseline.
4. [full-language-review-2cb1bc3-2026-09-26.md](../full-language-review-2cb1bc3-2026-09-26.md):
   new proposals and acceptance conditions. A recommendation here does not select
   a mechanism or override (1)–(2).

Delivered at this baseline (do not re-decide; preserve as constraints):

- Shared handler-free actions + request-aware server mounts (U01/U02/U03).
- Maintained browser delivery with audited bundle, manifest pairing, ownership
  runtime (U04); cancelable admitted events (U05); observable handler faults (S02);
  visible HTML 503 (S01); grid focus/notice/pending state (S03).
- Structural browser audit (B01); Unicode regex advancement (B02).
- Bounded public generic-to-generic symbolic composition (P03).
- Immutable domain data, owner construction, explicit failures, shared
  client/server contracts, checked generics, audited browser capabilities,
  transaction commit uncertainty, assertion/fixture evidence, native operations.

Actual conflicts / reopenings (each needs a decision in this round, not silent reuse):

| # | New review ask | Existing rule it touches | Treatment |
|---|---|---|---|
| C1 | Allow `true`-before-`false` Boolean arms | User decision 2026-09-23: `false` first enforced; P15 Retain | User syntax choice (R08). Review taste alone does not override. |
| C2 | Allow meaningful final locals | C8/DI-23 narrow elision validity rule; P14 deferred demotion | User authoring-policy choice (R08). |
| C3 | Explicit `near` bindings / safe rename | LD38/LD39 + DI-08 name-based captures retained; P13 deferred | User syntax choice + technical packet (R08). |
| C4 | Finite error-set parameters | DI-03/LD14 deferred; P04 deferred | Technical decision; result-data first per review §5 (R06). Syntax only if evidence warrants. |
| C5 | Assertion setup region vs factories | P3.1/P4.1/DI-06 attached assertions + fixtures; no setup grammar (LD29 gate closed) | Technical decision with possible syntax (R07). |
| C6 | Iteration primitive vs self-tail lowering | LD21 deferred; relay = forwarding not tail-call; P05 deferred | Technical decision (R05). |
| C7 | Request/cancellation ownership changes | DI-15/LD30 ownership + drain retained; P06 deferred | Technical decision (R04). |
| C8 | Host adapter boundary vs catalogue additions | DI-19 controlled catalogue retained; P11 deferred | Technical decision (R01). |
| C9 | Richer input event fields / controls | First snapshot selected; U05 fixed; P08 deferred | Technical decision + library work (R02). |
| C10 | Captured HTML routes | Plain-route validation + GET-action JSON rules | Bounded technical decision (R03). |
| C11 | Bulk Map/Set construction | DI-20 deferred; C7 immutable copy-on-update retained; P18 deferred | Technical decision (R05). |
| C12 | SQL RETURNING / schema tooling / JSON/decimal | DI-14a deferred RETURNING; DI-14b/LD35 row validation; P19/P20 deferred | Technical + library decisions (R10). |
| C13 | Dynamic outbound destinations / worker carrier | DI-19 fixed-origin retained; O04/O05; P-carrier deferred | Technical + policy decision (R11). |
| C14 | S3 cancellation/deletion + deadline semantics | Catalogue contract vs `runtime/platform/s3.ts` behavior | **Defect**: contract discrepancy, not a taste question (R15). |
| C15 | Request-failure hook / redacted reporting | Server 500 conversion; main/late-owner/browser reporting exist | Technical decision (R12). |
| C16 | Editor completion/hover/refs/rename/format | LSP diagnostics + definition only; formatter machinery exists | Tooling decision (R09). |
| C17 | Linux/browser paired deployment, old-browser rollout | Debian 13+ amd64 path implemented; stale README corrected; not rerun in review | Qualification + docs (R13). |
| C18 | AI quality/latency/cost validation | Shape guarantees implemented; quality not established | Qualification scope (R14). |
| C19 | Invoice startup error example + webhook/docs limits | Authored error completions work; example stale | Docs/example defects (R16). |

No other user choice is reopened. Items marked Retain above keep their rule
unless the user explicitly revises it (C1–C3) or recorded reopening evidence is
produced (C4–C13).

## P01.3 Audience, priorities and scope

Carried forward (not re-asked):

- **Audience:** Can is made for AI coding agents. Human readability, familiarity,
  and comfort are not design goals. Human-hostile syntax is acceptable when it
  serves agents better. Every recommendation is evaluated against this audience.
- **Secondary goal:** token efficiency as total tokens per successful AI coding
  task (prompts + source + diagnostics + retries), per
  [evaluation-protocol](../preparation/evaluation-protocol.md) and
  [baseline](../../tests/baseline/README.md). Zero live agent attempts to date;
  no measured advantage is claimed. Source brevity never overrides correct
  contracts or repair reliability.
- **Standing constraints:** no compatibility obligation; lower to native
  JavaScript/Bun with only contract-preserving adapters; immutability, owner
  construction, explicit failures preserved.

Scope choices needing the user (asked in P06 alongside syntax; recorded here when answered):

- S1. Target application classes for this round: which of the review's SaaS
  shapes are in scope (controlled Bun backend / server-rendered CRUD; Can
  browser CRUD; rich-client widgets; integration-heavy webhooks/workers;
  object-upload workflows)?
- S2. Supported platforms: browsers, databases (SQLite/Postgres/MySQL?), object
  storage, Linux packaging — and what explicitly stays out.
- S3. Final ambition vs intermediate milestones: is the review's "normal SaaS
  choice" the target, or a bounded pilot + named follow-ups? What is the
  milestone order?
- S4. The review's SaaS recommendation is one input, not the whole roadmap.
  AI-native judgment features, editor tooling, and Linux operations keep their
  own standing unless the user reprioritizes them.

## P01.4 Coverage ledger

Full ledger: [finding-ledger.md](finding-ledger.md). Summary:

- Every review concern (§§1–6, production breadth, 4 concrete problems),
  every acceptance condition, and every relevant outstanding prior requirement
  (all P-deferred rows C4–C13, retained observations O01–O05 where they bound
  scope) has a ledger destination under R01–R16.
- Ledger kinds: defect (C14, R16-example, S3-deadline), missing capability,
  design choice, library work, tooling, operations, documentation.
- Inclusion in the ledger is not approval to implement.
