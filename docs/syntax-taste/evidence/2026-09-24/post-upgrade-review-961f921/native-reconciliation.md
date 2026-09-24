# Native audit evidence and calibration follow-up

The three probe programs and their unedited stdout, stderr, and exit statuses are saved in `/tmp/can-upgrade-native-961f921-evidence/`. Run each with `bun /tmp/can-upgrade-native-961f921-evidence/<name>.ts`. All exited 0 with empty stderr:

| Probe | Raw stdout | Interpretation |
| --- | --- | --- |
| `empty-race.ts` | `timed out` | `settle("race", [])` remained pending past the probe's 50 ms observation timer. The repo's own test deliberately pins this native `Promise.race([])` behavior. This is an intentional semantic hazard for dynamic spreads, not a compiler defect. Prefer a nonempty/guard helper first; changing empty-race completion is a separate design decision. |
| `owner-latency.ts` | `selected_ms 2` / `root_ms 83 kind ok` | An immediate `Promise.any` winner selected promptly, while `runOwnedRoot` waited for an 80 ms loser. This is a root-scope measurement, not a live HTTP measurement. HTTP impact is inferred from the request child scope in `runtime/platform/server.ts:219-244`. |
| `immutable-set-scaling.ts` | `immutable_set 10000 184` / `immutable_set 20000 649` / `immutable_set 40000 2527` | Point updates clone the native Set; indicative local timings grow approximately quadratically. |

I updated `/tmp/can-upgrade-native-961f921.md` to make those distinctions and to record the coordinator's later exact suite results. No production files changed.
