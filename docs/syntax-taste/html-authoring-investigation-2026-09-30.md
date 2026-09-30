# HTML authoring: tree proposal withdrawn

On 1 October 2026, the owner explicitly instructed that the tree syntax be discarded. This replaces the September 30 recommendation to retain it as a candidate. The earlier feedback alone had not authorized a broader restriction; this withdrawal follows the later explicit instruction.

The tree notation files, lowerers, generated tree examples and dependent comparison runners have been removed. The original investigation and deleted files remain in Git history. No production language or application code was involved.

No replacement authoring design is selected by this withdrawal. A fresh independent investigation is separate from these discarded proposals.

## Retained component observations

- Named typed field descriptions can centralize metadata, but interchangeable string values still permit wrong field bindings.
- Success-only assertions over opaque HTML values do not establish correct field wiring; independently observed output matters.
- Reusable helper proposals need executable assertion fixtures, not declaration inspection alone.

The [component example](preparation/html-authoring-followup-2026-09-30/components-current.can), [recorded execution results](preparation/html-authoring-followup-2026-09-30/probe-results.json), [output checks](preparation/html-authoring-followup-2026-09-30/output-checks.json) and [raw consultations](preparation/html-authoring-followup-2026-09-30/jev/) remain historical evidence. Their tree references and earlier advice are not active proposals or permission to resume that direction. The removed comparison runner is no longer reproducible from the current checkout; its historical version is available in Git.
