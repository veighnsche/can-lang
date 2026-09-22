# a85 — Deep-research prompt: best language for an AI coding agent (pre-decision, 2026-09-18)

Status: paste-ready prompt. We skipped the research phase for can-lang; this commissions it retroactively.

---

# Deep research: what makes a programming language good for AI coding agents?

## Context
I am designing `can-lang`, an experimental contract-first language optimized for AI agents (not humans) to read, write, and verify. Current thesis: verbose, explicit, single canonical form; no inference an agent would have to guess; tests + logic are one artifact (every function ships a decision table + branch evidence); declared error sets; machine-checkable contracts (termination, effects, exact numerics); transpiles to TypeScript.

I skipped the research phase. Now I need it retroactively: what have OTHER people already discovered about requirements for agent-friendly languages?

## Research question
What language requirements, design principles, and empirical findings exist about the best language (or language features) for AI coding agents to read, write, modify, and verify code reliably?

## Scope — cover all of these
1. **LLM-oriented PL design**: papers/proposals for languages or subsets designed for LLM generation (e.g. constrained grammars, canonical forms, explicit types, no sugar, structured errors). Both academic and industry (e.g. DSPy-style signatures, typed agent DSLs, "LLM-first" language sketches).
2. **Empirical error taxonomies**: what goes wrong when agents write code — syntax vs type vs semantic vs API-hallucination vs control-flow errors; which language features correlate with fewer agent errors (explicitness, verbosity, delimiters, naming, error messages, small stdlib, totality).
3. **Verification + contracts for agents**: decision tables / example-driven specs, property tests, refinement types, effect systems, termination checking — evidence on what actually helps agents vs what only helps humans.
4. **Tooling interface**: machine-readable artifacts (diagnostics JSON, coverage maps, LSP/AST access), formatter-canonicalized syntax, deterministic builds — what agent harnesses demonstrably benefit from.
5. **Counter-evidence**: cases where "agent-friendly" ideas failed, hurt, or didn't transfer (e.g. verbosity exhausting context, over-constraint reducing solvability, agents routing around strictness).

## Instructions
- Search academic literature (PLDI, POPL, OOPSLA, ICSE, FSE, NeurIPS/ICLR agent papers), industry engineering blogs, agent-harness postmortems (SWE-bench, SWE-agent, Devin/Cursor/Copilot reports), and language-design discussions. Verify version/date claims live; don't assert currency from training data.
- Distinguish (a) measured results with benchmarks, (b) reasoned proposals with prototypes, (c) pure opinion. Label each finding accordingly.
- For each finding give: claim, source (link + date), evidence strength, and any known replication or rebuttal.
- Surface disagreements explicitly (where sources contradict each other).
- Do NOT start from my thesis and confirm it. Treat can-lang's choices as hypotheses to check, and flag where prior art disagrees with them.

## Deliverable format
1. **Requirements catalog** — table: ID | requirement (one sentence) | why it helps agents | evidence (measured/proposed/opinion + source) | confidence (high/med/low).
2. **Design dimensions + tradeoffs** — e.g. verbosity vs context budget, strictness vs solvability, explicitness vs boilerplate errors; with guidance on where the sweet spot measured out.
3. **Top 10 highest-ROI requirements** ranked by evidence strength, each with falsifiable acceptance check (what experiment would prove/disprove it).
4. **Gaps in my thesis** — requirements others found important that can-lang currently ignores, and can-lang bets with no outside support.
5. **Source list** — full links, grouped by type (papers / benchmarks / postmortems / proposals), newest first within groups.
6. **Unresolved** — questions the sources don't answer, stated as open items, not silently dropped.

Depth over brevity. I want the uncomfortable findings, not validation.
