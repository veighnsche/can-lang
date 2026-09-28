// Offline template for can.performance-report JSON. No external packages.
// typst compile --input data=report.json report.typ report.pdf
#let data = json(sys.inputs.at("data", default: "report.json"))
#assert(data.at("kind") == "can.performance-report", message: "Expected a Can performance report")
#assert(data.at("schema_version") == 1, message: "Unsupported performance report schema")
#let meta = data.manifest
#let cases = data.cases
#let boards = data.rankings
#let preview = data.at("synthetic", default: false)
#let valid = meta.at("status", default: "unknown") == "complete" and meta.at("quality", default: "unknown") == "measurement"
#let source = if meta.at("source", default: none) == none { (:) } else { meta.source }
#let requested = meta.at("requested_suites", default: ())
#let completed = meta.at("completed_suites", default: ())
#let steps = meta.at("steps", default: ())
#let suites = (
  ("compiler", "Compiler"), ("assertions", "Assertions"),
  ("artifacts", "Artifacts"), ("generated", "Generated TypeScript"),
  ("runtime", "Runtime helpers"), ("codecs", "Codecs"),
  ("startup", "Startup"), ("browser", "Browser"),
  ("server", "Server"), ("io", "I/O"),
  ("editor", "Editor"), ("journeys", "Application journeys"),
)
#let ink = rgb("172C3C")
#let teal = rgb("087C80")
#let muted = rgb("586975")
#let pale = rgb("F0F5F6")
#let rule = rgb("D8E1E5")
#let amber = rgb("8D4D12")
#let mono = "DejaVu Sans Mono"
#let fmt(value) = if value == none { "—" } else if type(value) == float {
  if value != 0 and (calc.abs(value) < 0.001 or calc.abs(value) >= 1000000) {
    str(value)
  } else { str(calc.round(value, digits: 3)) }
} else { str(value) }
#let literal(value) = if type(value) == str { value } else { json.encode(value, pretty: false) }
#let time-units = ("ns/op", "ms/launch", "ms/interaction", "ms/trial", "ms/op", "ms/journey")
#let unresolved(row) = time-units.contains(row.unit) and (row.distribution.at("min", default: none) == 0 or row.distribution.median == 0)
#let unresolved-cases = cases.keys().filter(key => unresolved(cases.at(key)))
#let soft(value) = text(literal(value).replace("/", "/\u{200b}").replace("_", "_\u{200b}").replace(".", ".\u{200b}"))
#let small-label(body) = text(font: mono, size: 7.5pt, fill: muted, tracking: 0.4pt, body)
#let head-cell(body) = text(weight: "bold", fill: white, body)
#let frame-table(columns, compact: false, ..cells) = table(
  columns: columns,
  inset: (x: 6pt, y: if compact { 3.5pt } else { 6pt }),
  stroke: (top: none, left: none, right: none, bottom: 0.4pt + rule),
  fill: (x, y) => if y == 0 { ink } else if calc.rem(y, 2) == 0 { pale },
  ..cells.pos(),
)
#let count-for(id) = cases.keys().filter(key => key.starts-with(id + "/")).len()
#let board(kind) = boards.at(kind, default: (:))
#let eligible(kind) = if valid and ("ranked", "partial").contains(board(kind).at("status", default: "unavailable")) {
  board(kind).at("per_case", default: (:)).keys()
} else { () }
#let available-boards = ("targets", "baseline").filter(kind => meta.at("ranking_references", default: (:)).at(kind, default: none) != none or ("ranked", "partial").contains(board(kind).at("status", default: "unavailable")))
#let brief(value) = if value > 0 and value < 0.01 { "<0.01" } else { str(calc.round(value, digits: if calc.abs(value) >= 100 { 0 } else if calc.abs(value) >= 10 { 1 } else { 2 })) }
#let ratio(value) = str(calc.round(value, digits: 1))
#let duration(value, unit) = {
  if value == none { return "—" }
  if not time-units.contains(unit) { return fmt(value) + " " + unit }
  let seconds = value / if unit == "ns/op" { 1000000000 } else { 1000 }
  if calc.abs(seconds) >= 1 { brief(seconds) + " s" }
  else if calc.abs(seconds) >= 0.001 { brief(seconds * 1000) + " ms" }
  else if calc.abs(seconds) >= 0.000001 { brief(seconds * 1000000) + " µs" }
  else { brief(seconds * 1000000000) + " ns" }
}
#let case-name(key) = {
  let parts = key.split("/")
  let name = parts.slice(1).join("/")
  let prefix = parts.first() + "."
  if name.starts-with(prefix) { name = name.slice(prefix.len()) }
  let friendly = (frequency: "Word frequency", doubled: "Array mapping", "fold-sum": "Sum reduction", "generic-map": "Generic mapping", "captured-map": "Mapping with captured values", "branch-arithmetic": "Arithmetic and branching", "record-projection": "Record field access", "record-update": "Immutable record update", "tagged-variant": "Variant dispatch", "failure-recovery": "Failure recovery", "unicode-normalize": "Unicode normalization", "utf8-base64": "UTF-8 to Base64")
  if parts.first() == "generated" {
    let segments = name.split(".")
    let title = friendly.at(segments.first(), default: segments.first())
    let implementation = if segments.last() == "can" { "compiled Can" } else if segments.last() == "native-adapter" { "native with adapter" } else { "native" }
    return title + " · " + implementation
  }
  name.replace(".", " · ").replace("-", " ").replace("_", " ")
}
#let descriptions = (
  compiler: "Time to load, check and emit a project. Phases overlap; do not add them together.",
  assertions: "Time for the supervisor to run the complete assertion workload, including worker startup.",
  artifacts: "Time for each output-processing operation, including validation and publication.",
  generated: "Time to execute the compiled program or handwritten native reference on the named workload. Each is a complete workload, not one array element.",
  runtime: "Time for a complete helper workload, such as mapping or a sequence of immutable updates.",
  codecs: "Time to encode, decode or reject the complete payload. Native parsing omits schema and contract validation.",
  startup: "Time for one fresh process to launch, run its check, produce output and exit. OS caches are uncontrolled.",
  browser: "Time inside the control callback or property operation. This does not measure input-to-paint latency.",
  server: "Time to complete one scheduled request batch. Arrival-rate timing includes deliberate spacing; it is not single-request latency or capacity.",
  io: "Time for one complete bounded read or write/read workload. OS-managed caches remain enabled.",
  editor: "Time for the real language server to answer the named operation, including transport but excluding startup. Diagnostics alternate invalid and valid edits; the combined median hides their different checking costs.",
  journeys: "Time for one complete compiled application request, including its data conversion and business logic.",
)
// Stable named checkpoints make the first page useful without selecting winners
// or ranking unrelated units. Missing workloads are left unavailable.
#let checkpoints = (
  compiler: ("compiler/compiler.invoice-compare.pipeline", "Invoice project · compile pipeline"),
  assertions: ("assertions/assertions.flat.supervised-roots", "100-function fixture · all assertion roots"),
  artifacts: ("artifacts/artifacts.flat.publish-new", "First publication of compiled output"),
  generated: ("generated/generated.frequency.can", "Compiled word frequency workload"),
  runtime: ("runtime/runtime.map.insert-chain", "Immutable map update workload"),
  codecs: ("codecs/codecs.json.nested-records.decode-contract", "Decode nested records with validation"),
  startup: ("startup/startup.bun.application-bundle", "Launch bundled application"),
  browser: ("browser/chromium.c02.input-property-snapshot", "Chromium · input property snapshot"),
  server: ("server/runtime-http.get.bounded-clients", "GET batch · 200 requests, bounded clients"),
  io: ("io/runtime.binary-write-read", "Binary write and read workload"),
  editor: ("editor/editor.flat.completion", "Code completion response"),
  journeys: ("journeys/emitted-can.invoice.valid", "Valid invoice request"),
)
#let native-pairs = if not valid { () } else { cases.keys().filter(key => key.starts-with("generated/") and key.ends-with(".can")).map(key => {
  let base = key.slice(0, key.len() - 4)
  let other = if base + ".native" in cases { base + ".native" } else { base + ".native-adapter" }
  let can = cases.at(key)
  let native = cases.at(other, default: none)
  if native == none or not time-units.contains(can.unit) or can.distribution.median <= 0 or native.distribution.median <= 0 or can.parameters != native.parameters or can.unit != native.unit or can.timing_scope != native.timing_scope or can.at("iterations_per_sample", default: none) != native.at("iterations_per_sample", default: none) or unresolved(can) or unresolved(native) or can.at("ranking_issues", default: ()).len() > 0 or native.at("ranking_issues", default: ()).len() > 0 { return none }
  (key: key, can: can, native: native, ratio: can.distribution.median / native.distribution.median)
}).filter(pair => pair != none).sorted(key: pair => -pair.ratio) }
#let checkpoint-time(key) = {
  let row = cases.at(key, default: none)
  if row == none { return "not measured" }
  if unresolved(row) { return "unresolved" }
  duration(row.distribution.median, row.unit)
}
#let finding(title, body) = block(width: 100%, above: 8pt, below: 7pt)[
  #text(size: 12pt, weight: "bold", fill: teal, title)\
  #text(size: 10pt, body)
]
#let sum-wall(phase, suite: none) = steps.filter(step => step.at("phase", default: "") == phase and (suite == none or step.at("suite", default: none) == suite)).map(step => step.at("elapsed_seconds", default: 0)).sum(default: 0)

#set document(title: "Can performance report", author: "Can", date: none)
#set page(
  paper: "a4",
  margin: (top: 19mm, bottom: 18mm, x: 17mm),
  header: context [
    #small-label[CAN / PERFORMANCE]
    #h(1fr)
    #small-label(if preview { "SYNTHETIC PREVIEW" } else { upper(meta.at("quality", default: "unknown")) })
  ],
  footer: context [
    #line(length: 100%, stroke: 0.5pt + rule)
    #v(3pt)
    #text(size: 8pt, fill: muted)[
      #if preview { [Illustrative values only · no benchmark measurements] } else { [Saved evidence · sequential suite execution] }
      #h(1fr)
      #counter(page).display("1 / 1", both: true)
    ]
  ],
)
#set text(font: "Libertinus Serif", size: 10pt, fill: ink)
#set par(leading: 0.65em, justify: false)
#set heading(numbering: none)
#show heading.where(level: 1): it => block(above: 15pt, below: 9pt)[#text(size: 22pt, weight: "bold", it.body)]
#show heading.where(level: 2): it => block(above: 13pt, below: 6pt)[#text(size: 14pt, weight: "bold", it.body)]

#small-label[ENGINEERING EVIDENCE / CAN]
#v(5mm)
#text(size: 36pt, weight: "bold")[Performance report]
#v(2mm)
#text(size: 13pt, fill: muted)[Measured results across twelve parts of Can.]
#v(5mm)
#line(length: 100%, stroke: 2pt + teal)
#v(4mm)

#if preview {
  block(width: 100%, fill: rgb("FFF2DD"), inset: 10pt, radius: 3pt)[
    *SYNTHETIC PREVIEW — these values illustrate the layout.*
    No performance workloads were run to create this document.
  ]
}
#if not valid {
  block(width: 100%, fill: rgb("FFF2DD"), inset: 10pt, radius: 3pt)[
    *Not eligible for performance rankings.*
    Smoke, exploratory and incomplete results are evidence of functional checks or partial work, not accepted performance baselines.
  ]
}
#if meta.at("reason", default: none) != none {
  block(width: 100%, fill: pale, inset: 10pt)[*Failure or limitation:* #text(meta.reason)]
}

#let card(label, value) = block(width: 100%, fill: pale, inset: 10pt, radius: 3pt)[
  #small-label(label)
  #v(3pt)
  #text(size: 20pt, weight: "bold", value)
]
#v(4mm)
#grid(columns: (1fr, 1fr, 1fr, 1fr), gutter: 7pt,
  card("COMPLETED", str(completed.filter(id => requested.contains(id)).len()) + " / " + str(requested.len())),
  card("CASES", str(cases.len())),
  card("RESOLVED", str(cases.len() - unresolved-cases.len())),
  card("UNRESOLVED", str(unresolved-cases.len())),
)
#v(3mm)
#text(size: 9pt)[
  *Run status:* #text(meta.at("status", default: "unknown")) #h(8pt)
  *Evidence:* #text(if preview { "synthetic illustration" } else { meta.at("quality", default: "unknown") }) #h(8pt)
  *Selection:* #if requested.len() == suites.len() and suites.all(pair => requested.contains(pair.first())) { [all twelve slices] } else { [subset] }
]

#heading(level: 1, if valid { "What the measurements show" } else { "Recorded checks" })
#if native-pairs.len() > 0 {
  let pair = native-pairs.first()
  let lo = native-pairs.last().ratio
  finding("Generated code · compared with native execution", [
    Across #native-pairs.len() matched workloads, compiled Can takes #ratio(lo)–#ratio(pair.ratio)× the native reference time.
    The highest time ratio is *#case-name(pair.key)*: *#duration(pair.can.distribution.median, pair.can.unit)* versus *#duration(pair.native.distribution.median, pair.native.unit)*.
    These compare the same tested endpoint contracts; they do not isolate code generation from runtime-helper cost.
    #if pair.native.distribution.median_absolute_deviation / pair.native.distribution.median > 0.1 { [That ratio is approximate: the native reference has #ratio(100 * pair.native.distribution.median_absolute_deviation / pair.native.distribution.median)% relative spread.] }
  ])
}
#if valid and "compiler/compiler.invoice-compare.pipeline" in cases and "compiler/compiler.invoice-compare.check" in cases {
  let stages = (("checking", "check"), ("loading", "load"), ("emission", "emit")).filter(pair => "compiler/compiler.invoice-compare." + pair.last() in cases).sorted(key: pair => -cases.at("compiler/compiler.invoice-compare." + pair.last()).distribution.median)
  finding("Compilation · " + stages.first().first() + " is the largest measured phase", [
    The invoice comparison project takes *#checkpoint-time("compiler/compiler.invoice-compare.pipeline")* to load, check and emit.
    Checking alone measures *#checkpoint-time("compiler/compiler.invoice-compare.check")*; emission measures *#checkpoint-time("compiler/compiler.invoice-compare.emit")*.
    The phases are measured separately and must not be added together. Validation, assertions and publication are outside this compile timer.
  ])
}
#if valid and "editor/editor.flat.completion" in cases and "editor/editor.flat.rename" in cases {
  finding("Editor · time to receive a response", [
    Code completion measures *#checkpoint-time("editor/editor.flat.completion")* and rename *#checkpoint-time("editor/editor.flat.rename")* in the test project.
    These are end-to-end language-server response times, including transport. They measure the delay before a result is returned; they do not explain its cause.
  ])
}
#if not valid {
  [These records establish what the checks executed. They do not support performance conclusions. Use a complete measurement run before interpreting speed or ratios.]
} else if native-pairs.len() == 0 and not ("compiler/compiler.invoice-compare.pipeline" in cases) and not ("editor/editor.flat.completion" in cases) {
  [The following tables show all recorded workload durations. This run does not contain the standard checkpoints used for the findings above.]
}
#v(2mm)
#if unresolved-cases.len() > 0 {
  block(width: 100%, fill: rgb("FFF2DD"), inset: 10pt, radius: 3pt)[
    *Timing limitation: #unresolved-cases.len() cases are unresolved.*
    The timer could not reliably resolve these durations. Correctness can pass without a usable speed measurement; zero does not mean instantaneous.
  ]
}

#v(2mm)
#if available-boards.len() == 0 {
  text(size: 9pt, fill: muted)[*Cross-slice ranking:* No earlier run or reviewed target profile was attached. A defensible top ten across all twelve slices is therefore unavailable. Same-run native comparisons above remain useful within their matched workloads.]
} else {
  text(size: 9pt, fill: muted)[*Cross-slice ranking:* Reference-based top tens follow the measured results. Target shortfalls and changes since an earlier run have separate rankings.]
}
#pagebreak()
= Across all twelve slices

#set text(size: 8.8pt)
#frame-table((1.05fr, 2.35fr, 0.8fr), compact: true,
  table.header(..("Slice", "Selected checkpoint", "Typical time").map(head-cell)),
  ..suites.map(pair => {
    let (id, label) = pair
    let (key, caption) = checkpoints.at(id)
    let row = cases.at(key, default: none)
    let value = if row == none { "Not measured" } else if unresolved(row) { "Unresolved" } else { duration(row.distribution.median, row.unit) }
    let detail = caption
    if row != none and id == "assertions" { detail = str(row.parameters.at("size", default: "?")) + "-function fixture · all assertion roots" }
    if row != none and id == "server" { detail = "GET batch · " + str(row.parameters.at("request_count", default: "?")) + " requests, bounded clients" }
    (text(label), text(detail), text(weight: "bold", fill: if row != none and unresolved(row) { amber } else { teal }, value))
  }).flatten(),
)
#v(2mm)
#text(size: 8.5pt, fill: muted)[Named checkpoints, not a ranking or an overall score. All #cases.len() case results follow. “Typical” is the median across independent process trials. One millisecond (ms) is a thousandth of a second; one microsecond (µs) is a millionth.]
#v(2mm)
#v(4mm)
== How to read the results
Each row is a named workload, not a score for the whole slice. A short codec operation and a complete compilation do different amounts of work, so comparing their raw durations would be misleading.

The detailed tables show typical time and variation across independent runs. The generated-code findings use matched native workloads in this same run, so they do not require a historical baseline. A higher ratio means the compiled version took longer for that specific workload.

#if valid { [*What to do next:* Inspect the largest compiler phase, the generated/native gaps and the editor response times. Improve browser timing resolution before using browser numbers to judge speed.] } else { [*What to do next:* Use these records for functional validation. Performance conclusions require a complete measurement run.] }

#if native-pairs.len() > 0 [
#pagebreak()
= Generated code vs native
These comparisons are available *within this run*. They compare the same workload, input size and tested result contract. They do not need an earlier baseline.

A ratio of *2×* means the compiled Can version took twice as long as the reference for that workload. This table is ordered by that ratio, not by the absolute time saved in a complete application.
#set text(size: 9pt)
#frame-table((2fr, 0.9fr, 0.9fr, 0.75fr),
  table.header(..("Workload", "Compiled Can", "Native", "Time ratio").map(head-cell)),
  ..native-pairs.map(pair => (
    text(case-name(pair.key).replace(" · compiled Can", "")),
    duration(pair.can.distribution.median, pair.can.unit),
    duration(pair.native.distribution.median, pair.native.unit),
    text(weight: "bold", fill: teal)[#ratio(pair.ratio)×],
  )).flatten(),
)
#v(4mm)
#set text(size: 10pt)
*Interpretation:* These ratios compare generated code and its runtime calls with the handwritten reference. They do not establish one overhead for all Can programs, or show what a language rewrite would achieve.

The native implementations preserve the tested endpoints. They do not cover every ownership or failure behavior. The record-update reference keeps the nominal-record adapter; the recovery reference does not recreate internal failure construction.

All durations are for the entire configured workload. Full trial ranges and spread are in the case tables that follow.
]

#pagebreak()
#set text(size: 10pt)
= All measured results
Each row describes one workload. *Typical time* is the median; *trial range* shows the lowest and highest trial medians; *spread* is their median absolute deviation (MAD). Lower times mean less elapsed time for that same workload. Different workloads cannot be ranked by duration alone.

Times are converted to readable units without changing the evidence. “Per operation” means the whole configured workload, not each item it processes. This report is a reading guide. Exact case identifiers, input sizes and timer boundaries remain in the saved evidence and the Markdown report’s technical appendix.
#if unresolved-cases.len() > 0 { [*Unresolved* means at least one trial median was zero. Its observed range is preserved, but a reliable duration was not measured.] }
#for (id, label) in suites {
  let names = cases.keys().filter(key => key.starts-with(id + "/")).sorted()
  if names.len() > 0 {
    if ("generated", "codecs", "server").contains(id) { pagebreak() }
    heading(level: 2, [#label #text(size: 9pt, fill: muted)[· #names.len() cases]])
    text(size: 9pt, fill: muted, descriptions.at(id))
    v(4pt)
    set text(size: 8.5pt)
    frame-table((2.1fr, 0.95fr, 1.4fr, 0.8fr, 0.4fr), compact: true,
      table.header(..("Workload", "Typical time", "Trial range", "Spread", "Trials").map(head-cell)),
      ..names.map(key => {
        let r = cases.at(key)
        let d = r.distribution
        (soft(case-name(key)), if unresolved(r) { text(fill: amber, "Unresolved") } else { duration(d.median, r.unit) },
          [#duration(d.min, r.unit) – #duration(d.max, r.unit)], duration(d.median_absolute_deviation, r.unit), str(d.n))
      }).flatten(),
    )
  }
}

#if available-boards.len() > 0 [
#pagebreak()
= Cross-slice priorities
Ratios compare each case with its own declared reference. *2×* means twice the reference cost for a time metric. Higher-is-better metrics use the inverse ratio. These are descriptive rankings, not claims of statistical significance or business impact.

#for (kind, title, empty) in (("targets", "Target shortfalls", "No target exceedances"), ("baseline", "Relative regressions", "No relative regressions")) {
  if not available-boards.contains(kind) { continue }
  let b = board(kind)
  let reference = meta.at("ranking_references", default: (:)).at(kind, default: none)
  heading(level: 2, [Top 10 · #title])
  if reference != none {
    let label = if kind == "targets" { reference.at("name", default: "Unnamed target profile") } else {
      let base-source = reference.at("source", default: (:))
      "Baseline revision: " + base-source.at("revision", default: "not recorded")
    }
    text(size: 8.5pt, fill: muted, soft(label))
    parbreak()
  }
  if not valid {
    block(fill: pale, width: 100%, inset: 10pt)[Ineligible evidence. No ranked positions are shown.]
  } else {
    let state = b.at("status", default: "unavailable")
    text(size: 9pt)[*#text(state)* · #b.at("eligible_cases", default: 0) / #b.at("total_cases", default: cases.len()) eligible cases. #text(b.at("reason", default: "No compatible reference supplied."))]
    parbreak()
    if state == "partial" { text(fill: amber)[Partial coverage: conclusions apply only to eligible cases.] }
    let rows = if ("ranked", "partial").contains(state) { b.at("rows", default: ()).slice(0, calc.min(10, b.at("rows", default: ()).len())) } else { () }
    if rows.len() > 0 {
      set text(size: 8.5pt)
      frame-table((0.3fr, 2.55fr, 1.15fr, 1.15fr, 0.65fr, 0.9fr), compact: true,
        table.header(..("#", "Case / unit", "Current", "Reference", "Ratio", "Trials / MAD").map(head-cell)),
        ..rows.enumerate().map(((i, r)) => (
          str(i + 1), [#soft(r.case)\ #text(size: 7.5pt, fill: muted, r.unit)],
          fmt(r.observed), fmt(r.reference), text(weight: "bold", fill: teal)[#fmt(r.ratio)×],
          [#r.trial_count / #fmt(r.mad)],
        )).flatten(),
      )
    } else {
      block(fill: pale, width: 100%, inset: 10pt)[
        #if ("ranked", "partial").contains(state) and b.at("eligible_cases", default: 0) > 0 { [#empty among eligible cases.] } else { [No ranked positions available. Supply reviewed targets or a compatible measured baseline.] }
      ]
    }
  }
}
#v(2mm)
#text(size: 8.5pt, fill: muted)[Only ratios above 1 appear. Target shortfalls and historical regressions have different meanings and are never merged. Min–max and MAD describe independent trial medians, not request-tail percentiles. Overlapping phases and repeated variants are not independent costs.]


]

#pagebreak()
= Appendix · measurement details
The pages above contain the results. This appendix records the run settings and interpretation limits. Full reproduction details remain in the saved evidence.

== Run settings
#let settings = ("profile", "trials", "iterations", "warmups", "size")
#frame-table((1fr, 3fr),
  table.header(head-cell("Property"), head-cell("Recorded value")),
  ..settings.filter(key => key in meta).map(key => (text(key), soft(meta.at(key)))).flatten(),
  [Total wall time], [#fmt(meta.at("wall_seconds", default: none)) s],
  [Preparation drivers], [#fmt(sum-wall("prepare")) s],
  [Suite drivers], [#fmt(sum-wall("trial")) s],
  [Revision], soft(source.at("revision", default: "Not recorded")),
  [Source mode], soft(source.at("source_mode", default: "Not recorded")),
)
#v(3mm)
Each headline is the median of independent process-trial medians. Warmup batches are excluded. Timings retain the workload size, implementation variant and start/stop boundary. Ratios require compatible units, contracts, host/tool identity and measurement settings.


#if available-boards.len() > 0 {
  heading(level: 2, "Comparison coverage")
  for kind in available-boards {
    let excluded = board(kind).at("exclusions", default: (:))
    if excluded.len() > 0 {
      heading(level: 3, if kind == "targets" { [Targets] } else { [Baseline] })
      // Global exclusions occupy one row regardless of case count.
      let reasons = excluded.values().dedup().sorted()
      set text(size: 8.5pt)
      frame-table((2.6fr, 0.5fr),
        table.header(head-cell("Reason not ranked"), head-cell("Cases")),
        ..reasons.map(reason => (text(reason), str(excluded.values().filter(value => value == reason).len()))).flatten(),
      )
      if reasons.len() > 1 {
        for reason in reasons {
          block(above: 5pt)[*#text(reason)*\ #soft(excluded.pairs().filter(pair => pair.last() == reason).map(pair => pair.first()).sorted().join(", "))]
        }
      }
    }
  }
}

== Interpretation limits
- These are representative workloads, not exhaustive language or application coverage.
- Generated/native comparisons cover the tested endpoint contracts. Different failure paths or ownership requirements may change the result.
- Browser control timings exclude queueing, layout and paint. Unresolved observations need better instrumentation before comparison.
- Server rows are whole-batch durations. The arrival-rate test deliberately spaces requests and cannot be read as per-request latency.
- Compiler phase measurements overlap. Adding them would double-count work.
- Trial range and spread describe trial medians, not tail latency experienced by individual requests.

== Evidence and storage
Raw samples, correctness results, workload contracts and source/tool fingerprints remain in the compact evidence archive. Temporary execution files are cleaned separately. Exporting this report does not run performance workloads or modify that archive.
