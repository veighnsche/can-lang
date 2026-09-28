"""Evidence-only requests for three fresh TypeSafe Score assessments.

The six ordered descriptions are this project's advisory rubric. TypeSafe supplies
its Score primitive, not an official performance standard or American grade scale.
"""

from assessment import RUBRIC_VERSION, evidence_fingerprint, validate_score_answer
from ranking import timing_resolution_issue

# Every explanatory field has three independently written, semantically equivalent
# versions. Numeric evidence and technical case/contract identifiers stay exact.
CONTEXT = (
    "Assess the present performance of the measured Can workloads. This is an advisory engineering opinion, not compliance with an accepted SLO, historical regression, or proof of product-wide quality. No usage frequencies or causal profiles are known. Respect workload size, absolute duration, variation and timing boundaries. Grade typical execution for the eligible cases together; do not average incomparable durations or reward cheap native reference rows. Missing historical baselines alone do not prevent a current-quality opinion. Treat large relative overhead separately from its absolute cost; do not claim that any excess time is recoverable. Cases outside the stated subset are unassessed. A favorable grade is restricted to this sample and is not evidence about tails, scalability or unmeasured features.",
    "Give an engineering assessment of how well the supplied Can scenarios perform now. The outcome is advice, not an established service-objective verdict, a trend assessment or a judgment of the entire product. Execution frequency and causal profiling are absent. Account for input scale, elapsed cost, spread and the exact measured boundary. Consider eligible workloads collectively without combining unlike durations or letting fast control implementations improve Can's mark. A previous run is unnecessary for this limited present-day evaluation. Keep proportional inefficiency distinct from added elapsed time, and infer no realizable savings from their difference. Omitted scenarios receive no judgment. Good results here cannot establish tail behavior, growth at other sizes or capabilities beyond these fixtures.",
    "Judge current measured workload fitness for Can as an engineering adviser. Do not present the judgment as meeting a validated SLO, detecting a change over time, or certifying general application quality. There is no frequency-of-use evidence and no profile locating causes. Weigh payload scale, time consumed, variability and measurement scope. Evaluate only the included production cases as a group, never summing unrelated timers or crediting Can for its native controls. The absence of an older measurement does not itself block this scoped opinion. Distinguish relative slowdown from absolute extra work; neither quantifies optimization savings. Excluded work remains outside the assessment. Even excellent observed performance says nothing about request tails, larger-scale growth or uncovered functionality.",
)
RUBRICS = (
    (
        "The tested work has severe avoidable-looking delays or inefficiency relative to its stated purpose; its observed cost makes this workload a major performance concern.",
        "The tested work has substantial waiting or execution overhead for its purpose; its current performance needs significant attention.",
        "The tested work is usable but its delays or inefficiency leave clear room for improvement; meaningful weaknesses temper otherwise adequate results.",
        "The tested work performs well overall for its purpose and size; remaining delay or overhead is moderate and does not dominate the measured task.",
        "The tested work performs very well: measured delay and overhead are small for the task, with no material weakness apparent in the eligible evidence.",
        "The tested work performs exceptionally well for its purpose; strong relevant evidence supports very low delay or near-reference efficiency with consistent measurements, beyond merely being acceptable.",
    ),
    (
        "Observed execution cost is a major concern: the supplied task shows extreme waiting or inefficiency that appears disproportionate to the work it performs.",
        "Performance warrants substantial improvement because the named scenarios impose considerable latency or overhead in their intended role.",
        "The scenarios remain practical to use, yet noticeable costs or inefficiencies limit them to an adequate result with evident improvement opportunities.",
        "Execution is generally good at the tested scale: some overhead or delay remains, but it is moderate relative to the measured purpose.",
        "Execution is very good for these scenarios, with low task-relative waiting and overhead and no consequential shortcoming visible in the included measurements.",
        "Evidence relevant to the actual task supports outstanding execution, showing consistently minimal waiting or efficiency close to the applicable reference rather than simple adequacy.",
    ),
    (
        "These measurements reveal a serious performance problem for the named workload: apparently disproportionate inefficiency or prolonged delays dominate its execution.",
        "The workload's intended use is burdened by considerable execution overhead or waiting, making substantial performance work appropriate.",
        "Performance is serviceable in the measured scenarios, although meaningful latency or efficiency weaknesses offer clear scope for improvement.",
        "The workload is broadly efficient and responsive for what was tested; any remaining measured delay or overhead is limited rather than predominant.",
        "The available eligible results show excellent task-relative speed and little overhead, without an important observed performance weakness.",
        "Consistent, applicable measurements demonstrate exceptional task-relative speed or efficiency approaching its relevant comparison, providing stronger support than ordinary satisfactory performance.",
    ),
)
SCOPES = {
    "compiler": (
        "Assess developer waiting for loading, checking and emitting the named project sizes. Pipeline and phase clocks overlap; do not add them. Judge developer waiting from full pipelines; cheap diagnostic phases must not offset slow pipeline results. TypeScript validation, assertion execution and publication are excluded. Use an advisory view of local build cost, without an approved compile-time budget.",
        "Judge local compilation cost for the supplied projects, covering load, check and emission. Individual phase times overlap the pipeline measurement and cannot be summed. Let complete pipelines determine the waiting assessment; many fast phases cannot compensate for a slow full build. Neither TS validation nor assertions nor publishing is timed here. No compile-speed objective has been adopted.",
        "Evaluate the delay a developer experiences for the recorded source sizes through load/check/emit. Phase timings and the complete pipeline are overlapping observations. Base the waiting grade on full builds, without allowing numerous inexpensive component timers to mask costly pipelines. Validation of emitted TS, tests and artifact publication fall outside these clocks. The opinion uses no agreed compilation target.",
    ),
    "assertions": (
        "Assess elapsed time to run complete precompiled assertion workloads through the production supervisor, including child startup. Worker counts and assertion counts differ; this is developer test-cycle cost, not time for one assertion. No formal budget exists. The summary does not establish exact assertion populations; judge fixture turnaround, not per-assertion efficiency.",
        "Judge the supervised execution of already compiled assertion sets as a developer feedback task. Process startup is included, and fixtures vary in assertion and worker counts. Whole-set duration must not be described as individual-assertion latency; there is no accepted target. Exact assertion totals are unavailable in this summary, limiting judgment to whole-fixture turnaround rather than efficiency per assertion.",
        "Evaluate developer waiting for each full assertion fixture using the real supervisor. Children starting up contribute to the measurement, while the number of assertions and workers varies. These totals are not per-assertion timings and have no established time limit. Since this summary lacks exact assertion counts, assess completion of each fixture without inferring single-assertion efficiency.",
    ),
    "artifacts": (
        "Assess local output validation, source-map processing and first or reused publication as distinct developer workflow steps. Their timers must not be summed and no end-to-end build budget is supplied.",
        "Judge the cost of the separate artifact tasks: validating output, processing source maps, and publishing with or without reuse. These observations are not additive, and there is no adopted complete-build target.",
        "Evaluate each measured artifact operation in the local development loop, including validation, source maps and new versus reused publication. Do not combine the clocks into build time; an agreed full-build limit is absent.",
    ),
    "generated": (
        "Assess efficiency of emitted Can execution using only matched native pairs as controls. Ratios compare tested endpoint contracts, not every semantic edge case, and include runtime-helper costs. Consider absolute extra milliseconds as well as ratios. Never grade native controls as Can work or infer user-visible delay from a microsecond workload. No universal slowdown cutoff is adopted.",
        "Judge the execution efficiency of compiled Can relative to the compatible handwritten controls. Matching covers the exercised result contracts, not all possible semantics; helper overhead remains included. Weigh additional elapsed milliseconds alongside multiplicative differences. Control rows must not improve the Can grade, and tiny operation times do not establish UI impact. There are no agreed ratio bands.",
        "Evaluate how efficiently the generated program runs compared with its verified native counterparts for these fixtures. Endpoint equivalence is limited to tested behavior, and runtime support is part of the measured execution. Retain both proportional and absolute extra cost. Exclude reference implementations from the subject being graded, avoid translating microseconds into product responsiveness, and impose no fixed global multiplier thresholds.",
    ),
    "runtime": (
        "Assess complete helper workloads for their stated size, including immutable updates, completion handling and ownership work. Native-named rows have not been accepted as matched controls by this rubric and are conservatively excluded; their identifiers alone do not prove semantic differences. This is helper cost, not generated-code-only cost or measured application impact.",
        "Judge the timed helper sequences at their recorded scale: collection updates, ownership operations and completion machinery. This rubric has not established native-labeled rows as comparable references, so it omits them conservatively; different names alone do not establish unequal semantics. The result concerns helper execution rather than code generation alone or established user impact.",
        "Evaluate runtime-support execution for the provided payloads, covering immutability, ownership and completions. Native variants have not been admitted as equivalent controls for this assessment and are excluded from grading; naming differences by themselves say nothing conclusive about behavioral equivalence. Restrict the conclusion to helper workload cost; neither isolated codegen cost nor product impact is measured.",
    ),
    "codecs": (
        "Assess complete validated encoding, decoding and rejection workloads at their supplied payload sizes. Native parsing lacks schema validation and is excluded as an equivalent comparator; invalid inputs exercise intended rejection. Do not penalize correctness checks simply for doing more work than a raw parser.",
        "Judge the conversion and validation workload, including successful encode/decode and expected failure paths for the stated inputs. A bare native parser performs less work and is omitted from equivalent comparisons. Necessary contract checks cannot fairly be counted as waste merely because plain parsing skips them.",
        "Evaluate the supplied data-conversion tasks with their validation and intended invalid-input handling intact. Exclude native parse controls from equivalent-reference reasoning because they omit schema checking. Required validation is part of the task, so a cheaper unchecked parser does not alone demonstrate poor implementation.",
    ),
    "startup": (
        "Assess fresh-process launch through child checks, captured output and exit. The boundary includes setup and loading; OS/Bun caches are uncontrolled. Minimal/runtime/generated and application variants perform different work. No cold-cache or time-to-interactive claim is supported.",
        "Judge complete new-process startup measurements, including harness setup, module loading, child verification, output capture and termination. Cache state was not controlled. The minimal, runtime, generated and app variants differ in work; these figures establish neither cold-start cache behavior nor UI readiness.",
        "Evaluate elapsed launch-to-exit cost with its setup, parsing/loading, child correctness work and collected output. Operating-system and Bun caching remain unmanaged. Each minimal/runtime/generated/application scenario has a different payload. Do not interpret these clocks as cold-cache results or time until a user interface becomes usable.",
    ),
    "browser": (
        "Only callback/property operations were timed, with no presentation delay. Unresolved samples cannot support a speed grade; browser responsiveness is unassessed.",
        "The timer covers control callbacks or property access and omits painting. Values beneath its resolution are unusable for grading, leaving perceived browser response unknown.",
        "Recorded intervals stop within callback or property work before visual feedback. Insufficient timing resolution prevents a performance mark and establishes no input-to-paint result.",
    ),
    "server": (
        "Assess only bounded-client batches through final response consumption, for the recorded request count and local host. Arrival-rate totals include deliberate scheduling and are excluded from speed grading. No single-request percentiles, saturation capacity, Internet latency or production SLO is inferred.",
        "Judge completion cost for the fixed-client request groups running locally, ending after response bodies are consumed. Open-arrival batch time contains programmed spacing and is omitted. The evidence does not establish per-request tails, maximum capacity, remote-network delay or production objective compliance.",
        "Evaluate the local bounded-concurrency workloads as complete request batches with their stated counts and final body consumption. Scheduled-arrival durations intentionally include pacing, so do not grade those totals. Restrict conclusions to batch execution without claiming individual latency quantiles, saturation throughput, external networking or service guarantees.",
    ),
    "io": (
        "Assess bounded local file workloads using exactly matched native-contract controls where available. Timing includes copies and cancellation appropriate to each contract; filesystem cache remains enabled and fsync/durable writes are excluded. This is local adapter efficiency, not storage-device throughput.",
        "Judge the local adapter tasks for bounded reading or writing against compatible native implementations. Their boundaries include required ownership copies or stream cancellation, keep OS caches active and omit durability synchronization. Do not infer physical disk bandwidth from this comparison.",
        "Evaluate the recorded bounded file operations relative to native controls only when contracts and timing boundaries match. Include their prescribed copying and cleanup, recognizing enabled filesystem caching and no fsync guarantee. The grade describes adapter execution rather than hardware storage performance.",
    ),
    "editor": (
        "Assess response delay through JSON-RPC transport and production language-server work, excluding initialization and visible paint. Provisional engineering goals are 100 ms for completion/hover/definition and 500 ms for formatting/rename; they are proposals, not adopted standards. Mixed diagnostics alternate cheap syntax failures with valid semantic checking and are excluded. Consider uncertainty near a proposed boundary.",
        "Judge language-server response time including wire handling and actual server computation but without startup or display presentation. Use proposed, unvalidated targets of 100 ms for lookup/completion and 500 ms for rename/format. The diagnostic aggregate mixes invalid-syntax and valid-semantic edits and must be omitted. Treat results close to either suggested limit cautiously.",
        "Evaluate the wait from client request to LSP reply, covering framing and server processing while leaving initialization and painting outside scope. Candidate goals, not accepted rules, are 100 ms for hover, definition and completion and 500 ms for formatting and rename. Do not grade combined diagnostics because quick malformed edits and semantic rechecks are interleaved. Allow for variation around these suggested limits.",
    ),
    "journeys": (
        "Assess full local compiled request workflows including decode, emitted business logic and response serialization. The invoice fixture exercises record validation and reduction, not a deployed SaaS application: database, authentication, network and UI are absent. Invalid requests follow expected rejection paths.",
        "Judge the complete in-process application fixtures from input conversion through generated logic to serialized output. Invoice work covers records and validation/reduction, without database access, authentication, external transport or display. It is not evidence for an entire SaaS deployment; rejection is correct for invalid inputs.",
        "Evaluate these locally executed compiled application journeys with decoding, business computation and encoded response inside the timer. Invoice cases test record processing and validation, including intentional invalid-input failures. Since storage services, identity checks, networking and interface work are absent, the result cannot characterize a production SaaS system.",
    ),
}
SAMPLING = (
    "Each case median summarizes independent process-trial batch medians, after the recorded warmups. Min, max and MAD describe those trial medians, not individual request tails. Raw sample counts indicate measured batches per process. Trials are warm within their workload; natural garbage collection is included and caches are not flushed. Startup retains its separately stated fresh-process boundary.",
    "Case statistics aggregate one batch median from each independently launched trial after warmup removal. Their extrema and deviation reflect process medians rather than tail response times. The batch totals are provided per trial. Workload state is warm, collection occurs naturally and no cache reset is performed; startup still uses its documented new-child timing scope.",
    "The reported center comes from independent process medians over measured batches, excluding warmup work. Range and MAD concern these trial-level centers, not percentile latency of requests. Per-process sample counts specify batch populations. Measurements allow ordinary garbage collection and warm workload caches without flushing, while launch tests keep their stated fresh-process interval.",
)
QUESTION = (
    "For the `{suite}` slice, assess the performance fitness described in `slices.{suite}.scope` using only its eligible measurements and matched comparisons. Select a position on the six-level advisory rubric. Different tasks do different work: use task-relative engineering judgment, not raw duration ranking. The grade concerns the included workloads collectively, not the fastest control or all possible features.",
    "Rate current workload performance for `{suite}` according to `slices.{suite}.scope`, its supplied case evidence and valid reference pairs. Apply the six ordered advisory descriptions. Evaluate cost relative to the task instead of ordering unlike tasks by elapsed time. Judge the included production set, not its quick control implementations or capabilities outside the fixtures.",
    "Using the eligible cases and compatible controls under `slices.{suite}`, determine how well `{suite}` performs within its stated scope. Return a level on the ordered engineering-opinion scale. Compare effort with the purpose it serves rather than sorting arbitrary durations. Your assessment covers the measured production workloads as a group, without extrapolating to untested functionality or grading their references.",
)
BASIS = {suite: texts[0] for suite, texts in SCOPES.items()}


def excluded_reason(key, row):
    suite = key.split('/')[0]
    if timing_resolution_issue(row):
        return "unresolved-timing"
    if row.get('ranking_issues'):
        return "measurement-qualification"
    if suite == 'editor' and key.endswith('.diagnostics'):
        return "mixed-diagnostic-workload"
    if suite == 'server' and key.endswith('.arrival-rate'):
        return "deliberately-paced-batch"
    if suite == 'runtime' and key.endswith('.native'):
        return "different-reference-contract"
    if suite == 'codecs' and 'native' in key:
        return "unvalidated-reference"
    return None


def role(key):
    return "reference" if (key.endswith(('.native', '.native-adapter')) or '/native-contract.' in key) else "subject"


def eligible_cases(data, suite):
    return {key: row for key, row in data['cases'].items()
            if key.startswith(suite + '/') and not excluded_reason(key, row)}


def pairs(rows):
    result = []
    for key, row in rows.items():
        if key.startswith('generated/') and key.endswith('.can'):
            base = key[:-4]
            other = next((k for k in (base + '.native', base + '.native-adapter') if k in rows), None)
        elif key.startswith('io/runtime.'):
            other = key.replace('io/runtime.', 'io/native-contract.', 1)
        else:
            continue
        reference = rows.get(other)
        if (reference is None or any(row.get(field) != reference.get(field)
                for field in ('unit', 'parameters', 'timing_scope', 'iterations_per_sample'))):
            continue
        a, b = row['distribution']['median'], reference['distribution']['median']
        if a <= 0 or b <= 0:
            continue
        divisor = 1_000_000 if row['unit'] == 'ns/op' else 1
        result.append({'subject': key, 'reference': other, 'time_ratio': a / b,
                       'extra_ms': (a - b) / divisor})
    return result


def build_requests(data):
    # Validate inventory before sending evidence outside the process.
    from assessment import measured_slices
    measured_slices(data)
    variants = []
    for version in range(3):
        slices, questions = {}, {}
        for suite in data['manifest']['requested_suites']:
            rows = eligible_cases(data, suite)
            # Browser callback measurements never establish the requested UX basis.
            if suite == 'browser' or not any(role(k) == 'subject' for k in rows):
                continue
            facts = {}
            for key, row in rows.items():
                divisor = 1_000_000 if row['unit'] == 'ns/op' else 1
                stats = row['distribution']
                parameters = {k: v for k, v in row['parameters'].items()
                              if isinstance(v, (int, float, bool)) or k in ('contract', 'shape', 'target', 'method', 'size_unit')}
                facts[key] = {'role': role(key), 'parameters': parameters,
                              'median_ms': stats['median'] / divisor,
                              'min_ms': stats['min'] / divisor, 'max_ms': stats['max'] / divisor,
                              'mad_ms': stats['median_absolute_deviation'] / divisor, 'independent_n': stats['n'],
                              'iterations_per_sample': row.get('iterations_per_sample'),
                              'measured_batches_per_trial': row.get('raw_sample_counts', [])}
            slices[suite] = {'scope': SCOPES[suite][version], 'cases': facts, 'matched_pairs': pairs(rows)}
            questions[suite] = {'type': 'score', 'instructions': QUESTION[version].format(suite=suite),
                                'criteria': list(RUBRICS[version])}
        variants.append({'model': 'jev-latest', 'state': {'context': CONTEXT[version], 'sampling': SAMPLING[version],
                         'settings': {k: data['manifest'].get(k) for k in ('profile', 'trials', 'iterations', 'warmups', 'size')},
                         'slices': slices},
                         'questions': questions})
    return variants


def make_payload(data, requests, responses):
    if len(requests) != 3 or len(responses) != 3:
        raise ValueError('Grading requires three fresh consultations')
    expected = set(requests[0]['questions'])
    for request, response in zip(requests, responses):
        if (set(request['questions']) != expected or not isinstance(response, dict)
                or set(response.get('answers', {})) != expected
                or not isinstance(response.get('model'), str) or not response['model']):
            raise ValueError('Jev response does not match the requested slice questions')
        for suite, answer in response['answers'].items():
            validate_score_answer(answer)
            # The response must describe exactly the levels we supplied.
            legend = {str(i): text for i, text in enumerate(request['questions'][suite]['criteria'])}
            if answer.get('legend') != legend:
                raise ValueError(f'Jev returned a different rubric for {suite}')
    slices = {}
    for suite in data['manifest']['requested_suites']:
        excluded = [k for k, v in data['cases'].items() if k.startswith(suite + '/') and excluded_reason(k, v)]
        row = {'basis': BASIS[suite], 'excluded_cases': excluded,
               'answers': [r['answers'][suite] for r in responses] if suite in expected else []}
        if suite not in expected:
            row['ungraded_reason'] = ('Callback timing does not measure visual responsiveness; usable input-to-paint evidence is needed.'
                                     if suite == 'browser' else 'No eligible production measurements remain.')
        slices[suite] = row
    return {'kind': 'can.performance-assessment', 'schema_version': 1,
            'rubric_version': RUBRIC_VERSION, 'rubric': list(RUBRICS[0]),
            'evidence_fingerprint': evidence_fingerprint(data),
            'models': [r['model'] for r in responses], 'slices': slices}
