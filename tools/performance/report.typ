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
#let sum-wall(phase, suite: none) = steps.filter(step => step.at("phase", default: "") == phase and (suite == none or step.at("suite", default: none) == suite)).map(step => step.at("elapsed_seconds", default: 0)).sum(default: 0)

#set document(title: "Can performance report", author: "Can", date: none)
#set page(
  paper: "a4",
  margin: (top: 19mm, bottom: 18mm, x: 17mm),
  header: context [
    #small-label[CAN / PERFORMANCE]
    #h(1fr)
    #small-label(if preview { "SYNTHETIC PREVIEW" } else { meta.at("quality", default: "unknown").upper() })
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
#text(size: 13pt, fill: muted)[Twelve slices. One sequential run. Explicit comparison standards.]
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
  card("TARGET-ELIGIBLE", str(eligible("targets").len())),
  card("BASELINE-ELIGIBLE", str(eligible("baseline").len())),
)
#v(3mm)
#text(size: 9pt)[
  *Run status:* #text(meta.at("status", default: "unknown")) #h(8pt)
  *Evidence:* #text(if preview { "synthetic illustration" } else { meta.at("quality", default: "unknown") }) #h(8pt)
  *Selection:* #if requested.len() == suites.len() and suites.all(pair => requested.contains(pair.first())) { [all twelve slices] } else { [subset] }
]

= Suite overview
#set text(size: 9pt)
#frame-table((2.1fr, 1fr, 0.5fr, 0.7fr, 0.7fr, 0.9fr),
  table.header(..("Slice", "Status", "Cases", "Target", "Baseline", "Driver s").map(head-cell)),
  ..suites.map(pair => {
    let (id, label) = pair
    let ran = steps.any(step => step.at("suite", default: none) == id)
    let state = if not requested.contains(id) { "Not requested" } else if completed.contains(id) { "Complete" } else if ran { "Incomplete" } else { "Not run" }
    (text(label), text(state), str(count-for(id)),
      str(eligible("targets").filter(key => key.starts-with(id + "/")).len()),
      str(eligible("baseline").filter(key => key.starts-with(id + "/")).len()),
      if ran { fmt(sum-wall("trial", suite: id)) } else { "—" })
  }).flatten(),
)
#v(3mm)
#text(size: 8.5pt, fill: muted)[Target and baseline columns count cases with usable references. Case counts include workload and implementation variants; they are not percentages of feature coverage. Driver wall time includes orchestration and must not be summed into an overall performance score.]

#pagebreak()
#set text(size: 10pt)
= Cross-slice priorities
Ratios compare each case with its own declared reference. *2×* means twice the reference cost for a time metric. Higher-is-better metrics use the inverse ratio. These are descriptive rankings, not claims of statistical significance or business impact.

#for (kind, title, empty) in (("targets", "Target shortfalls", "No target exceedances"), ("baseline", "Relative regressions", "No relative regressions")) {
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

#pagebreak()
= Measurement context
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

== Reference coverage and exclusions
#for kind in ("targets", "baseline") {
  let excluded = board(kind).at("exclusions", default: (:))
  if excluded.len() > 0 {
    heading(level: 3, if kind == "targets" { [Targets] } else { [Baseline] })
    set text(size: 8.5pt)
    frame-table((1.3fr, 2fr),
      table.header(head-cell("Case"), head-cell("Reason not ranked")),
      ..excluded.pairs().sorted(key: pair => pair.first()).map(((key, reason)) => (soft(key), text(reason))).flatten(),
    )
  }
}

#pagebreak()
= Per-case evidence
The tables preserve the original unit for each operation. Exact parameters and timing boundaries follow each slice; they must match before a comparison is interpreted.
#for (id, label) in suites {
  let names = cases.keys().filter(key => key.starts-with(id + "/")).sorted()
  if names.len() > 0 {
    heading(level: 2, label)
    set text(size: 8.5pt)
    frame-table((2.5fr, 0.85fr, 0.9fr, 1.2fr, 0.7fr, 0.4fr),
      table.header(..("Case", "Median", "Unit", "Min–max", "MAD", "n").map(head-cell)),
      ..names.map(key => {
        let r = cases.at(key)
        let d = r.distribution
        (soft(key), fmt(d.median), text(r.unit), [#fmt(d.min)–#fmt(d.max)], fmt(d.median_absolute_deviation), str(d.n))
      }).flatten(),
    )
    for key in names {
      let r = cases.at(key)
      block(above: 7pt, below: 4pt)[
        #text(weight: "bold", soft(key))\
        #text(fill: muted, "Boundary: ")#text(r.timing_scope)\
        #text(fill: muted, "Parameters: ")#soft(json.encode(r.parameters, pretty: false))
        #for issue in r.at("ranking_issues", default: ()) { [\ #text(fill: amber, issue)] }
      ]
    }
  }
}

#pagebreak()
= Provenance
These identifiers bind the report to its source, dependencies, instrument and comparison references. Exporting the PDF does not run workloads or change the saved evidence.
#for (label, value) in (("Source", source), ("Host and tools", meta.at("environment", default: (:))), ("Isolation", meta.at("isolation", default: (:))), ("Comparison references", meta.at("ranking_references", default: (:)))) {
  heading(level: 2, label)
  set text(size: 8pt)
  for (key, value) in value.pairs().sorted(key: pair => pair.first()) {
    block(above: 5pt, breakable: true)[*#text(key)*\ #soft(json.encode(value, pretty: false))]
  }
}
