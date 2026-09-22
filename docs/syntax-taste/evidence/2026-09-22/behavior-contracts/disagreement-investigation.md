# Payload disagreement investigation

Packet choices for `payload` were `reuse`, `reuse`, `new_records`. Their probability assigned to `reuse` was 0.71, 0.89 and 0.36 respectively. This is a real wording-sensitive disagreement despite semantically equivalent requests. The classifier supplies no explanation; none is inferred from confidence.

Rechecked the evidence rather than taking a majority vote:

- `compiler/internal/catalogue/catalogue.json` defines exactly the seven payload shapes used in B1. There is no proposed field deletion, addition, renaming or exposure-policy change requiring a second representation. Reserved ID 1106 is not presently allocated.
- C4 permits error values as nominal variant leaves. The proposed finite `http::failure_detail` is therefore a use of the existing data model, not a new universal failure type.
- `runtime/domain.ts` already separates error values from failure occurrence/cause. Public detail can reuse the value while a new normalized occurrence retains its original occurrence privately. New detail records would not solve origin classification; both alternatives need that independently.
- The earlier full-review disagreement investigation reached the same engineering distinction. It remains relevant evidence, not an additional fresh consultation counted here.

Decision: reuse existing typed payload leaves. The native-origin versus emitted-origin table and original-cause retention are mandatory independent of representation. This keeps all selective recovery fields and avoids seven equal-shape catalogue records/conversions. If a future public contract intentionally needs different fields, that is a new design decision; the current design is settled without pretending the classifier established consensus.

No selected-option disagreement occurred on the other eight choices. Snapshot and wrapper-bound probabilities still varied; agreement is not proof. Their decisions follow the explicit-contract/no-compatibility/local-ownership requirements and the source asymmetries documented in B1–B7.
