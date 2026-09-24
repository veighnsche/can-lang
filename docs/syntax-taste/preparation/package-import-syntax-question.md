# DI-05a syntax question: addressing same-named dependency packages

24 September 2026 · historical comparison packet. The user subsequently
selected dependency-qualified source imports; see the
[confirmed syntax scope](confirmed-syntax-choices.md#dependency-qualified-source-imports--di-05a).

Current `uses [model as x]` resolves `model` through a global short-name map;
two independently authored dependencies each exposing `model` are rejected
before aliases matter ([evidence](packages-assertions-evidence.md#package-and-error-identity-already-in-use)).
The technical requirement is one canonical resolved identity per package
instance, dependency-relative lookup, direct-import visibility, reserved
catalogue names, private-package checks, reproducible locks and native module
emission. Neither option below asks either dependency to rename its source.
Three fresh [Jev judgments](jev-core-contracts/findings.md#di-05a-both-lookup-candidates-can-pass-author-renaming-cannot-count)
split between the two options. No agent comparison has measured their
context/repair cost. The examples were illustrative when compared; option A
is now selected.

Suppose the root manifest already names direct dependencies `billing` and
`crm`; both contain a package declared `model`.

**A. Qualify in source.** A `uses` entry names the direct dependency key and
package at the point of use. The final alias remains file-local:

```text
uses [billing::model as bill_model, crm::model as crm_model]
```

The root manifest needs no additional import-handle table. Inside either
dependency, its own `uses` resolves against *its* dependency keys. A missing
key/package or transitive-only key is a source diagnostic. A root-local
package can still use the current local form, and catalogue names remain
reserved. This puts complete import identity in source at the cost of longer
entries.

**B. Bind handles in the manifest.** The project manifest maps unique handles
to direct dependency keys/packages; source imports only the handle and chooses
a file-local alias:

```json
"import_handles": {
  "bill_model": {"dependency": "billing", "package": "model"},
  "crm_model": {"dependency": "crm", "package": "model"}
}
```

```text
uses [bill_model as bill, crm_model as customer]
```

The manifest must reject missing/ambiguous targets and lock the resolved
mapping. Each dependency's own manifest interprets its handles, not the root's.
This can shorten repeated imports but makes an agent inspect another file to
learn source identity. Renaming a handle differs from changing the target;
diagnostics must identify both.

Either choice must pass the same two-unchanged-library composition test,
alias edits, transitive-import rejection, private-package protection and
relocated-checkout build. The source punctuation and exact JSON field names
shown here are part of the syntax question, not established compiler behavior.
An alternative spelling is welcome if it preserves the complete contract.
