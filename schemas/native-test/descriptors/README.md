# K17 descriptor and environment contract (version 1)

K17 pins the wire fields for managed child launches: the exact fd
0/1/2/3/4 wiring per launch and the distinct C CLI versus direct-entry
environments. It implements no launcher and qualifies no behavior; later
descriptor lanes (K18/K19, QF1/QF2) build fixtures and qualification on
these shapes.

## Roles

`N` (native owner) owns every descriptor in the map, including the
`ExtraFiles` pair (fd 3 snapshot pipe, fd 4 inherited lease). `C` is the
child under test and holds descriptors only as wired by N. There is no
other owner identity on the wire.

## Files

| Schema | Validates |
| --- | --- |
| `descriptor-map.schema.json` | One launch: run/launch ids, fresh snapshot binding, exactly fds 0–4 with direction, byte/EOF/lifetime states and byte counts |
| `environment.schema.json` | One launch environment: `c-cli` or `direct-entry` entry, bounded argv, sorted env names plus binding digest, fd-routed log sink |

Each file is self-contained (shared `id` and `digest` shapes are repeated
per file) so consumers can adopt one envelope without a cross-file
resolver, following the P01 convention.

## Wire conventions

- Every wire field is `snake_case`. `schema_version` is the constant
  `"1"` on every envelope; unknown versions are rejected.
- `run_id`, `launch_id`: `^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`.
- Digests: `sha256:<64 lowercase hex>`. Paths alone are never identities.
- Objects reject unknown fields (`additionalProperties: false`).
- Fixed fd contract (child-side directions): fd 0 `in` (stdin), fd 1
  `out` (stdout), fd 2 `out` (stderr), fd 3 `in` (N-owned snapshot
  pipe), fd 4 `hold` (N-owned inherited lease, lifetime only, no bytes).
- `byte_state`: `none` (fd 4 only), `streaming`, `exhausted`.
  `eof_state`: `none` (fd 4 only), `open`, `eof-sent`, `eof-observed`,
  `closed`. `lifetime`: `active`, `released`.
- Snapshot `state` is the constant `"fresh"`; every launch document
  carries its own `launch_id` and `generation_digest`.
- Environments carry argv and env *names* only, plus `env_digest` binding
  the materialized environment. Values, secrets and log paths are
  unrepresentable: `log_sink` is `fd1`, `fd2` or `none`.

## Semantic rules (checked by `validate.py` beyond shape)

Descriptor map:

1. `descriptors` lists fds 0,1,2,3,4 exactly once; fd numbers above 4
   are already rejected by shape, so no private owner fd can appear.
2. Each fd carries its fixed direction (0/3 `in`, 1/2 `out`, 4 `hold`).
3. fd 4 carries no byte/eof state and zero byte counts.
4. Fds 0–3 carry tracked (non-`none`) byte and EOF states.
5. `bytes_accepted <= bytes_offered` on every entry.
6. `exhausted` bytes require sealed EOF (`eof-sent`, `eof-observed`
   or `closed`); `closed` EOF requires `exhausted` bytes.
7. `released` lifetime requires `closed` (or `none` for fd 4) EOF.

Environment:

8. `c-cli` entry requires non-empty `argv` (argv0 present);
   `direct-entry` requires empty `argv`.
9. `env_names` are unique and sorted (canonical binding order).

## Acceptance mapping

- *No private owner fd*: fd range 0–4 is pinned by shape, the exact
  five-entry set by rule 1, and `owner` is the constant `"N"` (N owns
  ExtraFiles); any other owner or fd is rejected.
- *No secret argv/log*: argv items and env names are bounded and
  names-only; no value/secret/log-path field exists and unknown fields
  are rejected, so secret material cannot ride the wire.
- *Each launch has fresh snapshot state*: `snapshot.state` is the
  constant `"fresh"` and each map carries its own `launch_id` plus
  `generation_digest`; reused state is unrepresentable.

## Checks

Run the fixture suite (valid documents accepted, invalid documents
rejected with the expected reason):

```sh
python3 schemas/native-test/descriptors/validate.py
```

`fixtures/valid/*.json` are raw wire documents.
`fixtures/invalid/*.json` are raw wire documents that must be rejected;
`fixtures/invalid/manifest.json` maps each file to its schema and a
required error substring. The validator mirrors the P01 `validate.py`
pattern (documented JSON Schema subset, standard library only, loud
failure on unsupported keywords). No live host checks are run here;
host-dependent controls wait for qualified profiles.
