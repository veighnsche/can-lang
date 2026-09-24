"""Three freshly worded Jev consultations on the DI-06 hard case; --send calls API."""

from pathlib import Path
import difflib
import json
import os
import sys
import time
import urllib.error
import urllib.request

OUT = Path(__file__).resolve().parent

STATE = {
    'goal': [
        'Can targets AI coding agents rather than people. Correct behavior, reliable refactors, independent packages and explicit contracts dominate; total tokens per successful task are measured secondarily. There are no external users or compatibility duties. Ordinary Can operations should lower to native JavaScript/Bun. Decide the DI-06 cross-package fixture contract for a future implementation list, not present compiler code.',
        'The intended author is an AI coding agent. Prefer sound execution and diagnosable edits across separately composed libraries. Count all prompt, code, diagnostic and retry tokens through success only as a secondary objective. Human familiarity and source brevity are not goals. The language may break prior spellings; it should emit equivalent JS/Bun operations. This choice concerns fixture ownership semantics.',
        'For an agent-authored language with no outside users, assess test reliability, package isolation and stable contracts first. Task-wide token use is a lesser empirical criterion, and human ergonomics carry no weight. Existing Can syntax is not a compatibility constraint; use native Bun/JS behavior where possible. Recommend a DI-06 planning disposition after the private-clock probe.',
    ],
    'baseline': [
        'Current attached assertion roots have qualified identities and isolated FIFO queues by root, lexical table and invocation, but a helper lexical when row activates on the running root short display label. An earlier 28-root comparison found caller customer passing with a helper-owned text conversion fixture, while renaming only that caller root to renamed made it execute the real conversion and fail without a checker diagnostic. Whole-helper caller stubs survived the rename but hid helper-body mutations. Ordinary callable injection preserved the helper body for a simple converter.',
        'Can already distinguishes full assertion roots and queue occurrences, so the known flaw is foreign short-name selection, not queue allocation. In a reproduced conversion example, customer selected a helper row; renamed did not, although the helper file was untouched and static checking accepted both. A caller-local stub of the complete helper result was stable but stayed green when the helper changed. Passing a typed conversion function did run caller code and helper code together in that simple case.',
        'The existing runtime keeps per-root/table/invocation FIFO reservations and emits distinct evidence labels. Its selector nevertheless lets a library row match an unrelated caller assertion by a coincidentally equal display name. Changing only customer to renamed broke the earlier current-Can fixture case after compilation. A whole-helper stub made the assertion independent of that name yet bypassed internal logic; a typed injected converter exercised both sides when the dependency was easy to expose.',
    ],
    'hard_case': [
        'The new preregistered current-Can probe places clock::wall_millis() inside a private helper function behind public helper::stamp(int). Two ordinary app functions call stamp, and an app assertion needs a controlled 1000 result while exercising the helper suffix. Eight project variants checked and emitted; 58 attached roots ran, with only the renamed ambient case failing as predicted. The ambient helper customer row passed before rename and silently fell through afterward. A whole-helper stub passed both names, including after a suffix mutation. Changing the public stamp signature to accept callable int () emits [] now forced both ordinary call sites and the assertion call to pass a callable, moved a real clock adapter into app, and added a fake there; the caller ran real helper logic. A second design kept stamp(int), exported stamp_with_clock(int, callable), and changed only the assertion call, but published the private clock dependency as a test-facing contract. Both injection designs passed after rename; separately changing fake time or helper suffix broke their caller assertions with real-can evidence. Neither is a fixture-supplied completion. No exported-scenario implementation or agent-token comparison exists.',
        'A harder executed example has helper::stamp(int) reading clock::wall_millis() in a private routine, then converting and suffixing the time; two production-style app functions call the public entry. All eight designed project forms passed checking and emission, and 58 roots were executed. Only ambient selection failed its renamed caller assertion: customer got a helper fixture, renamed got the actual clock. The caller whole-helper row remained green after changing the helper suffix, so it offered no integration coverage. For direct injection, stamp gained a typed now callable and all three app calls changed; app also defined a real-clock adapter and fake. For the alternate test entry, the two normal calls stayed, while helper exported stamp_with_clock and the test called it with a fake. The fake and suffix were each mutated: in either injected design the integrated assertion failed, carrying real-can. The test entry reveals an internal effect in the package API. No scenario-link prototype or full-task token trial has run.',
        'A library-private clock read now makes the API tradeoff concrete. Present Can accepted all eight hard-case programs and executed 58 roots. A foreign customer row supplied time; after a caller-only label rename the same helper instead used wall time, failing without an ownership error. A stub at helper::stamp(7) worked across the rename but still passed when the helper suffix was altered. A public callable parameter let the caller supply time and check the real suffix, at the price of modifying two ordinary uses plus the test use and authoring a clock adapter in app. Exporting a second stamp_with_clock function preserved ordinary stamp(int) calls but exposed a test-only clock hook in the helper export list. For both, fake-clock and suffix perturbations independently failed the real-can caller path. No new-link code or agent efficiency data has been produced.',
    ],
    'required_contract': [
        'Remove implicit foreign-root display-name activation. Keep exact target/argument checking, root/table/invocation FIFO, recursion/callable/concurrent isolation and distinct real-can, supplied-completion and raw-provider-fixture labels. Deliberate cross-package helper-body simulation is requested; independent helper unit tests or a complete helper stub do not count. A requested missing, inaccessible, changed or unused scenario must diagnose rather than silently execute real code. Existing package identity and protected values are handled elsewhere. Compare the smallest semantic mechanism that actually satisfies this contract; source line count does not prove agent-task efficiency.',
        'A valid rule prevents another package assertion label from accidentally selecting a helper fixture. It must preserve checked calls and values, deterministic isolated queues for repeat/nested/recursive/concurrent calls, and honest provenance labels. The product requirement includes an intentional caller-to-helper simulation path through internal code. A stubbed whole result and a separate unit root are weaker evidence. If an explicit scenario disappears or changes target, fail visibly, including unused requested bindings. Other tracks establish package identity and validation. Do not infer coding-agent cost from authored length.',
        'Guarantee ownership rather than accidental label equality, while retaining the current exact-call checks and root/table/invocation FIFO isolation, including repeated, higher-order and concurrent work. Reports must distinguish real Can execution, fixture completions and provider fixtures. The caller should deliberately exercise helper internals under controlled behavior across packages. Checked selected links cannot fall through on missing names, incompatible signatures or unused requirements. The plan may separately decide identities and owner-created types. Whole successful-task tokens need measured trials.',
    ],
}

QUESTION = [
    'After the executed private-clock hard case, which DI-06 disposition is justified for the ordered implementation plan? Judge integration coverage, public contract cost and remaining evidence gaps.',
    'What fixture-ownership route should Can schedule now given the observed label failure and both callable-injection shapes? Select the least speculative complete next disposition.',
    'Which answer best fits the cross-package simulation requirement and present measurements for helper::stamp? Choose a concrete rule or the precise still-needed trial.',
]

OPTIONS = {
    'public_parameter': [
        'Adopt same-owner lexical fixture selection and require the public helper operation itself to accept ordinary typed injectable dependencies when a caller needs internal simulation. The hard case passes with real-can helper-body coverage, but stamp(int) becomes stamp(int, callable) and every normal caller must supply time. Keep whole-helper stubs only for caller-only tests; no scenario syntax is added.',
        'Use F1 local-only rows plus a callable parameter on the functional API as the integration mechanism. The measured fake time and helper suffix both matter, and the caller rename is harmless. This route edits all three stamp calls, moves a real clock adapter to app, and treats the passed fake as real Can code rather than fixture supply. It avoids new linking machinery despite the public API distortion.',
        'Restrict lexical fixtures to their owner; simulate internals by passing dependencies through the ordinary exported stamp signature. The executed direct-injection program works and needs no fixture feature, while two unrelated app uses and the assertion gain a now argument. Report its real-can provenance and retain complete helper stubs for paths where helper internals need no integration check.',
    ],
    'test_entry': [
        'Adopt local-only lexical rows and an explicitly exported second helper entry, stamp_with_clock(int, callable), sharing the production stamp body. The two normal stamp(int) calls remain untouched; the caller test changes one call and runs the helper body with real-can evidence. This still exposes a private effect through a public test-facing API, but no new fixture-link language is required.',
        'Choose an owner-exported injection hook beside the unchanged production function. The tested stamp_with_clock path takes a fake clock and preserves body coverage after caller rename. Its cost is a new public operation whose typed parameter reveals a private clock dependency, plus owner implementation and one assertion call edit; it reuses ordinary Can semantics and no scenario routing.',
        'Keep stamp(int) as the functional API, add exported stamp_with_clock for controlled callers, and restrict ambient fixtures by owner. This current-Can variant also fails when either fake time or helper suffix changes. It avoids editing two normal app uses but publishes an internal testing dependency as a separate package contract; evidence stays real-can.',
    ],
    'exported_scenario': [
        'Adopt F2: helper exports a named scenario for stamp that owns a fixture of the private clock call; app assertion explicitly binds its particular stamp invocation to that scenario. The real stamp body runs while an internal result is supplied, without changing the functional function signature or exporting a clock callable. Compile-time checks bind owner, target, args and scenario; missing, changed or unused links fail. This is a new parser/checker/emitter/runtime feature with no executed prototype or agent-token measurement.',
        'Schedule a helper-owned exported scenario plus checked caller link. The helper publishes a named simulation contract instead of a callable clock API, and app selects it at the stamp call site independent of display labels. Exact target and signature checks and isolated FIFO routes prevent silent fallthrough. This promises internal-body fixture coverage while keeping stamp(int), but current evidence does not execute the proposed mechanism.',
        'Introduce F2 named library scenarios: a scenario declared with stamp controls the private clock result, and a caller assertion attaches the declared scenario to one invocation. Owner identity, call-site identity, argument and signature compatibility and unused selection are checked. The public production call remains stamp(int). The tradeoff is new compilation and runtime routing, since no scenario implementation has yet been run.',
    ],
    'bounded_prototype': [
        'Do not yet adopt F1 injection or F2 scenarios as the final cross-package rule. Schedule a disposable exported-scenario prototype against this exact clock case, including rename, changed helper/fake, vanished link, repeated and nested invocation, FIFO isolation and provenance. Then compare source/API/refactor effects and full successful-agent-task tokens with both executed injection forms. Meanwhile schedule the proven same-owner lexical repair and honest stub evidence independently.',
        'Reserve the final feature choice for one narrow implementation trial: build only enough helper-owned scenario linking to execute the private-clock assertion, reject missing or incompatible links, and test duplicate calls/nesting and report labels. Compare it against the now measured public-parameter and test-entry baselines, including creation/refactor/repair token cost. Apply local fixture ownership and whole-helper-stub provenance regardless.',
        'Keep F1 versus F2 unresolved pending a small scenario experiment, rather than treating an unimplemented promise as verified. The test must attach a caller invocation to a helper-exported clock scenario, survive assertion rename, fail on link drift, isolate repeated/nested calls and preserve supplied-completion provenance. Measure agent work through success beside the two current injection idioms. Repair foreign-label selection immediately as a separate definite task.',
    ],
}

def strings(value, path=''):
    if isinstance(value, str):
        yield path, value
    elif isinstance(value, dict):
        for key, item in value.items():
            yield from strings(item, f'{path}.{key}' if path else key)

def payload(index):
    return {
        'model': 'jev-latest',
        'state': {key: variants[index] for key, variants in STATE.items()},
        'questions': {'fixture_disposition': {
            'type': 'choice',
            'instructions': QUESTION[index],
            'criteria': {key: variants[index] for key, variants in OPTIONS.items()},
        }},
    }

def main():
    requests = [payload(index) for index in range(3)]
    maps = [dict(strings(item)) for item in requests]
    for left in range(3):
        for right in range(left + 1, 3):
            assert maps[left].keys() == maps[right].keys()
            repeated = [key for key in maps[left] if key not in ('model', 'questions.fixture_disposition.type') and maps[left][key] == maps[right][key]]
            assert not repeated, (left, right, repeated)
    audit = {
        'semantic_equivalence_review': 'Manually verified: all three requests state the same agent-first goal, current selector and queue facts, earlier conversion result, private-clock program and 58 roots, ambient/stub/injection outcomes, three modified calls versus one exported test entry, absent scenario prototype/token trial, shared required guarantees, and the same four alternatives. No prior Jev answer is supplied.',
        'all_explanatory_strings_pairwise_distinct': True,
        'pairwise_similarity_by_field': {
            f'{left+1}-{right+1}': {key: round(difflib.SequenceMatcher(None, maps[left][key], maps[right][key]).ratio(), 3) for key in maps[left] if key not in ('model', 'questions.fixture_disposition.type')}
            for left in range(3) for right in range(left + 1, 3)
        },
    }
    OUT.mkdir(parents=True, exist_ok=True)
    for index, item in enumerate(requests, 1):
        (OUT / f'request-{index}.json').write_text(json.dumps(item, indent=2) + '\n')
    (OUT / 'wording-audit.json').write_text(json.dumps(audit, indent=2) + '\n')
    if '--send' not in sys.argv:
        return
    key = os.environ.get('TYPESAFE_API_KEY')
    if not key:
        raise RuntimeError('TYPESAFE_API_KEY is absent')
    for index in range(1, 4):
        body = (OUT / f'request-{index}.json').read_bytes()
        request = urllib.request.Request('https://api.typesafe.ai/v1/systemone', data=body,
            headers={'Authorization': f'Bearer {key}', 'Content-Type': 'application/json'}, method='POST')
        for attempt in range(4):
            try:
                with urllib.request.urlopen(request, timeout=90) as response:
                    raw = response.read()
                    metadata = {'http_status': response.status, 'attempt': attempt + 1, 'requested_model': 'jev-latest'}
                (OUT / f'response-{index}.json').write_text(json.dumps(json.loads(raw), indent=2) + '\n')
                (OUT / f'metadata-{index}.json').write_text(json.dumps(metadata, indent=2) + '\n')
                break
            except urllib.error.HTTPError as error:
                failure = {'status': error.code, 'body': error.read().decode(errors='replace'), 'attempt': attempt + 1}
                (OUT / f'failure-{index}-{attempt+1}.json').write_text(json.dumps(failure, indent=2) + '\n')
                if error.code not in (429, 500, 502, 503, 529) or attempt == 3:
                    raise
                time.sleep(2 ** attempt)

if __name__ == '__main__':
    main()
