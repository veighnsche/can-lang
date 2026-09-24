"""Three fresh wording variants for the unresolved DI-05 instance rule."""
from pathlib import Path
import datetime
import json
import os
import sys
import urllib.request

OUT = Path(__file__).resolve().parent

state = {
    "goal": [
        "Can is written by AI coding agents. Correctness and reliable edits lead; total tokens per successful task are a secondary measurement. There are no external users or compatibility duties. Design a future package/error identity contract, not a claim of implementation.",
        "Choose an identity rule for an agent-authored language with no outside adopters to migrate. A correct, repairable graph matters before secondary whole-task token cost. This decision is for a planned compiler contract only.",
        "The planned Can redesign targets AI agents, including syntax that may be harsh for people. Favor sound composition and predictable changes, then measure successful-task tokens. Current users do not impose migration compatibility; no candidate is built yet."
    ],
    "observed": [
        "The current loader gives each dependency a graph-wide key and ID can.project.dependency/<key>, rejects one key targeting two directories, and rejects two keys for one real directory. Its package map is globally keyed by short name. A root lock snapshots each dependency under that global key with a path, manifest/source/fixture digests and numeric error registry. Parser imports only bare package names. The user selected future source uses [billing::model as bill_model, crm::model as crm_model], where billing and crm are direct dependency keys. Current application errors need global integers, but accepted future design uses unnumbered declarations and owner-qualified textual report keys. Matching already uses nominal declaration plus concrete generic arguments.",
        "Today project identity is derived from a single globally interpreted dependency key; the loader disallows reuse of a key for different directories and two keys for the same directory. Short package names also have one flat global table. Locked dependency rows contain root-relative paths, manifest/source/fixture hashes and active/retired numeric allocations. Existing uses entries have no dependency qualifier. The user chose uses [billing::model as bill_model, crm::model as crm_model] for the planned syntax, with each key relative to its importing project. Planned application errors drop authored numbers in favor of qualified textual identity; current nominal checks distinguish full generic specializations.",
        "Current graph IDs depend on global dependency aliases, and source package lookup uses global short names. The loader rejects both conflicting alias-to-directory mappings and duplicate aliases of one real directory. Root lock entries pin path plus three digests and numeric error registry. Planned source qualification is fixed by the user's answer: uses [billing::model as bill_model, crm::model as crm_model], with dependency-relative keys. Separately accepted unnumbered application errors need an owner-qualified textual report key; type matching already distinguishes generic arguments."
    ],
    "required": [
        "Two unchanged libraries may both expose model and be imported directly; an importer may not reach an undeclared transitive package. Local aliases and checkout absolute paths cannot define nominal identity. Repeated references to the same resolved release coalesce; two independently locked releases or versions stay distinct. Rename a dependency key without changing its target, add an unrelated dependency, or relocate the checkout: existing error report keys stay the same. Locks must deterministically detect content/edge conflicts and freeze the resolved graph. Retired declaration names cannot be reused within their owner's promised scope, but foreign tombstones cannot reserve them. Avoid globally coordinated numeric errors. No source rewrite of either library should be needed merely to resolve duplicate short package names; a one-time manifest/lock schema migration is allowed.",
        "The target graph allows independently authored model packages in one caller and confines imports to direct dependencies. An alias, absolute checkout directory or unrelated node cannot change a nominal error key. Shared occurrences of one locked release merge, while different releases remain separate even if their package names match. The same checked graph after dependency-key rename or relocation must preserve unchanged declaration identities; graph growth by an unrelated project does too. Reject contradictory pin data or target edges. Owner-local retirement prevents its own reuse only. The redesign may replace manifests and locks because compatibility is not required; it should not force vendor source renaming for composition.",
        "Accept duplicate short package declarations from two independent dependencies, without granting access to transitive-only packages. Identity must survive local import alias edits, direct dependency-key edits that retain the same resolved target, physical relocation and unrelated graph expansion. Multiple paths to the exact locked release should share one instance; separately selected releases must never merge by short name. Validate lock pins and graph edges reproducibly, enforce owner-scope error tombstones without foreign reservations, and remove application integer coordination. Manifest/lock migration is permitted, but unchanged library source must still compose."
    ]
}

instruction = [
    "Which canonical project-instance and lock rule best supports these package and error contracts while minimizing unproved assumptions?",
    "Select the planned owner and resolved-release identity scheme that can satisfy every stated merge, stability and retirement requirement.",
    "For the future compiler specification, which of these complete ID/lock approaches should engineering adopt given the observed loader and required tests?"
]

criteria = {
    "publisher_release": [
        "Each project manifest declares a persistent publisher-owned project UUID and an immutable release-instance UUID. The root lock records both IDs, checked content/edge digests and direct dependency targets for every graph node. The same pair with identical pinned content and edges coalesces; a pair with conflicting bytes or edges fails; distinct releases remain distinct. Package/error nominal keys include the pair, package and declaration. A publisher-scoped retirement ledger forbids name reuse across its releases; alias/key edits and relocation do not change keys. Authors must mint release IDs and preserve the stable project ID.",
        "Use two explicit manifest identities: a long-lived owner UUID and a release UUID. Lock every node's IDs, content hashes and resolved outgoing edges. Identical ID pairs and pins share an instance, disagreements reject, and different release IDs coexist. Qualified type/report keys contain owner plus release plus package/declaration. A separate owner-level retired-name set applies across releases. Dependency labels and filesystem location are only lookup paths. This adds publisher ID administration but gives root-independent identity.",
        "Put persistent project UUID and immutable release-instance UUID in owning manifests, then pin their pair and verified tree/edge state in the lock. Intern repeated exact instances by pair; reject duplicate pairs with divergent payload or edges and keep distinct releases separate. Form nominal names from those IDs with package/declaration and use project-level tombstones across releases. Local dependency keys and paths select nodes rather than defining identity. The publisher manages IDs and release changes."
    ],
    "root_lock_instance": [
        "Let each root lock allocate stable opaque UUIDs for resolved nodes; record canonical path/edge pins and keep UUIDs when alias keys change or unrelated nodes are added. Equal lock UUID with equal pins coalesces; distinct UUIDs separate releases. Error keys include the root-assigned UUID, package and declaration; retired names live in per-root locked metadata. Checkout moves can preserve the lock. The same library may get different report identity under separate roots and publisher-wide retirement needs reconciliation.",
        "Assign each loaded project an instance UUID in the importing application's lock instead of its manifest. Reuse that ID across local dependency-label changes and graph growth, bind it to verified content and edges, and merge matching lock IDs. Source error keys reference the lock UUID. A root can relocate with its lock, but unrelated importers may assign different IDs and owner-level tombstones become root-local or require another mechanism.",
        "Make the root lock the authority for random node instance IDs, preserving each ID through local edge renaming and physical relocation. Bind IDs to payload and outgoing target data, coalesce identical pins, separate independently assigned IDs, and qualify error keys by lock instance. This avoids publisher release metadata but leaves cross-root declaration identity and publisher-wide retirement unclear without extra coordination."
    ],
    "content_instance": [
        "Compute each instance ID from normalized manifest, source, fixtures and resolved dependency-target digests, excluding absolute paths and alias spellings. The lock verifies that Merkle-style ID and edges; equal content coalesces, changed source or dependency target gets a new ID. Error keys include that digest, package and declaration. Relocation and unrelated graph growth are stable; even a small owner edit changes all nominal keys, and retired-name policy needs separate durable owner metadata.",
        "Derive graph identity from a cryptographic digest of canonical project contents and resolved outgoing nodes. Lock the hash tree; equal trees merge and different versions separate without minted IDs. Normalize dependency labels and machine paths so alias edits and relocation do not perturb the hash. Source edits inevitably produce fresh nominal/report identities even for unchanged declarations, so retirement requires an additional stable owner namespace.",
        "Build a path-independent instance hash from the project tree plus pinned dependency graph, using the lock to verify it. Equal fingerprints intern and different snapshots remain separate; spelling-only local handles should be normalized out. Reports use this fingerprint with package/declaration. An unrelated graph node leaves hashes alone, but any edit inside one project changes every error identity and an owner-level tombstone ledger cannot follow the hash alone."
    ]
}

for entries in state.values():
    assert len(entries) == 3 and len(set(entries)) == 3
assert len(set(instruction)) == 3
for entries in criteria.values():
    assert len(entries) == 3 and len(set(entries)) == 3

for index in range(3):
    request = {
        "model": "jev-latest",
        "state": {name: entries[index] for name, entries in state.items()},
        "questions": {
            "canonical_instance": {
                "type": "choice",
                "instructions": instruction[index],
                "criteria": {name: entries[index] for name, entries in criteria.items()},
            }
        },
    }
    (OUT / f"request-{index + 1}.json").write_text(json.dumps(request, indent=2) + "\n")

(OUT / "wording-audit.json").write_text(json.dumps({
    "manualReview": "All three requests state the same current graph/lock behavior, fixed user import syntax, accepted unnumbered error direction, required alias/relocation/graph-growth stability, exact-instance merging, version separation, and owner retirement. Every explanatory field, instruction and option description is fresh prose; technical syntax and identifiers are stable. Options remain in the same order. No preferred answer or previous Jev response was included.",
    "mechanicalReview": "Every three-member explanatory prose set is pairwise distinct by exact string comparison; all request files are retained.",
    "limit": "Semantic equivalence is manually reviewed and cannot prove freedom from phrasing effects. Jev supplies judgment, not implementation proof."
}, indent=2) + "\n")

if "--send" in sys.argv:
    key = os.environ["TYPESAFE_API_KEY"]
    for index in range(1, 4):
        payload = (OUT / f"request-{index}.json").read_bytes()
        req = urllib.request.Request(
            "https://api.typesafe.ai/v1/systemone",
            data=payload,
            headers={"Content-Type": "application/json", "Authorization": "Bearer " + key},
        )
        started = datetime.datetime.now(datetime.timezone.utc).isoformat()
        with urllib.request.urlopen(req, timeout=60) as response:
            body = response.read()
            status = response.status
        (OUT / f"response-{index}.json").write_bytes(body + b"\n")
        (OUT / f"response-{index}.metadata.json").write_text(json.dumps({
            "startedAt": started, "status": status, "endpoint": "v1/systemone"
        }, indent=2) + "\n")
        data = json.loads(body)
        print(json.dumps({"request": index, "model": data.get("model"), "answers": data.get("answers"), "usage": data.get("usage")}), flush=True)
