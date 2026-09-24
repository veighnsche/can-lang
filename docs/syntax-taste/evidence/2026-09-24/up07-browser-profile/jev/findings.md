# UP07 Jev findings (worker consultation, 2026-09-25)

Question: sealed portable browser runtime profile mechanism.
Model: jev-1.13.0 via v1/systemone. Three fresh reworded requests; full
request/response bytes saved alongside this note. Wording audit in
wording-audit.json.

## Choices (all three requests, confidence 1.0 each)

- profile_structure: seam_plus_overlay (unanimous)
- identity_sealing: async_startup_verify (unanimous)
- assert_surface: assert_free_variants (unanimous)

## Disagreements

None. No option split across rewordings.

## Treatment

Advice only, not proof. The decisive evidence remains executable: Bun
trap-free pins (invoked/traps===0), the Array.isArray look-through probe,
the no-trap-free-proxy-detection proof, and the selected-behavior packet
(no second runtime, no Node-alias shims, no handwritten hash, assertion
context outside the shipped profile). All of it independently selects the
same three options. Implementation proceeds on that basis; conformance
tests below verify the built profile rather than the consultation.
