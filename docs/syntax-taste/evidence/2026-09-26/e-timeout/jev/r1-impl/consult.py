"""Three fresh Jev consultations on E-TIMEOUT R1 implementation details.

The main R1 consultation (race_and_own, layered, reuse_service_error) is
settled and NOT relitigated here: every native S3 await races the E04
cancel-absent raceBoundary with source s3, bounds layer
min(deadline_ms remainder, trailing boundMs, ambient budget remainder,
adapter ceiling), and expiry lowers to s3::service_error with code
timeout. These two questions resolve the only open implementation
points: caller-held upload-handle semantics after an op expiry, and
whether explicit discard_upload cleanup races. Every explanatory state
field, instruction and option description is rewritten in full across
the three rounds. Technical identifiers, task IDs, numeric facts and
code spellings stay exact. No prior Jev answer appears in any request.
Usage: python3 consult.py (generate) ; python3 consult.py --send (3 calls).
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
'Can is an agent-oriented programming language with zero external users, so no backwards compatibility is owed to old syntax, ABI, spellings, generated TypeScript layouts, or goldens. E-TIMEOUT R1 implements the retained X-R15-3 requirement: every hung S3 await is bounded through the E04 cancel-absent raceBoundary with source s3, bounds layer min(deadline_ms remainder, trailing boundMs, ambient budget remainder, adapter ceiling), and expiry lowers to s3::service_error with code timeout. That mechanism is settled. This consultation resolves only the two open implementation points: what happens to a caller-held upload handle after upload_write or upload_finish expiry, and whether explicit discard_upload cleanup races. Decide from the supplied facts without asking the user.',
'Judge implementation choices for Can, a programming language whose consumers are coding agents rather than people. Nobody outside the project uses it, which removes any duty to preserve earlier grammars, binary interfaces, source spellings, emitted TypeScript arrangements, or recorded goldens. Work item E-TIMEOUT R1 carries the retained X-R15-3 acceptance with its mechanism already fixed: each S3 native await runs under the E04 cancel-absent raceBoundary sourced s3, each per-await bound takes the minimum across the deadline_ms remainder, trailing boundMs, ambient budget remainder, plus the adapter ceiling, and each expiry answers s3::service_error carrying code timeout. Only two application points stay open: the lifecycle of a caller-owned upload handle once upload_write or upload_finish times out, and whether the explicit discard_upload scrub takes part in racing. Reach every selection below from this material alone, posing no question to the user.',
'This choice concerns S3 upload-handle and cleanup-race details inside bounded S3 waiting for a language built for AI coding agents, where dependable behavior outranks every other goal. The project has no outside adopters, so keeping prior syntax, interfaces, spellings, generated file shapes, or goldens carries no weight. E-TIMEOUT R1 must bound each hung S3 await, and the bounding design stands decided: cancel-absent raceBoundary races with the s3 source, per-await bounds combining the deadline_ms remainder with trailing boundMs, the ambient budget remainder, and one adapter ceiling under min(), and expiry expressed as s3::service_error with code timeout. What remains is threading that design through caller-held upload boxes and the discard_upload scrub. Make each call from the evidence given here, without further questions.',
],
'mechanism': [
'UploadBox in runtime/platform/s3.ts is a caller-held token over {client, key, type, partSize, sink, state} with states open, finished, discarded and a lazily created NetworkSink writer. uploadWrite awaits sink.write plus sink.flush; uploadFinish awaits sink.end plus a metadata stat; after R1 each of those awaits races and expiry answers s3::service_error with code timeout while the native stays owned until settlement. The pinned Bun natives admit no abort: concurrent sink.write interplay was never probed, end() waits on hung same-sink state (E07 H2 shows end(Error) stuck on a hung flush), an un-ended sink pins the event loop from creation, and only a settled end releases it. retire() marks the box discarded and runs the destructive scrub (end then delete the key); scope drain retires abandoned open handles; discardUpload runs retire and answers upload_closed for non-open boxes. E08 D3 pins that discarded handles reject upload_write, upload_finish, and discard_upload reuse with upload_closed carrying state discarded. Scope-drain retire cannot race: drain has no caller to receive a timeout and reports only through cleanupFailed, so it always runs the scrub to completion. Emitted S3 calls keep their current arity; this slice touches runtime/platform/s3.ts plus tests plus evidence only.',
'The upload handle the caller keeps is an UploadBox holding client, key, type, partSize, a lazily built sink, and a state of open, finished, or discarded. Its write path awaits sink.write and sink.flush and its finish path awaits sink.end with a stat call; under R1 every one of those promises races the boundary, and expiry yields s3::service_error with code timeout while owned work runs on. Live fault-tap evidence fixes the native facts: no abort entry point exists, two overlapping sink.write calls have no qualified behavior, end() blocks against stuck state on its own sink (E07 H2), a writer dropped before ending wedges the loop from creation, and release arrives solely through a settled end. Cleanup flows through retire(), which poisons the state to discarded and then scrubs destructively with end followed by key deletion; abandoned handles retire at scope drain, and discardUpload answers upload_closed whenever the box is not open. The E08 D3 leg locks the reuse rule: discarded boxes refuse upload_write, upload_finish, and discard_upload with upload_closed naming state discarded. Retire during scope drain is unconditionally unraced, because drain owns no timeout channel back to any caller and surfaces trouble only as cleanupFailed. Emitted call shapes stay frozen; only runtime/platform/s3.ts, tests, and evidence move.',
'Caller code drives multipart work through an UploadBox token bundling the client, key, content type, part size, an on-demand sink, and lifecycle state across open, finished, and discarded. uploadWrite suspends on sink.write and sink.flush, uploadFinish on sink.end plus stat, and R1 wraps each suspension in the cancel-absent race so expiry reports s3::service_error with code timeout while the native remains owned through settlement. Measured native behavior admits no cancellation: simultaneous sink.write calls were never qualified, end() suspends on its own hung sink (E07 H2 measured end(Error) waiting on a frozen flush), writers abandoned un-ended pin the event loop from birth, and a settled end is the single release. The retire() routine flips state to discarded and applies the destructive end-then-delete scrub; scope drain retires orphaned open boxes, and discardUpload maps non-open boxes to upload_closed. E08 D3 freezes reuse semantics: a discarded handle rejects upload_write, upload_finish, and discard_upload with upload_closed and state discarded. Drain-time retire never races under any option here, since drain returns nothing to a caller and can only flag cleanupFailed. Emitted arities do not change; the edit surface is runtime/platform/s3.ts with tests and evidence.',
],
'limits': [
'Jev is a classifier, not a researcher: it supplies typed choices with probabilities, never prose reasons or new facts. Its distributions are advisory only, never acceptance proof; these calls execute no Can code, start no MinIO leg, and calibrate nothing. Do not relitigate the settled R1 mechanism (race_and_own, layered bounds, reuse_service_error): decide only the two implementation questions below. A timeout must report uncertainty honestly and never imply the write was cancelled or rolled back. Treat each round independently: no request carries any prior answer. Decide every question below from the supplied state alone.',
'The model here classifies rather than investigates: answers are typed selections plus distributions, without rationale text or fresh evidence. Such scores guide engineering judgment but prove nothing; this round runs no compiler, no runtime, no MinIO harness, and no live leg. The settled R1 mechanism stands outside this consultation, so weigh only the upload-handle and discard-race questions as posed. Any expiry outcome must stay honest about uncertainty and must not suggest cancellation or rollback occurred. Rounds stand alone with no earlier verdict leaking into later state. Ground each answer strictly in the material provided.',
'Expect no research from this consultation: the system returns a chosen option with a probability spread, not explanations or discoveries. Those numbers advise the design and certify no behavior; nothing here executes Can programs, touches MinIO, or measures domain accuracy. Only the two questions below are open; the race_and_own mechanism, layered bound sources, and service_error vocabulary decided earlier are fixed context, not alternatives. Every timeout report must confess uncertainty and never hint that the write stopped or rewound. Each of the three rounds is self-contained; later requests contain no trace of earlier outcomes. Answer only from the context handed to this call.',
],
}

questions = {
'upload_expiry': {
'instructions': [
'After upload_write or upload_finish race expiry, what should happen to the caller-held upload box? Judge honesty about the unknown write, protection against unqualified native interplay, and convergence of stranded state.',
'Choose the upload-box lifecycle once an upload op times out. Weigh truthful unknown-write handling, safety against unprobed concurrent sink behavior, and reliable orphan cleanup.',
'Decide caller-handle semantics after an upload await expires. Favor honest uncertainty, no unqualified simultaneous-native use, and orphan convergence through proven cleanup.',
],
'criteria': {
'poison_discard': [
'On upload op expiry, immediately mark the box discarded while keeping the sink reference, and attach the standard destructive scrub (end then delete the key) to the hung native settlement so storage converges through qualified machinery. Further caller operations on the handle answer upload_closed with state discarded, exactly the E08 D3 reuse shape. No caller code can build on unknown-write state, no retry can interleave a second sink.write beside the hung one, and no later finish or discard can wait on the same stuck state. The price is losing a handle that might have settled well: the caller must beginUpload again, and a deferred scrub deletes a write that may have landed intact — destructive, consistent with the remedy destructive posture that already completes-then-deletes partial work.',
'When an upload await expires, poison the handle at once by flipping its state to discarded, retain the sink for later cleanup, and run the established destructive scrub of end followed by key deletion once the lingering native settles. Every later caller operation on that token, write, finish, or discard, reports upload_closed naming state discarded, matching the qualified E08 D3 contract. Callers are structurally barred from extending unknown state, from racing a fresh write against the stuck one, and from parking a finish or discard behind the same frozen sink. The cost falls on handles that would have recovered: the caller reopens the upload from scratch, and the deferred scrub removes bytes that may have arrived whole — a destructive outcome in line with a remedy that already destroys partial uploads rather than preserving them.',
'At upload expiry, transition the box to discarded without delay, hold its sink for deferred cleanup, and execute the proven end-then-delete scrub after the hung native lands. Subsequent use of the token in any operation returns upload_closed with state discarded, the exact shape E08 D3 pins for dead handles. Unknown-write state becomes unreachable to the caller, second-write concurrency against the hung native becomes impossible, and finish or discard can never stall on the wedged sink. The tradeoff charges recovered handles: callers start over with beginUpload, and scrubbing erases a write that might have completed correctly — destructive convergence of the same kind the remedy already applies to partial keys.',
],
'leave_open': [
'On upload op expiry, leave the box open with no scrub attached: the hung native settles naturally into the still-open handle and the caller keeps full agency to retry the operation, finish, or discard. A transient stall costs only the timeout outcome rather than the handle, and no deferred scrub can delete a write that landed well. The price is unqualified interplay and back-door waits: a retry interleaves a second sink.write beside the hung one with no probed behavior, a finish or discard during the hang waits on the same stuck state (discard has no bound of its own), and a finish after a timed-out write can complete possibly-duplicated bytes, converging state the timeout outcome honestly called unknown.',
'When an upload await expires, keep the handle open and attach no cleanup: the stuck native resolves into the live box on its own schedule while the caller remains free to repeat the call, complete the upload, or throw it away. Brief stalls forfeit nothing but the single timeout answer, and a healthy write can never be destroyed by cleanup the caller never asked for. The cost is unverified concurrency plus unbounded cleanup: retrying stacks another sink.write onto the frozen one outside all qualification, finishing or discarding mid-hang parks behind identical stuck state with discard itself unbounded, and completing after expiry can seal bytes whose fate the timeout already declared unknown.',
'At upload expiry, preserve the open box and schedule nothing: the lingering native lands in the live handle whenever the service answers, and callers choose freely between retry, finish, and discard. Short stalls end at the timeout outcome with the upload intact, and well-landed bytes face no surprise deletion. The tradeoff accepts unprobed native overlap and hidden waits: a retry overlaps sink.write calls in a shape E07 never measured, finish or discard issued during the stall block on the wedged sink while discard awaits no bound, and a finish can ratify duplicated content after the timeout honestly reported uncertainty.',
],
},
},
'discard_race': {
'instructions': [
'Should the explicit discard_upload scrub awaits race the layered bound like every other S3 await? Judge uniformity of the bound, honesty of cleanup reporting, and fit with the owner model. Scope-drain retire stays unraced under both options.',
'Choose whether explicit discard_upload cleanup takes part in per-await racing. Weigh bound-everywhere uniformity, truthful cleanup outcomes, and compatibility with drain-time retire that cannot race.',
'Decide if discard_upload scrubs race or run to completion. Favor complete bounds, honest cleanup records, and coherence with the unraced scope-drain path shared by both options.',
],
'criteria': {
'race_like_others': [
'Race the discard_upload scrub awaits (end, then key delete) against the layered bound like all sibling awaits: expiry answers s3::service_error with code timeout naming discard_upload while the scrub continues owned in the background to convergence, with retire made single-flight so a second discard observes the pending scrub instead of starting a rival one. Every await the adapter issues is then bounded with no cleanup exception, and a hung service cannot hold the explicit-discard caller past the bound. The price is background convergence the owner model never observes: the returned outcome no longer reports what cleanup did, a second discard during the pending scrub needs its own pending-state answer, and the scrub outcome, ok or first-failed-await, lands nowhere the caller can read.',
'Give explicit discard_upload the same per-await races as every sibling operation: the end and delete awaits race the layered bound, expiry reports s3::service_error with code timeout for discard_upload, and the interrupted scrub proceeds owned behind the answer until it converges, guarded by single-flight retire so concurrent discards share one scrub rather than doubling it. No adapter await escapes the bound, including cleanup, so a frozen service never parks the discarding caller indefinitely. The cost is unwitnessed convergence: the caller learns timeout instead of the true cleanup result, overlapping discards require a pending answer outside the current vocabulary, and the scrub final state, success or first failure, is recorded nowhere readable.',
'Apply the standard layered race to the discard_upload scrub steps of end and key delete: on expiry the caller receives s3::service_error with code timeout for discard_upload while the live scrub keeps running owned toward convergence, with retire serialized so a repeated discard joins the running scrub instead of launching another. Bounds cover the adapter uniformly with cleanup included, and hung services cannot trap explicit discard callers. The tradeoff is invisible cleanup: the timeout answer displaces the honest scrub outcome, a discard arriving mid-scrub needs a pending-state response the contract lacks, and the eventual scrub verdict, clean or first-fault, reaches no observer.',
],
'unraced_cleanup': [
'Run the explicit discard_upload scrub to completion unraced: end then key delete execute without a boundary race and discard always reports the honest converged outcome, ok or the first failed await. Cleanup stays the convergence machinery itself rather than another raced caller: no background scrub escapes owner observation, no pending-state vocabulary is needed, and the synchronous scrub legs already qualified under E08 keep their exact meaning. The price narrows the surviving indefinite wait to the explicit-cleanup path: against a hung service, discard waits with the stuck end until the service recovers, and scope drain, which shares the unraced retire, waits with it. Data operations stay fully bounded; only deliberate cleanup can still block.',
'Let explicit discard_upload finish its scrub with no race: end followed by key deletion runs straight through and the caller always gets the true converged answer, success or earliest fault. The cleanup path remains the trusted convergence engine instead of a background task nobody watches: owner observation stays complete, no new pending answer enters the vocabulary, and the E08-qualified synchronous scrub behavior is preserved bit for bit. The cost confines any remaining unbounded wait to intentional cleanup: a frozen service holds the discarding caller at the stuck end until recovery, and scope drain waits alongside on the shared retire. All data awaits keep their bounds; solely explicit discard can still park.',
'Execute the discard_upload scrub unraced to its end: the end and delete steps take no boundary race and discard reports exactly what cleanup achieved, clean success or first failure. Convergence machinery stays synchronous and fully observed instead of splitting into a timeout answer plus a hidden background scrub: no pending-state response must be invented, and the qualified E08 scrub legs mean precisely what they measured. The tradeoff keeps one deliberate path unbounded: with a wedged service the discarding caller waits on the hung end until the service returns, joined by scope drain on the same retire. Data-path awaits are bounded throughout; only explicit cleanup retains a wait.',
],
},
},
}

for name, entries in state.items():
    assert len(entries) == 3 and len(set(entries)) == 3, f"state {name} must have 3 distinct phrasings"
for qname, q in questions.items():
    assert len(q['instructions']) == 3 and len(set(q['instructions'])) == 3, f"{qname} instructions need 3 distinct phrasings"
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
    'review': 'Manually checked equal facts, constraints and alternatives across all three requests before sending. Each of the 3 state fields, 2 instructions and 4 option descriptions has a distinct complete phrasing (9 fields). Task IDs (E-TIMEOUT R1, X-R15-3, E07 H2, E08 D3, E04), file paths (runtime/platform/s3.ts), identifiers (UploadBox, uploadWrite, uploadFinish, discardUpload, retire, scrub, sink.write/flush/end, raceBoundary, source s3, s3::service_error, upload_closed, boundMs), state names (open/finished/discarded), and code spellings are intentionally stable. Settled R1 mechanism is fixed context, not an alternative. No response or favored recommendation appears in any later request state.',
    'same_facts': [
        'Can agent language; zero external users; no compat for syntax/ABI/spellings/TS layouts/goldens; E-TIMEOUT R1 retained X-R15-3 with settled mechanism (race_and_own, layered min(deadline_ms remainder, trailing boundMs, ambient budget remainder, adapter ceiling), reuse_service_error code timeout); this consult resolves only upload-box-after-expiry and discard-race implementation points',
        'UploadBox caller-held {client, key, type, partSize, lazy sink, state open/finished/discarded}; uploadWrite awaits sink.write+flush, uploadFinish sink.end+stat, each raced post-R1 with expiry service_error timeout and native owned to settlement; natives have no abort, concurrent sink.write unqualified, end() waits on hung same-sink state (E07 H2), un-ended sink pins loop from creation, only settled end releases; retire() marks discarded + destructive scrub (end then delete); scope drain retires abandoned handles; discardUpload runs retire, upload_closed for non-open; E08 D3 pins discarded-handle reuse rejections; scope-drain retire cannot race (no timeout channel, cleanupFailed only); emitted arities frozen; slice owns runtime/platform/s3.ts + tests + evidence',
        'Limits: Jev classifies, no research; advisory only, no acceptance proof; no Can code runs, no MinIO leg starts, no calibration; settled mechanism not relitigated; timeout must report uncertainty honestly, never imply cancelled/rolled back; rounds independent, no prior answer leaks; decide from supplied state',
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
