# F05 Jev consultations — carrier protocol shape

Three fresh consultations (jev-1.13.0), independently worded, sent 2026-09-26.
Requests, responses, metadata and the wording audit sit beside this file.
Advice only: agreement does not prove correctness and was checked against the
decided P09-B3 contracts before adoption.

## Questions

- `envelope`: body-carried `timestamp_ms`+`nonce` in the signed JSON body
  (`body_fields`) vs header-carried canonical signature (`header_canonical`)
  vs bearer secret plus nonce (`bearer_nonce`).
- `claim`: single-row claim with companion paging (`single_page`) vs
  multi-row claim per request (`multi_batch`).
- `ack`: ownership-free idempotent ack by `delivery_id` (`idempotent_open`)
  vs lease-ownership-checked ack (`ownership_checked`).

## Outcome (unanimous)

| Question | 1 | 2 | 3 |
| --- | --- | --- | --- |
| envelope | body_fields 0.76 | body_fields 0.98 | body_fields 0.75 |
| claim | single_page 0.96 | single_page 0.99 | single_page 0.95 |
| ack | idempotent_open 0.97 | idempotent_open 0.63 | idempotent_open 0.93 |

Adopted: `body_fields` + `single_page` + `idempotent_open`.

## Disagreement notes

- Consultation 2 splits ack 0.63/0.37 toward `ownership_checked`: attribution
  is genuinely tighter when the reporter must hold the lease. `idempotent_open`
  still wins because the decided guarantee is at-least-once with downstream
  `delivery_id` dedup, and the state machine never un-delivers; ownership
  checks would turn every late or post-expiry ack into a reclaim flow with no
  safety gain. Heartbeat and dead_letter keep ownership checks, so stolen
  leases still cannot extend or kill another worker's row.
- Consultations 1 and 3 give `header_canonical` ~0.21: binding method+path is
  real defense in depth. `body_fields` wins because it reuses the already
  proven provider verification shape with no canonical assembly, and every
  field the Can side acts on (including the op fields) sits inside the signed
  bytes; routing tamper cannot change the decided outcome, only which
  authenticated handler runs, and each handler revalidates its own body.
- `bearer_nonce` never exceeds 0.03: no per-message integrity is a real loss
  even under loopback/trusted scope. Not adopted.
