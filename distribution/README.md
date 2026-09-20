# Native target qualification (I01)

`target.json` freezes the initial upstream Bun archive, extracted executable,
revision, macOS arm64 target, minimum OS, native API inventory, and behavior
probes. Archive bytes were checked against the saved upstream release metadata;
the extracted executable digest was measured independently. Hashes bind bytes
and do not replace publisher-signature or release notarization checks (I39).
macOS 13.0 is the upstream minimum, not a claim that every supported OS release
has been exercised. Each report records the actual OS and runtime component
versions (including ICU and Unicode).

Acquire the exact `upstream.url` archive separately, then run:

```sh
python3 distribution/qualify.py --archive /absolute/path/bun-darwin-aarch64.zip --report /absolute/path/native-capabilities-v1.json
```

The harness never downloads anything or selects Bun from PATH. It checks the
archive and executable digests, creates a temporary versioned layout with
`runtime/bun`, and invokes that absolute executable from an unrelated directory
with an empty home, no ambient runtime configuration, PATH without Bun, and
network access denied by macOS `sandbox-exec`. Sandbox failure is a gate failure;
there is no online fallback. This is a qualification fixture, not the I02
launcher/sidecar installer. The temporary layout is removed after execution.

The report uses schema version 1 and kind `can.native-capability-report`, binds
the manifest and probe source hashes, records each capability and behavioral
result, and fails closed on a missing API or mismatched runtime. The accompanying
nine tests include removal of the actual `JSON.rawJSON` and `Array.fromAsync`
functions and rejection of runtime identity differences. The manifest's API
presence checks do not claim full HTTP, SQL, or language semantics coverage.

CI uploads `native-capabilities-v1` separately from any language conformance
results. The old Ubuntu/Z3 verifier workflow did not qualify the approved target
and has been replaced. This gate does not claim completion of C12/P15 language,
provider, resource, packaging, or application conformance.

Changing the pin requires updating provenance and both hashes, incrementing the
target ID, and rerunning qualification. Never follow `latest` or add API shims.
