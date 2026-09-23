# G-EVENT decision audit (B1-05.02)

Three independently rewritten packets asked four identical questions over
the nine-program comparison evidence. All prose (state, instructions,
criteria) differs across packets; question keys, option keys, identifiers
and measured numbers are stable. Verified mechanically before dispatch.
Model: `jev-1.13.0` via `jev-latest`, three HTTP 200 rounds.

## Results

| Question | Round 1 | Round 2 | Round 3 |
|---|---|---|---|
| surface | library 0.80 (conf 0.70) | library 0.82 (conf 0.73) | library 0.77 (conf 0.66) |
| sugar | later 0.60 (conf 0.41) | later 0.63 (conf 0.45) | later 0.48 (conf 0.21) |
| drain | event 0.91 (conf 0.86) | event 0.57 (conf 0.35) | mixed 0.51 (conf 0.26) |
| transcript | fifo 0.80 (conf 0.70) | fifo 0.97 (conf 0.95) | fifo 1.00 (conf 1.00) |

Requests, responses and SHA metadata: `request-N.json`,
`response-N.json`, `response-N.metadata.json` in this directory.

## Agreement (accepted as advice)

- Transcript `fifo` (0.80/0.97/1.00): repeated selectors on one read
  site replay FIFO with per-row argument verification. Independently
  verified in `runtime/assert/fixtures.ts` before consultation.
- Sugar `later` (weak, pilot runner-up 0.33-0.41): build the core now,
  add a consume helper only on boilerplate evidence. Accepted under the
  pull core: a style-L consume remains a catalogue-only addition later.

## Disagreement (investigated, surface overridden)

Jev prefers surface `library` 3/3 against the pull recommendation. The
packets presented both sides honestly (pull's 152-vs-88-line cost and
per-arm closes; library's proven `$callback` precedent), so the split
was judged, not smuggled. The decision still selects pull, on contract
evidence the packets did not fully carry:

1. B1-06.02 requires "using B1-05 for incremental reads/writes", and
   the B1-06 title promises incremental bodies. A consume-only core has
   no incremental read/write operations to reuse; only pull supplies
   them. Downstream need, not taste, forces the choice.
2. B1-05.07 requires fixtures to "drive the same dispatch/owner path
   as native events" and to test extra events, missing terminals and
   exhaustion "through real dispatch". Under library dispatch, the loop
   is adapter-internal: fixtures can only substitute the whole run, so
   native and fixture runs take different paths by construction. Pull
   keeps the loop in caller code where per-read substitution applies.
3. The acceptance Runtime rows (abort during pending read, close twice,
   cancelled read rejects late, queue cap reached) name caller-visible
   lifecycle operations. Under library style those operations do not
   exist at the Can level and the rows degrade to adapter unit tests.
4. Jev's own answers are mutually incoherent: `transcript=fifo`
   presumes repeated caller-visible read sites to attach transcripts
   to, which only pull programs have. Library + fifo cannot both hold.
5. The planned runtime layout (`transport/stream/readable`,
   `writable`, `lifecycle`, `budget` in `filetree.md`) is already
   pull-shaped; library style would strand it.

Probabilities here partly reflect the packets' emphasis on caller
brevity and precedent, which are real virtues but do not satisfy the
cited contract rows. The drain answers assumed a push dispatcher and
are moot under pull: bounded demand replaces the drain case (see the
W3 pull program). The sugar answers assumed a pull core and transfer
unchanged.

## Decision (B1-05.02 exit)

- Surface: shared-event pull (E). No new grammar: all nine programs
  parse, and pull needs only catalogue entries (4 ops, 2 handle types,
  3 errors, producers) over existing recursion, error-arm and fixture
  machinery.
- Sugar: deferred. Revisit consume only when maintained pull programs
  prove the close boilerplate is a burden.
- Drain: demand. No drain case in the caller contract.
- Transcripts: repeated-selector FIFO with argument verification, using
  the existing `withFixture` engine semantics.

This packet set the questionnaire flaw on record: sugar/drain assumed
a pull core while surface stayed open, so the surface minority/majority
cannot be read as a full ranking. The override above rests on cited
contract rows, not on re-weighting the advice.
