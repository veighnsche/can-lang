# I32 response conversion and route collisions

All three fresh requests selected private immutable response descriptions and
normalization-aware collision rejection. Response confidence was 0.99, 0.98,
and 0.66; the third request assigned 0.21 probability to consuming the Can
token. This reduced confidence prompted an explicit contract check: P10 calls
the response immutable and specifies conversion after a callback completes;
it does not declare a consumable response resource. Therefore conversion is
once per completed dispatch, with a new native body each time. Reuse tests
consume two independent native responses made from the same immutable value.
Caching a native Response would instead share its consumable stream.

Route advice selected normalization-aware collisions in all three requests.
Use the same native URL and decoding sequence at registration and ingress.
Identical same-method spellings are duplicates; distinct same-method spellings
with one decoded normalized path are ambiguous. GET and POST at the same path
remain distinct registrations. This interpretation exercises both declared
collision errors without allowing order-dependent route replacement. No
implicit HEAD, captures or wildcard matching is introduced.

These judgments are design advice, not verification. Runtime, compiler and
staged integration checks remain required before I32 can be marked complete.
