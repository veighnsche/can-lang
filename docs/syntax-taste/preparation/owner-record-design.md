# DI-02: owner-controlled record design

**Status after the upgrade (24 September 2026):** Owner records were implemented in T05, including owner-only construction, projection, update and destructuring. The original contract below remains decision evidence; consult current source and the reconciliation for qualifications such as codec rejection and representation-based equality. See the [current reconciliation](../post-upgrade-reconciliation-2026-09-24.md).

24 September 2026. Planned language contract, not implemented compiler behavior.
This resolves the remaining declaration and boundary details in the
[accepted DI-02 direction](accepted-technical-decisions.md#di-02--owner-controlled-public-records-for-validated-values).
The user has directed that no further design questions be sent to them. Three
fresh Jev requests informed this engineering selection; they did not determine
it by vote. [Requests, responses and disagreement analysis](jev-owner-record/findings.md)
are retained.

## Selected declaration

Use `owner record` followed by the existing record name, optional generic
parameters and indented field block. `provides` continues to control whether
the **type name** is public; it does not export construction rights.

```can
package values
    provides [email, email_text]
    uses []

owner record email
    str address

fn str email_text
    given
        email value
    asserts
        sample: email("valid@example.test") => ok "valid@example.test"
    ok value.address
```

This is proposed source, not an executed example. A real API also publishes a
validating factory, as in the [executed baseline](validated-value-probe.md).
The example's assertion is in the owning package and therefore may construct
its input directly. The ordinary `record` form remains transparent.

The grammar extension is:

```ebnf
owner_record = 'owner', 'record', name, parameters?, NL,
               (INDENT, field+, DEDENT)? ;
```

`owner` is contextual at a declaration prefix before `record`; it remains
available as a data identifier elsewhere. It is not admitted before `variant`,
`error`, functions, catalogue opaque types, or generated native question
records. `owner record email choice ...` is rejected. There is one spelling;
`owned email` and `record email owner` are not aliases. Generic syntax is
`owner record checked<item>`; generic arguments preserve ordinary specialized
nominal identity. This is a record access policy, not a new resource kind or
an ownership/borrowing system.

Prefixing the existing record production keeps the policy explicit while
reusing field, generic, nominal-leaf and constructor machinery. A separate
`owned` production expresses the same semantics but duplicates a declaration
category; a suffix is technically viable but supplies no demonstrated benefit.
`opaque` was not shortlisted because Can already uses opacity for maintained
resource types with different equality, lifetime and leaf-admission rules.
No agent-token or usability improvement has been measured for this spelling.

## Package and access rules

The owner is the **canonical declaring package instance** selected by DI-05a.
All source files in that package share the privilege. Another package in the
same project, a re-export, a file-local import alias, or a same-named package in
another dependency does not acquire it. Re-exporting a type never transfers
ownership. Exported type references and their explicit generic arguments obey
ordinary public-signature visibility checks.

All stored fields are hidden from nonowners. Publish observations through
ordinary exported functions or methods; no field-level `public` modifier is
added. A hidden field may have a private owner type, because exporting the
record no longer publishes its representation. An exported projection's return
type must itself satisfy normal public-signature rules. The compiler still
knows hidden fields for whole-program type checks and representation lowering.

| Operation | Owner source | Nonowner source |
| --- | --- | --- |
| Refer to a public type; store, pass, return, or whole-bind an admitted value | Admitted | Admitted |
| Construct the record directly | Admitted | Rejected |
| Read a stored field directly | Admitted | Rejected; use an exported projection |
| Use `with` on this record | Admitted under ordinary update rules | Rejected, even for an apparently unchanged replacement |
| Match the public nominal leaf without fields | Admitted | Admitted |
| Destructure a constructor pattern | Admitted | Rejected, including all-wildcard/zero-field constructor patterns |
| Compare eligible values for ordinary Can equality | Admitted | Admitted |
| Derive a generic document codec or wire schema reaching this type | Rejected | Rejected |

`with` on an unrelated transparent outer record may replace an owner-record
field with another already admitted value. That does not mint or change the
protected inner record. Arrays, ordinary record fields, callable arguments and
results likewise transport admitted values without acquiring creation rights.
Public variants may include owner-record leaves under extensional DI-07 rules.

Authorization is lexical, not dynamic. A generic function is checked under the
canonical package of its declaration, including each concrete specialization.
Instantiating foreign generic code inside the owner cannot authorize its field
access, construction or update. Conversely, calling a public owner-authored
generic factory from another package does not remove that factory's legitimate
privilege. Inlining, generated helpers, aliases and fixture lowering must carry
the original checked owner rather than the caller's location.

## Matching and equality

A nonowner can test an exported nominal leaf, use `_`, or use `bind value` to
capture an entire admitted value. A leaf test must lower to a nominal identity
test without reading or binding hidden fields. Constructor patterns such as
`values::email(bind text)` and `values::email(_)` are denied outside the owner;
the public leaf name alone is sufficient for variant exhaustiveness. Owners
may use ordinary field patterns and the selected explicit `bind` syntax.

Keep existing structural immutable equality eligibility and behavior. The
record's canonical specialized nominal identity and eligible field values
participate; separate factory calls yielding equal representation compare
equal. A reachable resource, callable or other equality-ineligible member
continues to make its containing record ineligible. There is no implicit custom
equality hook or reference-identity replacement. Diagnostics can identify the
public owner type without exposing private member names.

Equality can distinguish hidden representations. This boundary protects
construction invariants; it promises neither information-theoretic secrecy nor
stable equality across owner representation changes. An API that needs its own
domain equivalence can publish a named comparison function. Keeping equality
does not grant decoding, field access or structural casts.

## Codecs, schemas and external values

Reject generic document encoding, decoding and **wire** schema derivation if
the complete concrete type graph reaches an owner record. Apply this uniformly
inside and outside its owner, including arrays, nested records, recursive
graphs and variants whose current runtime value happens to use an unprotected
leaf. Schema admission is a type-level property, not a branch-sensitive test.
This restriction also applies to any AI structured-output, SQL projection or
other boundary that uses the same generic schema to fabricate nominal data.
Compiler type metadata and nominal diagnostics are not wire-schema derivation.

The owner publishes explicit conversion functions:

1. Decode external data into a transparent wire record.
2. Validate it in the owner factory and construct the owner record there.
3. For output, project an admitted owner value into a transparent wire record
   and encode that record.

There is no automatic factory discovery, derived validator, custom-codec
registry, or decoder hook in this initial design. An explicit owner codec may
compose these steps as an ordinary function. Structural agreement with the
representation never grants an external value its nominal identity. Any
maintained native adapter returning an owner type must be an explicit trusted
owner construction path; arbitrary JS data, casts or generic schema projectors
cannot manufacture it. The initial language adds no such foreign constructor.

Jev preferred owner-only automatic derivation in all three consultations, with
varying confidence. That rule is coherent because owners are already trusted
to construct values. The engineering selection is stricter: generic decode
would introduce another implicit creation route that does not call the factory,
and owner-dependent schema admission complicates generic specialization. One
uniform wire boundary makes external conversion explicit and keeps the current
schema graph independent of caller privilege. It does not prove factory
predicates, nor does it prevent an owner from deliberately constructing invalid
data. This is a selected simplicity/explicitness tradeoff, not a measured
correctness advantage over the other viable policy.

## Assertions, fixtures and trust

Check assertion input, expectation and supplied-completion expressions under
their authoring package with the same constructor, update, field and schema
rules as ordinary code. Fixtures get no special right to mint protected values.
The harness must not deserialize a structurally matching fixture payload
directly into an owner record. A foreign fixture may pass an existing admitted
owner value obtained through a checked owner function; mere payload shape or
matching type text does not establish that provenance.

Owner-authored assertions may deliberately exercise raw constructors and
invalid representations to test owner code. They are trusted owner source,
not proof that every factory validates correctly. Supplied-completion evidence
continues to be labeled as supplied; it cannot establish what the real factory
would return. Positive factory tests must execute the real owner body, including
its invalid-input rejection path. Renaming a fixture label or moving fixture
lowering must not alter access rights.

A valid `tenant_id` establishes only the predicate its owner checks. Invoice
authorization still checks actor, membership, tenant, resource and revision at
the protected operation. Owner records must not be presented as proof of those
runtime facts.

## Native lowering and diagnostics

Retain the existing frozen nominal record representation and native JS/Bun
operations behind necessary identity/immutability adapters. Do not add a proxy,
wrapper allocation, runtime validator call on every projection, or new resource
lease merely because the declaration is owner-controlled. Owner construction
and `with` use ordinary checked record creation/update. Projection functions
lower to normal calls and property reads; eligible equality retains the current
native deep-comparison contract.

Record metadata needs distinct public-type visibility, defining canonical
package and representation-access policy. Enforce these at constructor lookup,
field projection, update, constructor-pattern checking, generic specialization,
schema admission and harness expression admission. Checking only constructor
lookup is insufficient. This is a checked Can boundary, not a sandbox against
someone modifying generated TypeScript or the compiler/runtime.

Diagnostics must name the attempted operation, public type and defining owner,
and suggest the relevant category of owner API without inventing a factory
name. For a codec rejection, identify the reachable protected type and suggest
wire-data conversion; report a useful public type path without dumping hidden
representation details. No backwards-compatible constructor or decoder escape
is retained.

## Acceptance contract for implementation

The [baseline probe](validated-value-probe.md) is executed evidence for the gap.
The following are planned production checks; none is claimed to pass under an
implemented owner-record feature yet.

| Case | Required outcome |
| --- | --- |
| Existing email/quantity owner factories, valid and invalid inputs | Valid calls return admitted values; real invalid calls emit the owner's named error. |
| Importer direct constructor, `with`, field read and nested constructor pattern | Compile errors in ordinary code and assertion/supplied-completion expressions. |
| Owner constructor, field read, controlled update and owner assertion | Admitted under ordinary typing; factory behavior separately tested. |
| Public type with a private internal representation type | Public signatures may expose the owner type; foreign access to the representation/private type remains denied. |
| Exported projector with a private return type | Rejected by ordinary signature visibility. |
| Leaf-only match, `bind`, `_`, public variant narrowing and forwarding | Admitted without exposing or rebuilding the value. |
| External wire decode then real factory, followed by explicit wire encoding | Successful roundtrip for valid data; predicate error for invalid data. |
| Direct or nested generic encode/decode/schema inside owner and importer | Rejected, including `option<owner_type>` when the test value is its empty leaf. |
| Same-name packages, alias edits, checkout relocation and distinct dependency instances | Privilege tracks the canonical package instance, not spelling or path accidents. |
| Foreign generic constructor/update/access instantiated by owner | Rejected; declaration-site authority remains foreign. |
| Owner generic factory instantiated by importer | Admitted, with owner privilege and ordinary concrete type checks. |
| Foreign fixture forwarding a real owner-produced value vs raw wire payload | Forwarding admitted; structural fabrication rejected; evidence distinguishes real and supplied paths. |
| Equal separate factory products, unequal products and resource-bearing owner records | Existing eligible deep equality for the first two; ineligible equality rejected for the last. |
| Transparent outer-record update with an admitted protected child | Admitted; protected child construction/update remains denied. |
| Generated TypeScript inspection | Ordinary immutable nominal objects and native operations; no unnecessary owner wrapper/proxy/lease. |

Production work must also test diagnostics, recursive schema rejection,
specialization cache ownership and harness transport. Comparative agent
creation/refactor/repair trials remain a separate evaluation gate; these checks
establish contracts rather than a token-efficiency claim.
