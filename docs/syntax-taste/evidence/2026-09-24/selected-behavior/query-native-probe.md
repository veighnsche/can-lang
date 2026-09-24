# Native query decoding check

24 September 2026. I ran `bun -e` in this checkout against four
`URLSearchParams` inputs. This is a native API observation, not a Can browser
qualification.

| Raw query | `getAll("invoice")` |
| --- | --- |
| `?invoice=%ZZ` | `["%ZZ"]` |
| `?invoice=%FF` | `["�"]` |
| `?invoice=a+b` | `["a b"]` |
| `?invoice=7&%69nvoice=8` | `["7","8"]` |

The first two values show why a checked `browser::query_parameter` cannot
rely on `URLSearchParams` to reject malformed percent escapes or UTF-8.
Before that native parser, the selected adapter bounds and scans every raw
pair, replaces `+` with space, then calls native `decodeURIComponent` on
each key/value. It uses `URLSearchParams` for the resulting pair semantics
and `getAll` for duplicate detection, including encoded-key aliases. The
full policy and acceptance cases are in the
[selected behavior packet](../../../post-upgrade-selected-behavior-2026-09-24.md#browser-build-and-event-execution).
