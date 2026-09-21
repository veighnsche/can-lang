# I27 phase lowering consultation

All three fresh requests chose `prepared_graph` (jev-1.13.0, confidence 1). No disagreement required investigation. This is advisory evidence only. The specification independently requires once-only preparation, retention of selected arm values, whole-batch validation, and ordered handlers. The existing initialization graph supplies deterministic forward-reference ordering and cycle rejection; named arms now participate through their description dependencies instead of being assumed externally ready.

Implementation validation must cover mixed batches, static and dynamic options, generated records, metadata and fallback scope, arm forward references and cycles, and failure ordering. Consultation agreement does not satisfy those tests.
