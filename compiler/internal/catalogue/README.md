# Compiler-owned catalogue

`catalogue.json` is the only authored inventory. `Builtin()` validates embedded
bytes and exposes immutable copies; there is no project-supplied catalogue
loader or registration API. `Generate` produces Go constants, the frozen
TypeScript mirror, the reserved error registry and `std/catalogue/README.md`.
Check mode fails on any stale mirror without changing files.

```sh
go run ./compiler/internal/catalogue/cmd/cataloguegen
go run ./compiler/internal/catalogue/cmd/cataloguegen --check
go test ./compiler/internal/catalogue
CAN_BUN_ARCHIVE=/absolute/path/bun-darwin-aarch64.zip go test -v ./compiler/internal/catalogue -run TestPackagedCatalogueBoundary
```

Descriptor names such as `array.map` are internal lookup keys; the source syntax
remains receiver methods. `Resolve` accepts concrete generic arguments and
checked callback signatures, enforces catalogue constraints and returns their
exact finite domain error union. Standard failures are never included. Project
nominal constraints are established by the later type checker through
`TypeAdmission`; it cannot override opaque catalogue types or primitive key
restrictions. Whole-project ID registry validation belongs to I49/P2.

`RequiredNativeBound` computes fixed, conditional authentication/codec, and
question/handler/body bounds. Authored native declarations must still spell
and satisfy their complete exported bound in I16. `ErrorIdentity` describes an
already checked nominal error specialization; it is not a general runtime
payload constructor or substitute for the type checker. I49 owns occurrence
creation, causes and safe failure rendering.

Each lowering recipe identifies native operations, the contract-specific
adapter and its implementation task. These descriptors do not implement the
future adapters. The exact fixed target/revision is required for lookup; there
is no substitute operation or fallback target. Assertion metadata separates
real computation, supplied external boundaries and scoped fixture/callback
adapters, to be enforced by I12/I18.

The integration fixture is explicit supplied evidence: it checks a domain
error occurrence against both generated views through the I02 sidecar. It is
not a claim that the new compiler or runtime failure handling already exists.
The test also executes runtime catalogue rejection checks with networking
blocked and PATH lacking Bun. CI supplies the pinned archive; ordinary tests
skip this integration check when the explicit archive is absent.
