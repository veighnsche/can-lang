# Schema asset approval — trust-model rulings

**Adopt content-and-origin approval, issued by an independent deployment authority, under one per-program policy, bound to an exact asset revision, role, and construction site. Approval is non-transitive and valid against an identified registry snapshot—not proof of successful retrieval, current browser-side authorization, or harmless behavior.**

These are **proposed rulings for adversarial security review**, not claims that the mechanisms already exist. The attached workstream leaves their design open and requires them to be settled before implementation planning.  Supplemental repository evidence below is pinned to `46fa5169183cb56aade2a4ddedb446c2740fe491`.

## Common definitions

| Term                  | Meaning in these rulings                                                                                                                                                                                                                   |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Asset request**     | Untrusted identifying data: asset identifier, exact revision, absolute URL, claimed integrity digest, and requested role. It contains no authority.                                                                                        |
| **ApprovedAsset**     | An opaque, immutable approval witness containing the approved request and its policy, registry, site, and validity bindings. No ordinary constructor, record reconstruction, string conversion, or production `seal` can manufacture it.   |
| **AssetPolicy**       | An opaque handle to the authoritative policy selected for this program’s construction session. Passing the handle does not let the caller select or change the governing policy.                                                           |
| **Registry**          | An externally accepted, signed, immutable snapshot of approval entries and revocations. A program declaration or repository file is only a proposal unless the external acceptance mechanism authenticates it.                             |
| **Construction site** | The exact resolved source call site invoking an asset builder: program identity, canonical module identity, containing function and revision, and certified call node. It is **not** a hostname, DOM position, or eventual rendering page. |

Every failure-row error below carries the offending asset. Section 11 specifies the payloads and distinguishes language errors from administrative failures and compiler diagnostics.

---

## 1. What is an approval?

### Ruling

**Require both an exact content digest and an explicitly permitted HTTPS origin, plus an exact URL entry.** Neither an origin nor a digest grants authority independently.

The first profile admits exactly one SHA-384 digest per asset revision, expressed in canonical SRI form. Multiple acceptable digests, wildcard origins, host suffix matching, path-prefix approval, and “any resource on this CDN” are rejected.

The signed entry binds the full URL, including path and query. URL handling must reject ambiguous or noncanonical input rather than silently repair it. Credentials, fragments, control characters, NUL, and relative URLs are excluded.

### Rationale

The digest identifies bytes; the origin restriction controls which endpoint the application is authorized to contact. A digest alone does **not** permit an attacker to substitute different bytes while preserving that digest, but it does leave destination selection unconstrained. Conversely, an approved host can serve changed content. These are different protections, which the workstream explicitly asks the model to distinguish.  SRI supplies the content-integrity mechanism; it does not itself establish this registry’s approval decision. ([W3C][1])

### Failure rows

| Input                                                                          | Ruling / rejected attack                                                                                                                 | Error                       |
| ------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------- | --------------------------- |
| Approved digest; arbitrary or unlisted origin                                  | Reject. Prevents unauthorized destination selection, including contacting an attacker-controlled endpoint with otherwise approved bytes. | `schema.asset_not_approved` |
| Listed origin; no digest                                                       | Reject. Prevents changed or compromised CDN content inheriting host approval.                                                            | `schema.asset_not_approved` |
| Neither digest nor origin approval                                             | Reject. Prevents arbitrary external inclusion.                                                                                           | `schema.asset_not_approved` |
| Both, with an exact active registry entry and all remaining bindings satisfied | Accept. This is necessary but not sufficient without the remaining checks.                                                               | None                        |
| Correct URL but different digest; correct digest but different path/query      | Reject. No approximate lookup or substitution.                                                                                           | `schema.asset_not_approved` |

---

## 2. Who approves?

### Ruling

Only **the program’s externally enrolled Asset Approver principal**, identified by its authenticated principal ID and pinned signing key, may authorize additions or expansions of registry authority.

The deployment owner enrolls that principal through a trust root outside the candidate program and outside the coding agent’s writable workspace. The program author, dependency maintainer, module owner, build script, and coding agent receive **no approval authority by virtue of those roles**.

For an addition, the approval service must authenticate the proposer and approver and reject approval by the same enrolled principal. Those identities come from the service’s authentication records, not editable declaration fields or Git author strings.

An approval signature covers the entire entry and its bindings. The signing key is unavailable to candidate build jobs. A signer that automatically signs every syntactically valid proposal is not an approval mechanism.

### Rationale

The relevant boundary is acceptance authority, not who can edit a file. Source declarations may request an approval, but cannot authenticate themselves. Independent credentials prevent a coding agent from turning an arbitrary URL into an approved asset merely by modifying policy beside its code. This directly addresses the workstream’s registry-poisoning requirement.  The deployment owner and enrolled approver remain trusted principals: compromise or deliberate misuse of their authority is outside the guarantee.

### Failure rows

| Input                                             | Rejection                                                     | Error                               |
| ------------------------------------------------- | ------------------------------------------------------------- | ----------------------------------- |
| Program adds an unsigned approval declaration     | Proposal has no external authorization.                       | `AssetRegistryChangeRejected`       |
| Proposer approves its own request                 | Authenticated proposer and approver identities coincide.      | `AssetRegistryChangeRejected`       |
| Valid entry edited after approval                 | Signature does not authenticate the changed entry.            | `AssetRegistryChangeRejected`       |
| Candidate supplies its own replacement trust root | Candidate cannot choose the authority checking it.            | `SchemaAuthorityInvalid` diagnostic |
| No enrolled approver or pinned trust root         | No production approvals are available; no permissive default. | `SchemaAuthorityInvalid` diagnostic |

The actual principal ID and public key are mandatory deployment inputs. None is supplied or asserted to exist by this design.

---

## 3. Where does policy live?

### Ruling

**One authoritative asset policy per program deployment identity.**

The protected acceptance workflow selects the exact policy revision and registry snapshot. Both are immutable during a construction session. Program source may reference them but cannot nominate a substitute authority.

Per-module restrictions are entries within that program policy, not independently selected module policies. No ambient global-policy fallback exists.

The public `AssetPolicy` argument remains explicit, as required by the consumer contract, but the builder verifies that it is the handle selected for its certified site and session. It is **evidence of the governing policy, not a policy-selection parameter**.

### Rationale

A genuine but more permissive policy is still the wrong policy when it does not govern the use being checked. Checking only whether the supplied policy was signed would permit policy substitution. Binding the selected policy to the program, environment, and site prevents that substitution while retaining the requested approved-asset/policy interface. The workstream explicitly identifies caller-controlled policy as an open authority question. 

### Failure rows

| Input                                                           | Rejection                                     | Error                               |
| --------------------------------------------------------------- | --------------------------------------------- | ----------------------------------- |
| Production site receives a legitimate development-policy handle | Wrong deployment identity.                    | Appropriate HTML builder rejection  |
| Module supplies its own permissive policy                       | Not the externally selected governing policy. | `schema.asset_not_approved`         |
| Asset witness and policy handle refer to different snapshots    | No mixing independently valid contexts.       | Appropriate HTML builder rejection  |
| Policy or registry absent; caller requests fallback             | Fail closed.                                  | `SchemaAuthorityInvalid` diagnostic |

“Appropriate HTML builder rejection” means `html.asset_stylesheet_rejected` or `html.asset_script_rejected`, according to the invoked operation.

---

## 4. What binds an approval to its use?

### Ruling

The approval binds:

**program/deployment identity + policy identity + registry snapshot + asset identity/revision + exact URL/digest + role + construction site + validity interval.**

The compiler certifies the **resolved call node**, never a caller-supplied site name. Source locations alone are insufficient. Copying a declaration into another module, reusing a basename, changing the target function, or changing authority-relevant code invalidates the certificate. Certificates are reconstructed for each checking session.

Adopt B2’s exact-node certification, canonical ownership checks, and fresh validation before test or linkage execution. B2 already uses those boundaries rather than trusting matching names or serialized certificates.

Because Schema owns `ApprovedAsset` while HTML owns `Html__Safe`, this requires an **explicit two-owner bridge**, not an assumed same-owner grant:

* Schema authorizes consumption of its opaque value by the exact pinned builder.
* HTML authorizes that builder to construct `Html__Safe`.
* The external program policy authorizes the application construction site.

The bridge produces the fixed, validated asset element. It exposes **no general URL getter, unseal, or raw-string serialization capability**.

### Rationale

B2’s owner-local rule stops unauthorized use of a capability; it does not establish that a module owner is an independent asset approver. Schema therefore needs both structural ownership certification and external asset authorization. The bridge is limited to these asset consumers, not a general declassification facility. This preserves the workstream’s approved-only interface and its prohibition on recovering raw URLs merely to work around a type boundary.  

**Boundary:** after authorized construction, `Html__Safe` remains transferable. This model does not restrict its eventual DOM or page placement. Such a guarantee would require context-carrying output rather than an unrestricted `Html__Safe`.

### Failure rows

| Input                                                       | Rejection                                                   | Error                               |
| ----------------------------------------------------------- | ----------------------------------------------------------- | ----------------------------------- |
| Module B directly spends a witness granted to module A      | Construction-site mismatch.                                 | Appropriate HTML builder rejection  |
| Copied site label or matching basename in another directory | Resolved identity does not match the grant.                 | `SchemaAuthorityInvalid` diagnostic |
| Missing Schema-side or HTML-side bridge authorization       | No authorized opaque-to-HTML boundary.                      | `SchemaAuthorityInvalid` diagnostic |
| Changed consumer code with an old certificate               | Fresh certification fails; old annotation cannot be reused. | `SchemaAuthorityInvalid` diagnostic |

---

## 5. Construction or retrieval?

### Ruling

**This workstream owns construction-time authorization against an admitted snapshot. It does not fetch or verify a remote response.**

`ApprovedAsset` means that the identified request passed authorization under policy snapshot P, registry snapshot R, and trusted evaluation time T. It does not mean that the remote server has already supplied matching bytes.

The HTML builders must carry the approved URL and exact integrity metadata into their fixed output, without caller overrides. Cross-origin integrity-enabled output uses the required CORS configuration; integrity failure must never trigger an unapproved fallback. SRI checks eligible fetched responses and rejects mismatching content rather than executing or applying it. ([W3C][1])

The future fetching layer must enforce the already-decided retrieval conditions: digest equality, permitted transport and destination, role-compatible response handling, current lifecycle authorization, and rejection rather than fallback. **Redirects are forbidden in this first trust profile.**

### Rationale

Comparing a requested digest with a registry digest authenticates the *expectation*, not the server’s bytes. The construction result must preserve that expectation for a later verifier without claiming that verification has occurred. This is exactly the workstream’s construction/retrieval distinction and no-fetch boundary.   Ordinary emitted HTML does not prove redirect behavior, current registry freshness, or successful network enforcement; those remain explicit integration obligations.

### Failure rows

| Input / event                                                         | Rejection                                                    | Error or owner                                             |
| --------------------------------------------------------------------- | ------------------------------------------------------------ | ---------------------------------------------------------- |
| Construction request claims a digest absent from its authorized entry | Requested expectation is not approved.                       | `schema.asset_not_approved`                                |
| Output construction attempts to omit or replace integrity metadata    | Builder is not an authorized sink implementation.            | `SchemaAuthorityInvalid` diagnostic                        |
| Server changes bytes after construction                               | Do not consume the response.                                 | Retrieval integrity failure; **not** a Schema call outcome |
| Fetch encounters any redirect                                         | Reject; do not follow or approve the destination implicitly. | Future fetching layer                                      |
| Network failure followed by an unsigned alternate asset               | Reject fallback.                                             | Future fetching layer                                      |

The retrieval failure classes are obligations, not new `.can` error declarations in this asset-only phase.

---

## 6. Transport

### Ruling

**Only absolute HTTPS URLs are approvable.** There is no HTTP, localhost, development, or “hash makes HTTP safe” exception in the production profile.

Protocol-relative URLs, `data:`, `blob:`, `file:`, inline content, and HTTPS-to-HTTP downgrade paths are excluded.

The containing document and its integrity metadata must also be delivered through the deployment’s authenticated secure delivery path. Schema records that deployment obligation; it cannot establish it while constructing a fragment.

### Rationale

Requiring secure transport avoids treating content integrity as a replacement for endpoint authentication and confidentiality. More importantly, an attacker able to modify the containing document can remove or replace its integrity metadata. The SRI specification explicitly identifies this limitation for insecure document delivery. ([W3C][2]) Transport approval is therefore necessary on both sides of the construction/retrieval boundary, not just a scheme check on the asset string.

### Failure rows

| Input / event                                                | Rejection                                                                                            | Error or owner                |
| ------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------- | ----------------------------- |
| HTTP asset with the correct digest                           | Ineligible transport.                                                                                | `schema.asset_not_approved`   |
| Protocol-relative or non-network asset URL                   | Unsupported asset locator.                                                                           | `schema.asset_not_approved`   |
| HTTPS request receives a downgrade redirect                  | Reject before following it.                                                                          | Future fetching layer         |
| MITM alters an HTTP-delivered page’s URL and digest together | Deployment violates the secure-document precondition; Schema has no browser-side guarantee to claim. | Deployment acceptance failure |

---

## 7. Lifecycle

### Ruling

Approval supports **explicit revocation/removal, mandatory finite expiry, and explicit rotation**.

An entry is usable only within its validity interval, with the upper bound exclusive. Registry snapshots carry monotonically increasing sequence numbers. The trusted acceptance workflow supplies the current accepted head and time; the candidate cannot provide a clock or lower the accepted sequence.

Removal becomes a retained revocation record, not deletion of audit history. Revoked asset revisions cannot be resurrected by replaying an old snapshot. Rotation creates a new exact revision and digest and requires explicit caller repinning.

Construction evidence is snapshot-scoped. Before a newly built artifact is accepted for deployment, the acceptance workflow rechecks registry freshness and expiry. Replaying an old construction result does not renew approval.

### Rationale

A signed registry can still be stale. Signature validation without freshness permits rollback to a snapshot preceding revocation. An external head and trusted time prevent that rollback without adding a clock or network fetch to hermetic language evaluation. The workstream expressly requires deletion/rotation rather than permanent add-only approval.  Already-served documents and already-executing scripts cannot be recalled by changing this registry; deployment withdrawal and retrieval-time checks own that remaining exposure.

### Failure rows

| Input                                                             | Rejection                                                                | Error or owner                                                   |
| ----------------------------------------------------------------- | ------------------------------------------------------------------------ | ---------------------------------------------------------------- |
| Previously valid entry is now revoked                             | No new approval or construction under the current snapshot.              | `schema.asset_not_approved` / appropriate HTML builder rejection |
| Trusted time equals or exceeds expiry                             | Entry is expired.                                                        | Same                                                             |
| Old, correctly signed snapshot is replayed                        | Snapshot is not the current accepted authority.                          | `SchemaAuthorityInvalid` diagnostic                              |
| New digest is substituted under an existing asset revision        | Immutable revision identity violated.                                    | `AssetRegistryChangeRejected`                                    |
| Registry changes or approval expires before deployment acceptance | Previous construction evidence is no longer acceptable for this release. | Deployment acceptance failure                                    |

No available current authority means rejection, not “use the last known approval.”

---

## 8. Transitivity

### Ruling

**Cut approval off at the directly referenced asset. Approval does not propagate through a dependency tree.**

Approving script A does not approve scripts loaded by A. Approving stylesheet A does not approve its `@import` targets, fonts, images, or other callback destinations.

A descendant can receive a separate approval for its own direct construction site, but cannot inherit its parent’s witness. This minimum offers no “approve tree,” delegated approval, or automatically trusted dependency option.

**This is a cutoff in the approval claim, not a promise that the browser blocks every descendant request.**

### Rationale

The registry does not fetch, parse, or execute assets, so it cannot discover or mediate every dynamically generated request. Pretending that a declaration such as “no imports” proves that property would introduce another self-attested security boundary. The attachment distinguishes approval from sandboxing and explicitly warns about transitive script and CSS behavior.  A future closed-tree execution profile would require independent request mediation; CSP’s `strict-dynamic`, for example, deliberately permits trust propagation and cannot be presented as this model’s non-transitive enforcement. ([W3C][3])

### Failure rows

| Input / event                                                    | Ruling                                                                                                             | Error or owner                |
| ---------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ | ----------------------------- |
| Request approval for B using only A’s approved witness           | Reject inherited authority.                                                                                        | `schema.asset_not_approved`   |
| Registry proposal requests “A and anything A loads”              | Reject unsupported authority expansion.                                                                            | `AssetRegistryChangeRejected` |
| Approved A dynamically loads unapproved B outside these builders | **Not checked by Schema. B is not approved.** Do not report a fabricated Schema rejection or whole-tree guarantee. | Execution/fetching boundary   |
| Caller asks Schema to certify that A has no imports or callbacks | Reject unsupported claim.                                                                                          | `AssetRegistryChangeRejected` |

An approved root containing a loader can therefore be accepted as that exact root. The approver is accepting its behavior—not obtaining a theorem about its descendants.

---

## 9. Role binding

### Ruling

Each approval has **exactly one role**. The minimum roles are external stylesheet and external classic script. Module scripts, workers, preload uses, and other execution modes are not aliases for those roles.

The role is authenticated inside `ApprovedAsset`. Each builder supplies its own required role; the caller cannot pass a role override. Matching is performed before any HTML is constructed.

The same bytes may be approved separately for two roles, but neither approval implies the other. File extensions, hostnames, MIME claims, and digest equality cannot substitute for a role grant.

### Rationale

Content identity is not interpretation authority. The same bytes may be interpreted differently in different contexts, so the consuming operation—not a caller-provided flag—must determine the required role. This makes the workstream’s CSS-as-script rejection a checked condition rather than a naming convention.  Actual response MIME and decoding validation belong to retrieval and do not replace this construction-time role check.

### Failure rows

| Input                                                       | Rejection                                           | Error                               |
| ----------------------------------------------------------- | --------------------------------------------------- | ----------------------------------- |
| Stylesheet approval passed to script builder                | Wrong approved role.                                | `html.asset_script_rejected`        |
| Script approval passed to stylesheet builder                | Wrong approved role.                                | `html.asset_stylesheet_rejected`    |
| Caller alters a role field or reconstructs the witness      | Opaque approval cannot be reconstructed or mutated. | `SchemaAuthorityInvalid` diagnostic |
| Classic-script approval used for module or worker execution | Unsupported role/mode.                              | `schema.asset_not_approved`         |

---

## 10. Versioning

### Ruling

**Exact pins only.**

A use pins an immutable asset identifier and revision; that revision fixes one URL, digest, and role. The construction session also identifies exact policy and registry snapshots. There is no version-range resolution, automatic upgrade, `latest`, wildcard revision, or fallback to another approved version.

Changing URL, bytes, role, or authority-bearing bindings requires a newly accepted revision or grant. Existing callers remain pinned and can fail after revocation; they are never silently retargeted.

A policy’s current revocation state overrides historical approval of an exact pin.

### Rationale

Ranges let selection move beyond the artifact actually reviewed, while immutable pins identify the reviewed object. However, an exact pin can still identify a known-vulnerable artifact; exactness does not establish safety. Revocation must therefore reject that pin instead of either blessing it forever or replacing it without the caller’s explicit change. This is consistent with the attachment’s exact-pin requirement and its separation of approval from harmlessness. 

### Failure rows

| Input                                                      | Rejection                                                                  | Error                               |
| ---------------------------------------------------------- | -------------------------------------------------------------------------- | ----------------------------------- |
| Version range, wildcard, or floating selector              | Not an exact request.                                                      | `schema.asset_not_approved`         |
| Exact revision now revoked for a vulnerability             | Historical approval is insufficient.                                       | `schema.asset_not_approved`         |
| CDN changes bytes without an asset revision change         | Construction retains the old digest; changed response must fail retrieval. | Retrieval integrity failure         |
| Two conflicting entries define the same immutable revision | Reject the snapshot; no first-match-wins registry.                         | `SchemaAuthorityInvalid` diagnostic |

---

## 11. Error contents

### Ruling

**Errors return the offending submitted asset, never registry-derived alternatives or expectations.**

Registry-dependent lookup failures are deliberately indistinguishable: unknown asset, wrong registered hash, unavailable role/site grant, wrong policy, revocation, and expiry all produce the same lookup error shape.

The new result kinds are:

| Kind                             | Channel and exact payload rule                                                                                                                                                      |
| -------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `schema.asset_not_approved`      | Language error from approval lookup. `asset` contains the original asset request, unchanged. No registered digest, policy contents, or alternate asset.                             |
| `html.asset_stylesheet_rejected` | Language error from stylesheet construction. `asset` contains the original supplied `ApprovedAsset` value, unchanged.                                                               |
| `html.asset_script_rejected`     | Language error from script construction. Same payload rule.                                                                                                                         |
| `AssetRegistryChangeRejected`    | Administrative approval-workflow result, **not** a callable `.can` registry-write API. Contains the submitted offending asset descriptor, not the existing entry or whole snapshot. |

`SchemaAuthorityInvalid` is a **compiler/acceptance diagnostic**, not an `emits` outcome. It covers missing trust roots, illegal construction of authority values, invalid sink certification, and malformed authority snapshots. It points to offending source/input without dumping trusted state.

### Rationale

Reporting “wrong hash; expected H” would reveal registry contents. Even returning different errors for “unknown URL” and “known URL, wrong hash” creates an avoidable membership oracle. The public result therefore explains which operation rejected the supplied asset without disclosing the registry’s comparison operand. The workstream requires offending-value payloads and specifically prohibits registry enumeration.   Successful authorized lookups necessarily reveal that the particular request is approved; this is not a claim of complete registry secrecy.

### Failure rows

| Input                                            | Required observable result                                                                                                         |
| ------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------- |
| Unknown URL versus known URL with a wrong digest | Same error kind and shape, apart from their different submitted asset payloads.                                                    |
| Registry-dependent denial                        | No expected digest, approved-host list, suggestions, matching entries, or alternate-policy contents in CLI, LSP, catalog, or logs. |
| Asset request containing NUL                     | Reject without stripping or replacing the offending value. No successful HTML/text-to-bytes path.                                  |
| Builder rejects a witness                        | Preserve that witness as the error value; diagnostic rendering must not expand hidden authority internals.                         |
| Exhaustiveness handles approval failures         | Handle the declared operation-specific outcome; no silent empty fragment or exception-based recovery.                              |

These names are proposed error contracts, not assertions that codes or declarations have already been allocated.

---

## 12. Fixture authority

### Ruling

The test harness has a **separate fixture trust root and registry**, isolated from production authority.

Test seals may construct fixture values or expected outcomes. They cannot create registry entries, authorize production sites, or suppress the approval check. Approval lookup and sink validation execute against the harness-installed fixture context; a `given` row cannot replace them with an authoritative success.

Positive fixtures must install an explicit fixture approval. A success-shaped value alone is not evidence of approval. Fixture certificates cannot be serialized into production authority or accepted under production keys.

The certifier runs before test/linkage execution, and both evaluation and emission require the appropriate validated authority. This adopts B2’s ordering specifically to prevent failed or missing certification being treated as trustworthy scripted evidence.

### Rationale

Tests need to represent hostile and mismatched values, including values ordinary source cannot construct. That representational permission must not become authorization permission. Separating fixture authority from production authority allows committed positive and negative rows while preserving the workstream’s explicit statement that test seals prove the check runs, not that an asset is blessed. 

### Failure rows

| Fixture input                                                                | Required result                                                   |
| ---------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| Sealed success-shaped asset, absent from fixture registry                    | Lookup or builder rejection; no approval inferred from the seal.  |
| Positive fixture approval removed between runs                               | Next run rejects; no reused certificate.                          |
| Fixture witness or fixture-signed registry supplied to production acceptance | `SchemaAuthorityInvalid`; fixture root is not a production root.  |
| `given` claims approval where the real fixture registry denies it            | Test/script rejection, never an authoritative success.            |
| Module order changes or another module copies the approved site’s name       | Same authorization result; no order-dependent or name-only trust. |

---

## Deferred boundary — decisions fixed, mechanisms outside this deliverable

| Owner                                | Deferred work                                                                                                                                               | Fixed obligation                                                                                                                                        |
| ------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Fetching and execution layer**     | Network access, TLS/DNS enforcement, response verification, MIME/decoding checks, redirect handling, cache revalidation, and lifecycle checks at retrieval. | No unverified bytes, redirects, downgrade, stale approval, or unapproved fallback. Do not claim ordinary emitted HTML already enforces every condition. |
| **Deployment operation**             | Concrete principal enrollment, protected signing/acceptance configuration, secure document delivery, and withdrawal of previously deployed artifacts.       | Candidate code cannot select its trust root. No root or current authority means failure. Registry removal does not magically recall running code.       |
| **A future broader asset model**     | Closed dependency trees, runtime request mediation, page/DOM-bound fragments, additional asset roles, and additional integrity profiles.                    | No inherited approval or implied sandbox. Current `Html__Safe` output remains transferable after authorized construction.                               |
| **General Schema / HTTP / SQL / UI** | Everything unrelated to the asset minimum.                                                                                                                  | No generic validation, database, networking, component, or resource-management layer is implied. The a13 layering remains separate.                     |

**The resulting security claim is narrow and testable:** an authorized construction site may emit an external-asset element only for an exact, currently admitted registry request under the externally selected construction policy. The result preserves the information needed for retrieval verification; it does not assert that retrieval happened, that descendants are approved, or that the asset is harmless.

No implementation plan, compiler changes, or passing-gate claim is included.

[1]: https://www.w3.org/TR/2016/REC-SRI-20160623/ "Subresource Integrity"
[2]: https://www.w3.org/TR/SRI/ "Subresource Integrity"
[3]: https://www.w3.org/TR/CSP3/ "Content Security Policy Level 3"
