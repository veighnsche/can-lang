# I18 fixture scheduler scalability correction

Independent review found two repeated operations per fixture release: scanning
all frame phases to detect quiescence, and sorting/shifting the entire pending
queue. Its ascending-queue probe measured 499,510 comparisons for 1,000 requests
and 7,998,010 for 4,000 requests.

Frame transitions now maintain the exact count of reserved/running frames.
Admission checks that count in constant time. A binary priority heap retains
lexical request ordering across releases and accepts new earlier-path arrivals
with logarithmic insertion/removal. An enqueue serial preserves stable ties for
conformance comparators. Native Promise delivery and explicit park/finish
boundaries are unchanged. No host priority-queue primitive exists; this is an
internal assertion contract adapter, not a replacement for a Can collection
operation.

An independent pending-request set retains every request while heap operations
run. If a comparator throws or returns a nonfinite result, all queued promises
are rejected and all affected frames become running, even after partial heap
mutation. The existing empty-coordination selection gate remains in place.

Regression coverage includes 1,000 and 4,000 participants with reversed arrivals
and a first participant that repeatedly rejoins ahead of the retained queue.
An explicit O(n log n) comparator budget checks algorithmic work rather than
elapsed time. Additional tests exercise a comparator exception during heap
removal, complete rejection/recovery, and stable equal-priority ordering.

The original independent probe against the corrected runtime measured 3,106,
15,965, and 80,151 comparisons for 256, 1,000, and 4,000 requests respectively.
These counts, not machine-dependent timings, establish the reduction in work.

Validation:
- Qualified Bun runtime suite: 151 tests, 1,117 expectations, all passed. This
  workspace-wide run includes the still-uncommitted I21 runtime tests.
- Strict TypeScript checking passed for the changed scheduler and tests.
- An isolated snapshot of commit `63211ee` plus only the scheduler correction
  passed `go test ./compiler/... ./tests/integration -count=1` with qualified Bun,
  the pinned local archive, and strict generated-TypeScript checking enabled.
- The new comparison-budget regression was also run against the old scheduler.
  It failed at 1,000 participants with 1,501,507 comparisons against a limit of
  79,960, proving the test detects the reported defect.

Three fresh design consultations and the wording audit are in `i18-barrier-jev`.
Their agreement is advice, not proof. I21 remains uncommitted and incomplete.
