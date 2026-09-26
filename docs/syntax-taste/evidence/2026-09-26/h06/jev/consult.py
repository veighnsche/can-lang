"""Three fresh Jev consultations on H06 generation-handshake design decisions.

Every explanatory state field, instruction and option description is rewritten
in full across the three rounds. Technical identifiers, task IDs, numeric
facts and code spellings stay exact. No prior Jev answer appears in any
request. Usage: python3 consult.py (generate) ; python3 consult.py --send (3 calls).
"""
from pathlib import Path
import json
import os
import sys
import urllib.request
import datetime
import hashlib
import re

P = Path(__file__).resolve().parent

state = {
'program': [
'Can is an agent-oriented programming language with zero external users, so no backwards compatibility is owed to old syntax, ABI, spellings, generated TypeScript layouts, or goldens. Task H06 (requirements R13/R02/R03, shared interface C-H, owned by release engineering) implements the generation handshake and CAS-safe pairing: the browser sends its paired server generation on action transport, the server answers a typed generation-mismatch error, and the app presents a blocking refresh prompt. Lane boundaries are fixed: H owns driver staging, pairing, pruning, and distribution; E owns transport and server mismatch patches; C owns the app prompt; A owns generated emission if required. H hands the implemented C-H interface plus immutable candidate builds to C06, H10, and H12. No compatibility is owed across generations: old app logic against a new server is a defined error, never silent interop. Decide from the supplied facts without asking the user.',
'Judge a mechanism for Can, a programming language whose consumers are coding agents rather than people. Nobody outside the project uses it, which removes any duty to preserve earlier grammars, binary interfaces, source spellings, emitted TypeScript arrangements, or recorded goldens. Work item H06, drawn from R13/R02/R03 under the C-H interface and held by release engineering, delivers the handshake between browser and server generations together with content-addressed pairing: the client transmits the server generation it was paired with, the service replies with a typed mismatch failure, and the application blocks on a refresh prompt. Ownership splits are frozen: staging, pairing, pruning, and distribution stay with H; wire and service mismatch edits belong to E; the prompt belongs to C; emission belongs to A when needed. H passes the finished C-H surface and unchangeable candidate builds onward to C06/H10/H12. Generations promise each other nothing: stale client behavior against a fresh service is a specified fault, not quiet cooperation. Reach every selection below from this material alone, posing no question to the user.',
'This choice concerns a language built for AI coding agents, where dependable behavior outranks every other goal and the token cost of a task is a secondary measure. The project has no outside adopters, so keeping prior syntax, interfaces, spellings, generated file shapes, or goldens carries no weight. H06 (sourced from R13/R02/R03 through interface C-H, release-engineering owned) ships the cross-generation handshake with safe pairing: browsers attach their paired server generation to actions, servers respond to disagreement with a typed error, and applications answer that error with a blocking refresh. The lane split is settled: H keeps driver staging, pairing, pruning, and distribution; E takes transport plus server mismatch work; C takes the prompt; A takes emission only if required. C06, H10, and H12 receive the completed C-H contract and the immutable candidates from H. Between generations there is no interop promise: aged app logic meeting a newer server produces a defined failure rather than muted misbehavior. Make each call from the evidence given here, without further questions.',
],
'mechanism': [
'Today the browser is built first: `canlc build --target browser` emits `browser/manifest.json`, whose browser build ID is a `can-browser-bundle-v1` hash over the published files. The server build pairs it through `--browser-manifest`: `verifyBrowserManifest` checks build identity, digest-bound routes, entry, table, script/source-map pairing, generation binding, the shared locked snapshot via `bindSharedLock`, and the post-bundle audit. The pairing record (`can.browser-pairing`) is staged at `browser/pairing.json` inside the server generation and hash-bound by the server manifest, so it cannot name the server `buildID` (that would be circular: the ID is a `can-output-generation-v1` hash over staged content). The server manifest (`can.output-generation`) carries its own `buildID`, equal to the generation directory name. Staging links identical bytes from `dist/cas` as read-only 0400 hardlinks while manifests and metadata stay unique 0600 writes; selection runs through `selectCurrentWithAssets` with pending-ledger crash safety; replaced digest URLs stay servable for seven days through the `dist/assets.json` ledger plus the `dist/assets` store; `Prune` deletes non-current generations and sweeps unreferenced store entries. Rebuilding identical inputs reports a stable build ID.',
'Construction order is browser before server. A browser target build writes `browser/manifest.json`, identifying the bundle with a `can-browser-bundle-v1` digest computed across its file list. Server compilation consumes that file with `--browser-manifest`, and `verifyBrowserManifest` confirms the bundle identity, the digest-derived routes, the entry and table members, every script owning its map, the binding to the browser generation, agreement of the common dependency snapshot through `bindSharedLock`, and the assembled-bundle audit. The resulting `can.browser-pairing` document lands at `browser/pairing.json` within the server generation and is pinned by the server manifest digest, which forbids it from quoting the server `buildID`: that value is a `can-output-generation-v1` digest across staged bytes, so quoting it there would hash itself. The `can.output-generation` manifest records that same `buildID`, matching its generation folder name. Identical staged bytes are hardlinked read-only (0400) out of `dist/cas`; manifests with metadata remain private 0600 files; promotion uses `selectCurrentWithAssets` with a pending record plus ledger replay after crashes; superseded digest URLs keep serving for seven days via the `dist/assets.json` roster and `dist/assets` bytes; `Prune` removes generations that are not current and garbage-collects orphaned store records. Rebuilding unchanged sources yields the same build ID.',
'The pipeline builds the client bundle ahead of the service. Browser compilation produces `browser/manifest.json` carrying a `can-browser-bundle-v1` digest over published outputs as the browser build ID. Service compilation attaches that manifest via `--browser-manifest`; `verifyBrowserManifest` then validates bundle identity, digest-locked routes, entry and table presence, script-to-map completeness, the tie to the browser generation, the overlapping lockfile pins with `bindSharedLock`, and the final bundle audit. Its `can.browser-pairing` record is stored at `browser/pairing.json` as part of the server generation and sealed by that generation manifest hash, hence it can never state the server `buildID` — the identifier is a `can-output-generation-v1` digest of the staged tree and would reference its own hash. Instead the `can.output-generation` manifest holds the `buildID`, identical to the generation directory label. Shared staged bytes arrive as 0400 hardlinks from `dist/cas`; manifests and metadata stay single-owner 0600 writes; `selectCurrentWithAssets` promotes with crash-safe pending state; replaced digest routes remain reachable seven days through the `dist/assets.json` ledger and `dist/assets` storage; `Prune` drops every generation beside current and collects unreferenced content. A rebuild from the same inputs prints an unchanged build ID.',
],
'limits': [
'Jev is a classifier, not a researcher: it supplies typed choices with probabilities, never prose reasons or new facts. Its distributions are advisory only, never acceptance proof; these calls execute no Can code and calibrate nothing. The E, C, and A slices of the handshake (transport/server mismatch, app prompt, emission) are specified as handoffs and are not yet implemented; judge the H-side mechanism and the contract they must satisfy, not their code. Treat each round independently: no request carries any prior answer. Decide every question below from the supplied state alone.',
'The model here classifies rather than investigates: answers are typed selections plus distributions, without rationale text or fresh evidence. Such scores guide engineering judgment but prove nothing; this round runs no compiler, no runtime, and no qualification. Transport and server mismatch edits (E), the prompt (C), and any emission (A) exist only as handed-off slices, so evaluate the build-side design and the obligations those slices inherit rather than reviewing implementation. Rounds stand alone with no earlier verdict leaking into later state. Ground each answer strictly in the material provided.',
'Expect no research from this consultation: the system returns a chosen option with a probability spread, not explanations or discoveries. Those numbers advise the design and certify no behavior; nothing here executes Can programs or measures domain accuracy. E wire and service changes, C prompt work, and A emission (if any) are pending handoffs, so the judgment covers the H-side build shape and the interface those lanes must honor, not finished code. Each of the three rounds is self-contained; later requests contain no trace of earlier outcomes. Answer only from the context handed to this call.',
],
}

questions = {
'acquisition': {
'instructions': [
'How should the browser bundle learn the paired server generation it must send on action transport? Judge fail-closed behavior, staleness races, and fit with the one-way build order.',
'Choose the channel that teaches the browser its paired server generation. Weigh silent-staleness hazards, extra requests, and consistency with browser-before-server construction.',
'Decide where the browser obtains the server generation it attaches to actions. Favor the path that keeps an old page honestly old without added round trips or freshness races.',
],
'criteria': {
'page_embedded': [
'The served page embeds its own serving generation — the server generation `buildID` — in one fixed machine-readable slot. Browser logic reads that slot from the live document and sends the value on every action request. A page rendered by generation N keeps sending N after a rollout to N+1, so the new server answers the typed mismatch and the app shows the blocking refresh. The identity travels with the document that defines the app logic: no extra request, no cache race, and an old page can never claim to be new.',
'Each document the server renders stamps the generation that rendered it — that generation `buildID` — into a single stable machine-readable location. Client logic copies the stamp out of the displayed page onto all outgoing actions. When service N+1 replaces N, the earlier document still carries N, the comparison fails, the typed error returns, and the blocking refresh appears. Generation knowledge rides alongside the markup it describes, adding zero round trips and zero freshness windows, while a stale page stays detectably stale.',
'Rendering writes the renderer identity — the serving generation `buildID` — into an agreed machine-readable field of the page itself. The browser lifts that field from the current document for every action it issues. After promotion past the rendering generation, the stored value disagrees, the service emits the typed mismatch, and the application blocks behind its refresh prompt. Because the value ships inside the artifact whose behavior it labels, there is no separate fetch to cache or mistime, and age is always visible.',
],
'metadata_route': [
'The browser fetches a served pairing-metadata route (dist metadata outside staged content) to learn the current server generation, then attaches it to actions. This costs an extra request per page or action window and opens staleness races: a cached metadata read can reintroduce the mismatch it was meant to prevent, while a fresh read can dress an old page in a new generation.',
'Client code queries a live metadata endpoint for the serving generation before acting. Besides the added round trip, the read can disagree in both directions with the acting page: cached data manufactures a mismatch on a current page, and current data hides the age of an outdated one.',
'The serving generation is discovered through a dedicated metadata fetch rather than the page. The lookup adds latency and a second freshness timeline; skew between the two timelines either blames fresh pages or excuses stale ones.',
],
'browser_build_echo': [
'The browser bakes its own browser build ID (known when the browser bundle is built) into the bundle and sends that value; the server compares it against the paired browser build ID in `browser/pairing.json`. This contradicts the H06 scope sentence, which requires the paired *server* generation on the wire, and it cannot distinguish two server generations paired with one browser build.',
'Send the bundle-owned browser build identifier — fixed at browser build time — and match it server-side to the paired ID inside `browser/pairing.json`. That replaces the mandated server-generation value with a different identity and goes blind whenever one browser build pairs with successive server generations.',
'Echo the browser build ID from the bundle and check it against the pairing record browser entry. The wire then carries a client-build label instead of the required server generation, losing the distinction between consecutive services that share a client bundle.',
],
},
},
'comparison': {
'instructions': [
'How should the server establish the expected generation and answer disagreement? Judge fail-closed strength, identity soundness, and behavior in both rollout and rollback directions.',
'Choose the server-side check and its failure answer. Weigh quiet-misbehavior risk, what counts as the expected value, and symmetry between promoting and reverting.',
'Decide how the service verifies the attached generation and what it returns on mismatch. Prefer loud typed refusal from a stable identity, identical forward and backward.',
],
'criteria': {
'manifest_startup_typed': [
'The server reads its own generation `manifest.json` `buildID` once at startup and compares it with the generation attached to every action request. Missing, malformed, or unequal values fail closed with the typed generation-mismatch error before any action logic runs. Generations are content hashes with no ordering, so the check is symmetric string inequality: rollback mismatches refuse exactly like rollout mismatches. Replaced digest asset URLs keep serving through retention, but aged logic against a fresh server is the defined error, never quiet interop.',
'At boot the service loads the `buildID` from the `manifest.json` of the generation it runs from, then matches that value against each incoming action attached generation. Absent, ill-formed, or differing values produce the typed mismatch failure with no domain logic executed. Because generation IDs are unordered content digests, any inequality refuses in either direction: reverting trips the same loud fault as advancing. Superseded digest routes remain reachable inside their retention window, while stale behavior against a newer service stays a specified fault rather than silent cooperation.',
'Startup pins the expected value from the serving generation manifest `buildID`; every action compares its attached generation to that pin. Omitted, corrupt, or non-equal inputs yield the typed mismatch before any handler runs. Content-hash IDs admit no ordering, so inequality refuses symmetrically — a step back fails as loudly as a step forward. Old digest URLs survive under retention, yet old logic facing a new server meets the defined error instead of muted misbehavior.',
],
'current_per_request': [
'The server re-reads `dist/current.json` on every request and judges the attached generation against the live selection pointer. This adds per-request IO, mistakes the mutable promotion pointer for the serving generation identity, and races selection moves: files from one generation can be judged against a pointer that already advanced.',
'Each request reloads the `dist/current.json` selection and compares to that moving target. Beyond the repeated reads, the pointer answers which build is selected, not which bytes are serving, so a mid-flight promotion misjudges in-flight work.',
'Look up the current-selection file per action and treat its value as truth. The lookup repeats needlessly, confuses the mutable selector with fixed serving identity, and lets promotion timing rewrite the verdict under running requests.',
],
'log_and_serve': [
'The server records the mismatch in a log and executes the action anyway. Old logic then runs against a new server with no signal to the app, which is exactly the silent wrong behavior the fail-closed policy forbids.',
'Note the disagreement internally but serve the request regardless. The application never learns its generation is stale, so cross-generation behavior proceeds quietly in violation of the defined-error rule.',
'Tolerate the mismatch with a log entry while running the handler normally. Stale clients operate against fresh services unnoticed, defeating the handshake entirely.',
],
},
},
'rollback': {
'instructions': [
'Given that prune deletes non-current generations, how should rollback to a prior paired build be provided? Judge disk cost, GC surface, and whether the rolled-back state is genuinely the old behavior.',
'Choose the rollback story under a prune that keeps only the current generation. Weigh retained bytes, collector complexity, and honest restoration of prior logic.',
'Decide what operators do to revert to an earlier paired build. Favor the path that restores true prior behavior without hoarding stale generations.',
],
'criteria': {
'rebuild_stable': [
'Rollback rebuilds the prior source: stable build IDs reproduce the byte-identical generation, which promotes through the same atomic selection path as any rollout. Replaced assets stay servable under the seven-day ledger either way. Prune keeps its contract, no stale logic lingers on disk, and the collector gains no new roots.',
'Reverting recompiles the earlier sources, whose stable identifiers regenerate the identical bytes, then selects the result through the ordinary atomic promotion. The seven-day asset roster covers replaced routes in both directions. Nothing about pruning changes, no outdated service tree accumulates, and garbage collection tracks no extra references.',
'To revert, rebuild from the previous inputs and promote the outcome normally: identifier stability recreates the exact generation, and retention serves replaced digests regardless of direction. The prune rule stands untouched, stale generations never pile up, and the content-store sweep keeps its current root set.',
],
'retain_prior': [
'Prune keeps the last N generations so rollback reselects without rebuilding. Stale server logic and paired bytes accumulate on disk, every retained tree widens the GC roots, the prune contract changes, and a stale generation can be served or qualified by mistake.',
'Hold several superseded generations beside current for instant reselection. The disk keeps outdated services and their bundles, collectors must trace them all, pruning rules grow exceptions, and operating or testing the wrong generation becomes possible.',
'Never delete recent generations; revert by reselecting one. Outdated logic piles up locally, the sweep inherits a larger live set, prune semantics complicate, and stale builds remain one misstep from production.',
],
'assets_only_rollback': [
'Rollback restores only retained asset bytes while server logic stays current. Old bytes under new logic are precisely the defined-error case, so this rollback still fails every old action; it reverts nothing observable and corrupts the retention story it borrows from.',
'Revert the bytes but keep the current service: retained assets return while handlers stay new. Since aged artifacts against fresh logic are the specified fault, every action still refuses — an apparent revert that changes no behavior and misuses retention.',
'Roll back assets alone, leaving the serving generation in place. Old resources meeting new code are the defined mismatch, and so actions keep failing; the operation restores nothing real while tangling retention with recovery.',
],
},
},
}

for entries in state.values():
    assert len(entries) == 3 and len(set(entries)) == 3, "each state field needs 3 distinct phrasings"
for q in questions.values():
    assert len(q['instructions']) == 3 and len(set(q['instructions'])) == 3, "each question needs 3 distinct instructions"
    for entries in q['criteria'].values():
        assert len(entries) == 3 and len(set(entries)) == 3, "each option must have 3 distinct phrasings"


def grams(text):
    words = re.findall(r"[A-Za-z0-9`*_./-]", text)
    return words


def word_grams(text, n):
    words = text.split()
    return {" ".join(words[i:i + n]) for i in range(max(0, len(words) - n + 1))}


problems = []
groups = []
for name, entries in state.items():
    groups.append((f"state:{name}", entries))
for qname, q in questions.items():
    groups.append((f"question:{qname}:instructions", q['instructions']))
    for oname, entries in q['criteria'].items():
        groups.append((f"question:{qname}:option:{oname}", entries))
for label, entries in groups:
    for a in range(3):
        for b in range(a + 1, 3):
            shared = word_grams(entries[a], 8) & word_grams(entries[b], 8)
            if shared:
                problems.append({"group": label, "pair": [a + 1, b + 1],
                                 "shared_8grams": sorted(shared)[:5]})

for i in range(3):
    payload = {
        'model': 'jev-latest',
        'state': {k: v[i] for k, v in state.items()},
        'questions': {
            k: {
                'type': 'choice',
                'instructions': q['instructions'][i],
                'criteria': {key: v[i] for key, v in q['criteria'].items()},
            } for k, q in questions.items()
        },
    }
    (P / f'request-{i+1}.json').write_text(json.dumps(payload, indent=2) + '\n')

hashes = {}
for i in range(1, 4):
    body = (P / f'request-{i}.json').read_bytes()
    hashes[f'request-{i}.json'] = hashlib.sha256(body).hexdigest()
(P / 'wording-audit.json').write_text(json.dumps({
    'review': 'Manually checked equal facts, constraints and alternatives across all three requests before sending. Each of the 3 state fields, 3 instructions and 9 option descriptions has a distinct complete phrasing (30 fields). Task IDs (H06/C06/H10/H12), requirement IDs (R13/R02/R03), interface C-H, file paths, kind strings, function names, 0400/0600 modes, the seven-day bound and code spellings are intentionally stable. No response or favored recommendation appears in any later request state.',
    'same_facts': [
        'Can agent language; zero external users; no compat for syntax/ABI/spellings/TS layouts/goldens; H06 handshake + CAS pairing under R13/R02/R03 C-H, release-engineering owned; browser sends paired server generation, server typed mismatch, app blocking refresh; lane split H staging/pairing/pruning/distribution, E transport/server, C prompt, A emission; handoff to C06/H10/H12; no cross-generation compat, fail-closed defined error',
        'Browser built first (canlc build --target browser, browser/manifest.json, can-browser-bundle-v1 ID); server pairs via --browser-manifest; verifyBrowserManifest checks identity/routes/entry/table/maps/generation binding/bindSharedLock/audit; can.browser-pairing at browser/pairing.json hash-bound by server manifest so it cannot quote server buildID (can-output-generation-v1 circularity); can.output-generation manifest carries buildID equal to dir name; dist/cas 0400 hardlinks, metadata unique 0600; selectCurrentWithAssets crash-safe selection; dist/assets.json + dist/assets seven-day retention; Prune deletes non-current + sweeps CAS; rebuilds stable',
        'Limits: Jev classifies, no research; advisory only, no acceptance proof; no Can code runs, no calibration; E/C/A slices are pending handoffs, judge H-side mechanism + contract; rounds independent, no prior answer leaks; decide from supplied state',
    ],
    'limit': 'Text inequality plus manual semantic review cannot prove absence of framing effects; investigate disagreement and treat agreement as advice, not proof.',
    'shared_8gram_violations': problems,
    'request_sha256': hashes,
}, indent=2) + '\n')
print(json.dumps({'generated': hashes, 'shared_8gram_violations': problems}, indent=2))

if '--send' in sys.argv:
    if problems:
        print(json.dumps({'refused': 'shared 8-gram violations present; reword before sending'}),
              file=sys.stderr)
        sys.exit(1)
    key = os.environ['TYPESAFE_API_KEY']
    for i in range(1, 4):
        req = urllib.request.Request(
            'https://api.typesafe.ai/v1/systemone',
            data=(P / f'request-{i}.json').read_bytes(),
            headers={'Content-Type': 'application/json', 'Authorization': 'Bearer ' + key},
        )
        start = datetime.datetime.now(datetime.timezone.utc).isoformat()
        with urllib.request.urlopen(req, timeout=60) as r:
            body, status = r.read(), r.status
        (P / f'response-{i}.json').write_bytes(body + b'\n')
        (P / f'response-{i}.metadata.json').write_text(json.dumps(
            {'startedAt': start, 'status': status, 'endpoint': 'v1/systemone'}, indent=2) + '\n')
        print(json.dumps({'request': i, **json.loads(body)}))
