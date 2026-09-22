# Prompt: settle the Schema trust model (before any implementation plan)

Paste this into a fresh designer with `docs/schema-workstream.md`
attached. No code in this phase. The deliverable is rulings, not slices.

---

`docs/schema-workstream.md` scopes the minimum asset-approval surface:
`ApprovedAsset`, `AssetPolicy`, a registry. Your job is the trust model
behind those three names. For each question below, return a ruling, a
one-paragraph rationale, and the failure rows it implies (what input,
what rejection, what error). A question answered "we'll decide later"
is a question failed — name it `unresolved:` with the exact missing
source instead.

## The questions

1. **What is an approval?** Content hash, host allowlist, or both?
   Rule each combination (hash-only, host-only, both, neither) as
   accepted or rejected, with the attack each rejected option allows.
2. **Who approves?** Name exactly which principal may add registry
   entries, and the mechanism that stops self-approval. If approval
   lives in program declarations, say who writes them and who checks
   the writer.
3. **Where does policy live?** Per-program, per-module, or global?
   Rule out caller-chosen policy: the policy governing a use site must
   not come from the caller being checked.
4. **What binds an approval to its use?** Asset, role (stylesheet vs
   script), consuming site — which bindings exist, enforced how? The
   Bytes B2 owner-local authority model is the template to adopt or
   argue against; do not invent a weaker one silently.
5. **Construction or retrieval?** Approval at build time and
   verification at fetch time are different moments. Name which moment
   this design owns, what guarantees cross the gap, and what is
   explicitly left to the future fetching layer.
6. **Transport.** Is `http` approvable? Either forbid it or write the
   justification and the MITM row.
7. **Lifecycle.** How does approval end — removal, expiry, rotation?
   A registry with only adds blesses compromised assets forever.
8. **Transitivity.** An approved script loading further scripts, a
   stylesheet with `@import`: rule the tree (covered, cut off, or
   forbidden primitives). Silence is not a ruling.
9. **Role binding.** What stops approved CSS spent as script? Name the
   mechanism, not the intention.
10. **Versioning.** Exact pins or ranges? Rule ranges out or defend
    them against the known-vulnerable-library case.
11. **Error contents.** Rejections carry the offending asset, never
    registry contents. Confirm the payload rule per new error kind.
12. **Fixture authority.** Test seals prove the check runs, never that
    an asset is blessed. State how the test harness respects this.

## Constraints (not decisions — already settled)

- Catalogue contract: approved external assets, never raw strings.
- Inline script/style stay excluded; the registry approves, never fetches.
- Fault contracts: declared, value-carrying errors; NUL policy holds
  wherever text crosses into bytes.
- `a13` layering: this is the asset minimum of Schema, not the general
  HTTP/SQL/UI layer — mark anything beyond assets as out of scope,
  not as implied.

## Deliverable

Per-question ruling + rationale + failure rows, then a deferred list
(anything pushed to fetching, to general Schema, or to later slices).
Ready means: an implementer could slice it without making a trust
decision. After this comes adversarial security review, then — only
then — the implementation plan.
