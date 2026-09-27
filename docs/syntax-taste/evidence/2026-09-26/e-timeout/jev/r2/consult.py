"""Three fresh Jev consultations on E-TIMEOUT R2 (overrun-tx peer release).

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

P = Path(__file__).resolve().parent

state = {
'program': [
'Can is an agent-oriented programming language with zero external users, so no backwards compatibility is owed to old syntax, ABI, spellings, generated TypeScript layouts, or goldens. E-TIMEOUT R2 releases the at-budget peer for overrun transactions: E09 W5-L3 measures the handler boundary returning sql::commit_unknown at 155ms against a 150ms bound while the peer receives its unknown:commit body at 2009ms, at transaction-callback settlement rather than bound expiry, because dispatch drains the per-request owner scope before the Response resolves. A hung transaction callback would therefore hold the peer past any bound. The user ruled that bounded waiting applies to the client response too, that internal timeout detection while holding the response is insufficient, and that a timeout must report uncertainty honestly without ever implying the write was cancelled or rolled back. The L3 leg pins the split (handler under 1500ms, peer at or past 1800ms and under 8000ms) and must flip to bounded-peer assertions. Decide from the supplied facts without asking the user.',
'Judge peer-release choices for Can, a programming language whose consumers are coding agents rather than people. Nobody outside the project uses it, which removes any duty to preserve earlier grammars, binary interfaces, source spellings, emitted TypeScript arrangements, or recorded goldens. Work item E-TIMEOUT R2 owns the W5-L3 overrun-transaction shape: with a 150ms bound the withTransaction boundary answers sql::commit_unknown handler-side at 155ms, yet the HTTP peer waits until 2009ms for the unknown:commit body because the request owner-scope drain gates the Response on transaction-callback settlement. An unsettled callback would pin the client beyond every deadline. Preparation return R2 records the user verdict: the response itself must be bounded, silent in-handler expiry is not enough, ownership plus cleanup need proper design and validation, and no timeout may hint at cancellation or rollback. The L3 pins (handler below 1500ms, peer from 1800ms to 8000ms) exist to flip into bounded-response assertions. Reach every selection below from this material alone, posing no question to the user.',
'This choice concerns bounded client responses for overrun transactions in a language built for AI coding agents, where dependable behavior outranks every other goal. The project has no outside adopters, so keeping prior syntax, interfaces, spellings, generated file shapes, or goldens carries no weight. E-TIMEOUT R2 addresses the E09 W5-L3 finding: the transaction boundary honors its 150ms bound handler-side (sql::commit_unknown at 155ms with budget escalation and owned late settlement), but dispatch answers the peer at 2009ms when the transaction callback settles, since the per-request scope drain runs before the Response returns. Any callback that never settles would therefore trap the peer indefinitely. The user commissioned engineering work with the constraint that the client response is part of bounded waiting, that detecting expiry internally while withholding the answer fails the requirement, and that uncertainty must be reported without suggesting the write stopped or rewound. Make each call from the evidence given here, without further questions.',
],
'mechanism': [
'Dispatch in runtime/platform/server.ts serveNative wraps each request in one ambient RequestScope (shared budget, abort signal, escalation collector) plus one owner withScope whose drain must empty before the served Response resolves. withTransaction in runtime/platform/sql/transaction.ts runs its guarded callback as an owner callback task inside a transaction inner scope nested in the request scope, so the request drain waits for callback settlement even though raceBoundary already returned commit_unknown at the bound; the measured split is handler 155ms versus peer 2009ms beside a 2s stall. Plain pool-lease legs (W5-L1/L2/L6/L7) already release the peer at the bound because leases hang off the pool resource outside the request scope, and only callback tasks plus groups inside the scope gate the drain. stop expires every live request scope with shutdown, closes sessions, then closes the server resource with a shutdownMs deadline where a past-deadline close stays owned for the external supervisor; wait bounds a signalled drain by shutdownMs the same way. The E04 honesty rules bind every outcome: a budget expiry never claims rollback or non-start (except the never-started transaction phase, which ran nothing by construction), commit_unknown is never retried and reconciles by reread, leases are never timer-revoked, markers stay frozen, and no post-disposal use is allowed. revokeRequest runs after drainage today, and serveOuter funnels drain failures into the redacted request report. This task may edit server dispatch plus runtime/transport plus tests plus evidence; runtime/platform/sql internals including withTransaction, and runtime/owner-core.ts, are read-only.',
'Each served request passes through two scopes in runtime/platform/server.ts serveNative: an ambient RequestScope carrying the shared budget, the abort signal, and the escalation collector, and an owner withScope that drains fully ahead of Response delivery. The transaction path in runtime/platform/sql/transaction.ts executes its guarded callback as a tracked owner task in an inner transaction scope beneath the request scope, forcing request drainage to await callback completion despite the boundary already yielding commit_unknown at expiry; observations show 155ms handler-side against 2009ms peer-side next to a 2s stall. Ordinary leased queries (legs W5-L1/L2/L6/L7) answer on time because their leases attach to the pool resource beyond request scope, while drain gating covers exactly the groups and callback tasks nested inside. Shutdown expires all live request scopes as shutdown, shuts sessions, and closes the server token under a shutdownMs deadline with overdue closes remaining owned for an outside supervisor; signalled waits apply the identical shutdownMs cap. E04 honesty law governs results: budget expiry asserts neither rollback nor non-start (apart from the never-started transaction phase that executed nothing), commit_unknown forbids retry in favor of reread reconciliation, timers revoke no lease, markers are immutable, and disposed handles admit no further use. Snapshot revocation currently follows drainage, while serveOuter routes drain faults to redacted request reporting. Editable surface here is server dispatch, runtime/transport, tests, and evidence; the SQL internals with withTransaction and runtime/owner-core.ts stay untouched.',
'Server dispatch in runtime/platform/server.ts serveNative installs per request both an ambient RequestScope (joint budget, abort signal, escalation sink) and an owner withScope emptied strictly before the Response goes out. withTransaction from runtime/platform/sql/transaction.ts runs its protected callback as a tracked owner chore within a transaction child scope under the request scope, so request draining stalls for callback settlement after raceBoundary has long since produced commit_unknown at the bound; the recorded gap is 155ms at the handler against 2009ms at the peer alongside a 2s stall. Leased non-transaction queries in W5-L1/L2/L6/L7 meet their bound at the peer because leases dangle from the pool resource outside request scope, whereas the drain blocks solely on inner groups and callback tasks. stop marks every live request scope shutdown, drops sessions, then shuts the server resource inside a shutdownMs deadline where belated closes persist owned under external supervision; wait caps signal-triggered drainage by the same shutdownMs. E04 candor rules control every expiry: no rollback or non-start claim (save the never-started transaction phase, empty by construction), commit_unknown never retries and heals via reread, no timer revokes a lease, markers freeze at creation, and disposal ends use. revokeRequest today trails drainage, and serveOuter delivers drain failures to the redacted request reporter. This slice may change server dispatch, runtime/transport, tests, and evidence; SQL internals including withTransaction plus runtime/owner-core.ts are read-only.',
],
'limits': [
'Jev is a classifier, not a researcher: it supplies typed choices with probabilities, never prose reasons or new facts. Its distributions are advisory only, never acceptance proof; these calls execute no Can code, start no server leg, and calibrate nothing. A timeout must report uncertainty honestly and never imply the write was cancelled or rolled back. Treat each round independently: no request carries any prior answer. Decide every question below from the supplied state alone.',
'The model here classifies rather than investigates: answers are typed selections plus distributions, without rationale text or fresh evidence. Such scores guide engineering judgment but prove nothing; this round runs no compiler, no runtime, no Postgres leg, and no live server. Any expiry outcome must stay honest about uncertainty and must not suggest cancellation or rollback occurred. Rounds stand alone with no earlier verdict leaking into later state. Ground each answer strictly in the material provided.',
'Expect no research from this consultation: the system returns a chosen option with a probability spread, not explanations or discoveries. Those numbers advise the design and certify no behavior; nothing here executes Can programs, serves traffic, or measures domain accuracy. Every timeout report must confess uncertainty and never hint that the write stopped or rewound. Each of the three rounds is self-contained; later requests contain no trace of earlier outcomes. Answer only from the context handed to this call.',
],
}

questions = {
'peer_release': {
'instructions': [
'How should the overrun-transaction peer gain its bound? Judge at-budget response delivery, preservation of the honest commit-unknown body, and blast radius beyond transaction requests.',
'Choose the peer-release shape for overrun transactions. Weigh bounded client timing, keeping the truthful unknown body, and effects on requests without owned callbacks.',
'Decide how dispatch frees the peer when transaction callbacks overrun. Favor on-time answers, intact uncertainty vocabulary, and minimal disturbance to ordinary requests.',
],
'criteria': {
'respond_then_drain': [
'Publish the handler Response through a side channel the moment the handler completes, release the peer immediately, and drain the request owner scope owned in the background: transaction callbacks settle, leases release, scoped resources close, and the drain still converges exactly as today, only after the client is answered. The peer receives the handler honest unknown:commit body at the bound instead of at settlement; stop and wait keep bounding shutdown through shutdownMs with overdue closes staying owned for the supervisor. Requests without pending owned work observe no change because their drain already completes instantly; only overrun shapes shift earlier. Ownership, commit_unknown, escalation, and late-settlement semantics stay exactly as qualified, with response timing decoupled from drain timing.',
'Answer the client from a side channel the instant handler work finishes while the request owner scope drains behind the response under continued ownership: callbacks land, leases drop, scoped resources shut, and drainage reaches the same end state, merely after peer delivery. The honest unknown:commit body lands at the bound rather than at callback settlement, and shutdown keeps its shutdownMs cap with late closes owned for external supervision. Ordinary requests lacking unsettled owned tasks behave identically since instant drains stay instant; solely overrun patterns move sooner. Qualified ownership plus commit_unknown plus escalation plus late-observation semantics persist untouched while response and drain schedules separate.',
'Deliver the handler Response via a side path as soon as it exists, freeing the peer at once, then empty the request owner scope asynchronously under ownership: callback settlement, lease release, and scoped-resource closure all still happen to the same settled conclusion, only behind the answered client. The truthful unknown:commit payload arrives on bound instead of on settlement; stop and wait retain shutdownMs bounding with tardy closes owned for the supervisor. Requests carrying no pending owned work see zero difference because immediate drains remain immediate, and only overrun cases answer earlier. All qualified ownership, commit_unknown, escalation, and late-record behavior survives with response time unhooked from drain time.',
],
'gated_with_fixed_timeout': [
'Keep the Response gated on the request-scope drain and race that drain against a fixed dispatch bound; when the drain wins the peer gets the handler body, but when the bound wins the peer gets a fixed 408 or 500 instead of the handler honest unknown:commit answer. This preserves response-after-drain ordering at the price of discarding the carefully qualified commit-unknown vocabulary exactly when it matters most: the overrun peer learns a generic timeout rather than the reconcilable transaction id. Every slow-drain request, transaction or otherwise, now risks a fixed status, widening blast radius beyond the overrun shape. Ordering purity trades against honest content on the path the requirement exists to fix.',
'Retain drain-before-response sequencing while bounding the drain with a fixed dispatch timer; a drain that finishes first yields the handler body, whereas a timer victory substitutes a fixed 408 or 500 for the handler candid unknown:commit reply. Order is preserved, but the qualified reconcilable vocabulary is thrown away on precisely the overrun path it was built for, leaving the client a bare timeout status without the transaction id. Any request whose drain lingers, transactional or not, can fall into the fixed response, broadening impact past the targeted shape. Sequencing cleanliness costs truthful content where truth matters most.',
'Hold peer delivery behind request-scope drainage and cap the wait with a fixed dispatch deadline; prompt drains deliver the handler answer, expired waits deliver a fixed 408 or 500 in place of the handler truthful unknown:commit message. The response-after-drain invariant survives while the meaningful commit-unknown content dies exactly on the overrun route it serves, handing the client an opaque timeout without reconciliation identity. All lingering drains across every request kind become fixed-status candidates, spreading consequences beyond transactions. Invariant tidiness is bought with dishonest-by-omission answers on the decisive path.',
],
},
},
'background_drain': {
'instructions': [
'How should the detached request-scope drain be observed? Judge supervisor visibility of cleanup failures, consistency with owned-settlement silence, and report noise.',
'Choose the observation posture for background drains. Weigh surfacing cleanup faults, matching the quiet-owned-settlement rule, and signal value per report.',
'Decide what background drainage reports. Favor visible cleanup failures, harmony with silent owned settlement, and minimal noisy output.',
],
'criteria': {
'observed_reported': [
'Route background drain failures into the redacted request reporter exactly as serveOuter does for inline drains today: a failed scoped-resource close still delivers one correlated record under the request correlation and source, while clean drains and ordinary late settlements stay silent. Supervisors keep the visibility they have now, since the same failures report identically before and after the timing change; only the delivery moment shifts behind the response. Owned-until-settlement quiet for healthy work is preserved because reporting triggers on drain failure, never on mere lateness. Same reports, same silence, later instant.',
'Deliver detached drain faults to the redacted request report on the current serveOuter pattern: broken scoped-resource closes produce a single correlated entry with request correlation plus source, and healthy drains with routine late landings emit nothing. Operator visibility matches today failure for failure because identical faults yield identical records, differing solely in arriving after the peer is answered. The hush over sound owned settlement continues since lateness alone never reports, only cleanup failure does. Equivalent coverage and equivalent quiet at a later time.',
'Keep background drain failures on the redacted request-report path used by serveOuter for inline drains: faulty scoped closes emit one correlated record naming request correlation and source, while successful drains and normal late settlements remain mute. Supervision sees what it sees today, fault for fault, with only the emission point moving past peer delivery. Silence around healthy owned completion endures because reports fire on cleanup faults rather than on tardiness itself. Identical reporting and identical reticence, shifted later.',
],
'silent_owned': [
'Swallow background drain failures silently, extending the owned-until-settlement quiet to cleanup faults once the peer is answered. This keeps post-response output minimal but blinds the supervisor: a scoped-resource close that fails after peer release vanishes without a correlated record, where the same failure reports today. Failures that operators currently diagnose (failed closes, cleanup marks) would need rediscovery through side effects. Minimal noise trades against lost cleanup-failure visibility on exactly the overrun paths most likely to need it.',
'Discard detached drain faults without reporting, stretching owned-settlement silence over post-response cleanup failures. Post-answer logs stay empty, yet supervision loses sight: scoped-close faults behind an answered peer leave no correlated trace though they report in the present inline design. Operator diagnosis of failed closes and cleanup marks degrades into inference from secondary symptoms. Quietest output costs the cleanup-failure signal on the overrun routes where diagnosis matters most.',
'Let background drain failures pass unreported, applying owned-settlement hush to cleanup faults after the client is answered. Nothing further is emitted post-response, but supervisors go blind: failed scoped closes behind a delivered answer produce zero correlated records despite reporting under the current inline drain. Faults now traceable through reports would surface only via downstream effects. Least noise means least cleanup-failure insight on precisely the paths likeliest to fault.',
],
},
},
'revocation_timing': {
'instructions': [
'When should the served request snapshot be revoked? Judge token-lifetime tightness, ordering with scoped-resource closure, and the usable-token window after handler return.',
'Choose snapshot revocation timing under detached drains. Weigh how long the token stays valid, interplay with resource shutdown, and exposure after the handler ends.',
'Decide the revocation point for request snapshots. Favor short token life, safe close ordering, and no usable token lingering past its need.',
],
'criteria': {
'revoke_after_drain': [
'Keep revokeRequest after drainage completes, as today: the snapshot token stays technically valid through the background drain, but the handler already returned so no user code can reach it, and scoped body readers still close inside the drain before revocation exactly as now. Close-before-revoke ordering is preserved bit for bit, and the only change is the longer valid window behind an answered peer, which is documented honestly. Minimal ordering churn with a precisely characterized token window nobody can exploit.',
'Retain post-drainage revocation on the present pattern: snapshot tokens remain nominally live across background draining, yet handler completion precedes it so no authored code can touch them, while scoped body readers shut within the drain ahead of revocation without alteration. The close-then-revoke sequence survives verbatim; solely the nominal validity span lengthens behind a served client, honestly recorded. Least sequencing disturbance with an exactly described unreachable window.',
'Preserve revocation trailing drainage exactly as now: tokens keep nominal validity during the detached drain while handler exit beforehand removes every user-code path to them, and scoped readers close inside drainage before revocation just like today. Close-before-revoke order stays identical; the nominal window merely extends past peer delivery and is documented as such. Smallest ordering delta with a fully specified unusable span.',
],
'revoke_at_response': [
'Revoke the snapshot token at peer release before the background drain runs: token lifetime ends with the response, the tightest possible window, while scoped body readers still close inside the drain without needing the token. This reorders revoke ahead of resource closure, inverting the current close-before-revoke sequence on every request, and any close path that consults the snapshot would observe revocation first. Tightest lifetime trades against a global ordering inversion plus a new revoked-during-close state for all drains to tolerate.',
'Withdraw the snapshot token when the peer is answered, ahead of detached drainage: validity stops at response for the narrowest window, and scoped readers shut inside the drain tokenlessly. The price is a universal sequence flip to revoke-before-close across all requests, with every close routine now running under prior revocation and any snapshot-consulting path seeing the revoked state. Narrowest lifetime costs a fleet-wide ordering reversal and a fresh revoked-mid-close condition.',
'Retire snapshot tokens at response delivery before background draining starts: the token dies with the answer for minimal exposure, while drain-internal reader closes proceed without it. Ordering flips everywhere to revocation-before-closure, and close logic that references the snapshot meets revoked state instead of live state on each request. Briefest validity is paid for with a global inversion and a novel revoked-during-drain posture every close must absorb.',
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
    'review': 'Manually checked equal facts, constraints and alternatives across all three requests before sending. Each of the 3 state fields, 3 instructions and 6 option descriptions has a distinct complete phrasing (12 fields). Task IDs (E-TIMEOUT R2, W5-L1/L2/L3/L6/L7, E04, E09), file paths (runtime/platform/server.ts, runtime/platform/sql/transaction.ts, runtime/owner-core.ts, runtime/transport), identifiers (serveNative, withScope, withTransaction, raceBoundary, guardCallback, RequestScope, revokeRequest, serveOuter, shutdownMs, commit_unknown, unknown:commit, sql::commit_unknown), numeric facts (150ms bound, 155ms handler, 2009ms peer, 2s stall, 1500/1800/8000 pins), and code spellings are intentionally stable. No response or favored recommendation appears in any later request state.',
    'same_facts': [
        'Can agent language; zero external users; no compat for syntax/ABI/spellings/TS layouts/goldens; E-TIMEOUT R2 releases at-budget peer for overrun tx; W5-L3 handler 155ms vs peer 2009ms at 150ms bound, drain gates Response on callback settlement; hung callback would hold peer past any bound; user: client response bounded too, internal detection insufficient, timeout honest never implying cancelled/rolled back; L3 pins (handler <1500ms, peer >=1800ms <8000ms) flip to bounded assertions',
        'serveNative per request: ambient RequestScope (budget/signal/collector) + owner withScope drained before Response; withTransaction guarded callback task in tx inner scope nested in request scope so drain waits past raceBoundary commit_unknown; pool leases off pool resource outside scope so L1/L2/L6/L7 release at bound; only inner groups+callbacks gate drain; stop expires scopes shutdown + closes sessions + server close under shutdownMs with overdue owned for supervisor; wait same cap; E04 honesty (no rollback/non-start claim except never-started phase, commit_unknown never retried reread reconcile, no timer lease revoke, frozen markers, no post-disposal use); revokeRequest after drainage; serveOuter drain failures to redacted report; editable server dispatch + runtime/transport + tests + evidence; SQL internals incl withTransaction and owner-core.ts read-only',
        'Limits: Jev classifies, no research; advisory only, no acceptance proof; no Can code runs, no server leg starts, no calibration; timeout honest never implying cancelled/rolled back; rounds independent, no prior answer leaks; decide from supplied state',
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
