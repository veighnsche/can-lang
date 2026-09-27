"""Three fresh Jev consultations on H08-followup bound-U design decisions.

Every explanatory state field, instruction and option description is rewritten
in full across the three rounds. Technical identifiers, model IDs, numeric
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
'contract': [
'BLK-01 requires a qualified metering profile to supply a conservative upper bound U for the complete encoded call provider input plus output consumption, covering instructions, schemas, questions, provider overhead, and applicable output limits; bytes or an estimated tokenizer count are not proof of such a bound, and the provider/model/version identity is pinned so any change invalidates qualification unless the proof explicitly covers it. H08/X-R14-1 found no profile qualifies, so W6-AI is honestly blocked: the 12/12 rejection-only fixtures with exact token sums 1156+276 prove guard behavior with zero provider I/O and are not a live pass. If trustworthy usage ever exceeds U, the ledger records the actual amount without clamping, quarantines the profile from new budgeted sends, and invalidates the budgeted result; unknown usage keeps the full hold unresolved and durable.',
'The BLK-01 accounting contract demands a conservative whole-call upper bound U from every qualified profile: provider input plus output for the full encoded request, including instructions, schemas and questions, provider overhead, and any output limits that apply. Proof excludes two things explicitly: byte counts and estimated tokenizer counts. Qualification pins one provider/model/version identity, and changing any of the three voids the qualification unless the original proof covers the change. H08 X-R14-1 analysis qualified nothing, leaving W6-AI blocked by its own definition of done; its green rejection fixtures (12/12, exact 1156 input plus 276 output scripted tokens) demonstrate fail-closed behavior without sending anything and claim no live result. A trustworthy actual above U is recorded unclamped, the profile is quarantined, and the result is invalidated; missing usage leaves the hold unresolved.',
'Under BLK-01, no profile sends budgeted traffic without a conservative upper bound U over the entire encoded call: all provider-side input and output tokens, with instructions, schemas, questions, overhead, and output limits inside the bound. Two derivations can never serve as proof: reasoning from bytes, and reasoning from an estimated tokenizer count. The qualification is pinned to one provider, model, and version triple; altering any element reopens qualification unless the proof already covers it. H08 applied this standard in X-R14-1 and qualified zero profiles, so W6-AI stands blocked as designed — the passing rejection suite (12 tests, exact scripted sums of 1156 and 276) exercises the guard with no provider contact and is fenced off from live claims. When authoritative usage tops U, the ledger keeps the true figure, quarantines that profile, and fails the budgeted result; when usage is unknown, the full reservation hold persists.',
],
'published': [
'TypeSafe published Models page (retrieved 2026-09-27) states for Jev 1.13, model ID jev-1.13.0: price $42 per Btok ($0.042 per Mtok, i.e. 4.2e-8 USD per token) charged per input token with output tokens free; context length 64k tokens per request, refined as 32k tokens for the state plus the longest question while the 64k budget covers the state plus all questions combined; rate limits 250,000 tokens per second and 1,200 requests per minute. The same documentation publishes no output-token maximum, no tokenizer, and no token-count endpoint; its error table (401, 422, 429, 529) documents no over-context status, so enforcement semantics for the 64k budget are unpublished. Rate limits carry an explicit instability warning (they can change without notice), while the price and context figures carry no such warning. Successful responses carry required usage.input_tokens and usage.output_tokens plus a model echo, and choice answers keep a fixed schema of choice, confidence, and probabilities.',
'Retrieved 2026-09-27, the TypeSafe Models page pins Jev 1.13 (jev-1.13.0) with three numbers: $42/Btok or $0.042/MTok input pricing with free output, a 64k-token per-request context length (32k of it for state plus the longest single question; 64k for state plus every question together), and throughput caps of 250k tokens/second with 1200 requests/minute. Nothing in the published docs caps output tokens, names the tokenizer, or offers a counting endpoint; the documented failures are 401, 422, 429, and 529, with no stated over-context outcome, leaving the 64k budget enforcement (reject, truncate, or honor) undocumented. Only the throughput caps are flagged as unstable without notice; price and context length are stated plainly. Every success returns integer usage.input_tokens and usage.output_tokens with the answering model ID, and choice answers always contain exactly choice, confidence, and a probability per option.',
'Public TypeSafe documentation fetched 2026-09-27 describes Jev 1.13 as model jev-1.13.0 with input-only pricing of $42 per billion ($0.042 per million, 4.2e-8 USD each) and zero output charge; a request context length of 64k tokens, of which state plus the longest question may take 32k and state plus all questions together the full 64k; and rate ceilings of 250,000 tokens per second alongside 1,200 requests per minute. The docs state no maximum output length, publish no tokenizer, expose no preflight counter, and list no over-context error among 401/422/429/529 — so what the service does past 64k input tokens is unrecorded. The dynamic-change warning attaches to the rate ceilings alone, not to price or context size. Successful calls report usage input_tokens/output_tokens as required integers and echo the model, with choice answers fixed to the choice/confidence/probabilities shape.',
],
'posture': [
'The TYPESAFE_API_KEY credential is present in the operator shell (presence verified, value never printed or stored), and this follow-up slice is authorized for roughly ten small counted live calls under $0.01 total, each recorded in a usage ledger with timestamp, endpoint, model, token counts, and purpose. Editable paths are runtime/ai/, runtime/test/ai-*, examples/native-ai/, the h08-followup evidence directory, and h08 recipe/registration updates; tools/runtime/ai-eval/ is outside this slice, so harness-schema changes must go as a handoff. The pinned wire identity is provider typesafe, model jev, version 1.13.0, request model field jev-1.13.0, endpoint v1/systemone, 30 s timeout, 65536 maxBodyBytes; the triage profile sends one choice question with five fixed categories over state of subject at most 200 chars plus body at most 2000 chars. The live gate still needs a spend cap, a pinned price, and a qualified U; this slice can resolve the price leg and narrow the U gap but cannot conjure provider publication.',
'Operator posture: TYPESAFE_API_KEY exists in the environment (checked for presence only, never emitted), and bounded live probing is approved for this slice — about ten calls, below one US cent in total, every one ledgered with time, endpoint, model, input/output tokens, and purpose, and no secrets in any artifact. This slice may edit runtime/ai/, runtime/test/ai-*, examples/native-ai/, h08-followup evidence, and the h08 recipe plus registration; the tools/runtime/ai-eval harness is out of bounds, so any schema change there becomes a follow-on handoff. Wire identity stays pinned: typesafe/jev/1.13.0 with model string jev-1.13.0 against v1/systemone, 30,000 ms timeout, 65,536-byte body cap; the worked triage profile is a single five-option choice question over subject/body state capped at 200/2000 characters. Of the live gate four inputs, credentials and bounded spend exist; price pinning is in reach now, while bound qualification still awaits provider-side grounding.',
'For this slice the operator shell holds TYPESAFE_API_KEY (presence confirmed, value withheld from all records) and roughly ten metered live calls under $0.01 are approved, each to appear in the usage ledger with timestamp, endpoint, model, token split, and purpose. Writable ground covers runtime/ai/, runtime/test/ai-*, examples/native-ai/, the h08-followup directory, and h08 recipe/registration text; tools/runtime/ai-eval/ stays read-only here, pushing harness redesign into a handoff note. The pinned call shape is provider typesafe, model jev, version 1.13.0, wire model jev-1.13.0, endpoint v1/systemone, 30 s deadline, 65536-byte response cap, and the triage workload is one five-category choice question over a 200-char subject with a 2000-char body. Credentials and bounded spend are settled for this slice; the price pin is achievable from publication, but only the provider can publish what qualification still needs.',
],
'risk': [
'A wrong U fails expensively in one direction only: the first call whose actual exceeds U overspends its reservation before quarantine stops the second, so margins trade per-call allowance cost against breach risk, and silent narrowing of the feature to its fixtures is forbidden. The fail-closed machinery is already proven green: unqualified profiles reject pre-send with missing-qualification, unknown usage retains the full hold, breaches quarantine durably, and the serial runner reconciles tokens from the ledger rather than trusting bodies. Measured fixture maxima bound the fixtures, never the feature — the next real ticket can always exceed them — so any measurement campaign is evidence toward a derivation, not a derivation itself. Jev role here is advisory classification only: these consultations judge design options from the supplied facts, perform no research, and agreement across rounds is advice rather than proof or bias removal.',
'The asymmetric cost sits on the low side of U: one actual above the bound spends real allowance before the quarantine lands, so every margin choice buys breach protection with per-call allowance, while quietly shrinking the feature down to its eval fixtures is disallowed. Proven fail-closed behavior already covers the machinery: pre-send rejection of unqualified profiles, durable full holds on unknown usage, persistent breach quarantine, and ledger-reconciled token sums in the serial runner. A maximum measured over fixtures constrains those fixtures alone — production input can surpass it at any time — hence measurements feed a derivation but never constitute one. This consultation itself is advisory: Jev classifies the stated options from the given facts without researching, and round-to-round agreement advises the engineer without proving anything or removing framing risk.',
'Exceeding the bound costs real tokens exactly once per profile: the overspending call settles before quarantine blocks its successors, so margin size exchanges allowance efficiency for breach safety, and redefining the feature as its fixture set is explicitly out of bounds. The guard machinery underneath is already proven: unqualified sends refuse before any I/O, undecoded usage keeps the whole hold, breaches quarantine the profile durably, and the runner totals usage from ledger records instead of response text. Fixture-measured peaks say nothing about future production inputs, which may be larger without warning; a measurement program therefore supports whichever derivation is chosen but cannot replace it. Jev advises only: it selects among the presented designs using the furnished facts, researches nothing, and its cross-round agreement is engineering input, not verification or debiasing proof.',
],
}

questions = {
'output_treatment': {
'instructions': [
'How should this follow-up treat the output-token side of U, given a published 64k input budget with unpublished enforcement and no published output maximum?',
'Choose the slice output-side strategy: the input budget is published but its enforcement is unknown, and no output ceiling is published at all.',
'Decide what this slice does about output tokens: input has a public budget with secret enforcement, output has neither a budget nor enforcement text.',
],
'criteria': {
'measured_allowance': [
'Adopt a measured output allowance: run the registered cases plus adversarial max-length repeats and exact-repeat determinism checks live, take the observed output maximum plus a large explicit margin as a recorded allowance with a breach-quarantine backstop, and keep the verdict unqualified pending provider-published output grounding. The input side proceeds separately toward the published 64k budget with boundary verification. This spends a few cents of bounded probing to quantify the residual gap exactly while claiming no proof the contract would reject.',
'Measure first, claim nothing: execute the frozen cases, hostile max-length states, and byte-identical repeats against the live endpoint, record the peak output plus a generous stated margin as an allowance (not a qualification) guarded by breach quarantine, and hold the profile unqualified until the provider publishes output grounding. Input-side work advances in parallel against the public 64k figure with accept/reject boundary probes. Bounded probe spend buys a precise measurement of what remains instead of a premature verdict.',
'Build the allowance from live data without promoting it to proof: sweep the registered sets, worst-case-length adversarial inputs, and determinism repeats; publish the observed output peak plus a wide margin as the recorded output allowance under quarantine backstop; leave U unqualified until provider text grounds the output side. The published 64k input budget gets its own boundary-probe verification track. A few cents of metered calls convert an open question into a measured remainder with no contract overclaim.',
],
'fixed_request_only': [
'Claim output exactness only where requests are bit-fixed: verify by repeated identical live calls that one fixed request returns identical output tokens, adopt that observed figure for fixed-request sub-profiles only, and leave variable-input output entirely open with no allowance number. The triage profile U then stays open on the output side while input-side boundary verification proceeds alone. This keeps every adopted number observationally exact but concedes that variable triage output cannot be bounded by this slice.',
'Restrict exactness to frozen bytes: re-send identical requests live, confirm the output count repeats exactly, and bless that count solely for the fixed-request shape that produced it; variable ticket text gets no output figure at all. Triage qualification then advances on the input leg only, with output acknowledged unbounded. All numbers stay exactly observed, at the cost of abandoning a triage output figure this slice.',
'Bound nothing that can vary: prove by live repetition that fixed request bytes yield a fixed output count, pin that count to fixed-request profiles alone, and publish no output number for variable triage input. Input-side 64k verification continues independently while output remains formally open. Exactness is preserved by shrinking the claim to cover only replayable requests.',
],
'defer_output': [
'Do no output-side work at all: confine this slice to input-side verification of the published 64k budget (boundary probes plus case sweep), and address output tokens only if the provider later publishes an output maximum or enforcement text. No allowance, no margin, no determinism campaign. This keeps the slice minimal and avoids spending probe budget on a side that cannot qualify yet, but leaves the output gap unmeasured.',
'Skip output entirely for now: verify the 64k input figure with accept/reject boundary probes and the case sweep, and park every output question until provider publication arrives. No measured allowance, no margin arithmetic, no repeatability study. Minimal spend and minimal surface, with the output unknown carried forward unquantified.',
'Park the output side: run only the input-track verification against the public 64k budget and defer all output measurement until the provider documents an output ceiling or its enforcement. Adopt no allowance and no margin, and run no determinism repeats. The slice stays small and the output question waits for provider text, unmeasured.',
],
},
},
'verification_protocol': {
'instructions': [
'What live-verification protocol should gate any future adoption of U for the pinned identity?',
'Choose the verification campaign that must pass before any candidate U is adopted.',
'Decide which live evidence gates a future U: which probes run, and which are skipped.',
],
'criteria': {
'boundary_plus_sweep': [
'Run boundary probes plus the full sweep: one over-budget state past 64k tokens to observe enforcement (typed rejection, truncation with truthful usage, or honored overage), one near-budget accept, all 18 registered cases, adversarial max-length repeats, and byte-identical determinism repeats — roughly half a dozen metered calls near one cent — with every usage figure reconciled from the ledger into a redacted qualification artifact. Skips nothing; distinguishes enforcement semantics instead of assuming them.',
'Probe the boundary and sweep the sets: send one state beyond the 64k budget to learn the real over-limit behavior (rejection, truncation with honest usage, or silent overage), one just under it, the full 18 frozen cases, hostile full-length repeats, and exact replays for determinism — about six ledgered calls around a cent — recording only hashes, lengths, and usage. Full coverage at minimal cost, with enforcement observed rather than presumed.',
'Verify edge and middle together: an over-64k probe reveals the true limit behavior (refusal, truncation with accurate usage, or acceptance), a near-limit probe confirms admission below it, then the 18-case sweep, worst-case repeats, and identical-byte replays complete the picture for roughly six calls under a cent, all usage ledger-reconciled and redacted. No gap is left to assumption.',
],
'sweep_only': [
'Sweep the registered cases only: run the 18 frozen cases live with margin over the observed peak and skip boundary probing, avoiding large-probe cost and unknown over-limit provider behavior. Cheaper and simpler, but enforcement semantics stay assumed and the 64k figure stays unverified at its edge.',
'Cover the middle, skip the edge: execute the 18-case sets live, margin the peak, and run no near-limit or over-limit probes, sidestepping big-call spend and uncharted provider reactions. Smaller and safer to run, yet the budget boundary itself remains untested.',
'Sets without boundaries: verify the frozen 18 live with a margined peak and omit every limit-edge probe, keeping spend tiny and provider behavior predictable. The case range is covered while cap enforcement remains a guess.',
],
'boundary_only': [
'Probe only the boundary: run the over-64k enforcement probe plus a near-limit accept and skip the full 18-case sweep, since the registration own live run will sweep the cases later anyway. Minimal calls that answer the enforcement question first; case-level figures wait for the gated run.',
'Test the edge, defer the sweep: one over-budget call plus one near-budget call settle enforcement semantics now, while the 18-case measurement rides on the future qualification run instead of this slice. Fewest calls to the key unknown, with routine coverage postponed.',
'Edges now, sets later: establish over-limit and near-limit behavior with two probes and leave the frozen-case sweep to the eventual gated evaluation. Answers enforcement with minimum spend while case evidence accumulates where it will be used.',
],
},
},
'price_pin': {
'instructions': [
'How should this slice pin the published input-only price into the registration single per-token price field?',
'Choose how the input-priced, output-free publication maps onto one per-token gate price.',
'Decide the price-table pin: one gate field must carry an input-only published tariff.',
],
'criteria': {
'conservative_single': [
'Pin the input price directly: record 4.2e-8 USD per token ($0.042/MTok) with source URL and retrieval date in the registration usdPerToken field, so the gate charges settled input-plus-output tokens at the input rate — a documented conservative over-charge since real output is free — which stops runs early and can never overspend the cap. The live gate price leg unblocks mechanically with zero harness changes, and the overstatement direction is safe.',
'Adopt the input tariff as the single price: write 4.2e-8 USD/token ($0.042 per million input tokens) plus provenance into usdPerToken, accepting that settled totals (input plus output) billed at the input rate overstate true spend because output costs nothing. Runs halt before the real cap instead of after it, the price gate opens without touching harness code, and the error favors safety.',
'Use one safe-direction price: pin 4.2e-8 USD per token with its source and date, letting the gate bill every settled token at the input rate even though output is actually free. The recorded over-charge is conservative by construction — spend tracking reaches the cap early — unblocking the price leg with no schema change.',
],
'split_table_handoff': [
'Record the split exactly and hand off the arithmetic: file {input 4.2e-8, output 0} USD per token in follow-up evidence with source and date, note that exact split charging needs a tools/harness change outside this slice, and leave the gate price-blocked until that handoff lands. Numbers stay exact, but the price leg remains closed for now.',
'Pin precisely, wire later: preserve the true input/output tariff pair (4.2e-8 and 0) in evidence with provenance, defer exact per-side charging to a harness extension this slice cannot make, and keep the gate requirement unsatisfied meanwhile. Exactness now, gate progress later.',
'Split on paper, single in code: document the real two-rate tariff with its source in the follow-up record, route exact split billing to a future harness slice, and accept that the live gate stays price-blocked until then. The pin is faithful while the gate waits.',
],
'no_pin': [
'Pin nothing: treat the published tariff as a billing claim no API response can confirm, keep the price table empty, and leave the price leg blocked until an operator-supplied sheet or invoice evidence arrives. Avoids blessing an unverifiable number but discards a public, checkable publication and keeps live evaluation fully gated.',
'Decline the pin: hold that web-page pricing is not verifiable through the API, retain the null price table, and wait for operator billing evidence before unblocking the price leg. Refuses any unverified figure at the cost of ignoring the provider own posted tariff.',
'Stay unpinned: regard published prices as unconfirmed until billing paperwork corroborates them, leave the table null, and keep live runs price-blocked. Maximum skepticism, but the gate never opens on publication alone.',
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
        'model': 'jev-1.13.0',
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
    'review': 'Manually checked equal facts, constraints and alternatives across all three requests before sending. Each of the 4 state fields, 3 instructions and 9 option descriptions has a distinct complete phrasing (48 fields). Model ID jev-1.13.0, endpoint v1/systemone, env name TYPESAFE_API_KEY, file paths, requirement IDs (BLK-01/X-R14-1/H08/W6-AI), token sums 1156+276, prices $42/$0.042/4.2e-8, budgets 64k/32k, timeouts, byte caps, char caps, HTTP statuses, and code spellings are intentionally stable. No response or favored recommendation appears in any later request state.',
    'same_facts': [
        'BLK-01 conservative whole-call bound U incl. instructions/schemas/questions/overhead/output limits; bytes and estimated tokenizer counts excluded as proof; pinned provider/model/version, change voids qualification; H08/X-R14-1 qualified nothing so W6-AI blocked; 12/12 rejection fixtures with exact 1156+276 scripted sums prove fail-closed with zero I/O, not a live pass; over-U actual records unclamped + quarantines + invalidates; unknown usage keeps full hold',
        'TypeSafe Models page retrieved 2026-09-27: Jev 1.13 jev-1.13.0, $42/Btok = $0.042/MTok = 4.2e-8 USD/token input-only with free output, 64k tokens/request context (32k state+longest question, 64k state+all questions), 250k tok/s + 1200 req/min rate limits; no output max/tokenizer/count endpoint published; errors 401/422/429/529 with no over-context status so 64k enforcement unpublished; dynamic-change warning on rate limits only; success carries required usage.input_tokens/usage.output_tokens + model echo; choice answers fixed choice/confidence/probabilities shape',
        'TYPESAFE_API_KEY present (presence only, never printed); ~10 bounded live calls under $0.01 authorized, each ledgered with time/endpoint/model/tokens/purpose, no secrets; editable runtime/ai/, runtime/test/ai-*, examples/native-ai/, h08-followup/, h08 recipe+registration; tools/runtime/ai-eval/ out of slice so harness schema changes are handoff; pinned wire typesafe/jev/1.13.0 model field jev-1.13.0 endpoint v1/systemone 30s timeout 65536 maxBodyBytes; triage one 5-category choice over subject<=200/body<=2000 chars; gate still needs spend cap + price + qualified U; slice can resolve price and narrow U gap only',
        'U failure asymmetric: first over-U call overspends before quarantine; margins trade allowance cost vs breach risk; narrowing feature to fixtures forbidden; proven fail-closed: unqualified pre-send missing-qualification rejection, unknown-usage full holds, durable breach quarantine, ledger-reconciled serial sums; fixture maxima bound fixtures not feature; Jev advisory classification only, no research, agreement is advice not proof or bias removal',
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
        decoded = json.loads(body)
        print(json.dumps({'request': i, 'model': decoded.get('model'),
                          'usage': decoded.get('usage'),
                          'answers': {k: {'choice': v.get('choice'),
                                          'confidence': v.get('confidence')}
                                      for k, v in decoded.get('answers', {}).items()}}))
