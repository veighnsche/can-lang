# UP23 served browser/runtime and invoice matrix — evidence

Task UP23 (lane B). All legs run against application-served pages and
content-addressed assets from compiler-paired builds (`--browser-manifest`);
no test-authored page, bundle, boot entry or engine shim stands on any
claimed path. Both required named browsers execute every leg; no
unavailable-engine skip counts.

## Matrix record (2026-09-25)

- Host: staged bundle on darwin/arm64, bun 1.4.2 rev 744846f84437,
  archive sha256 90987a3a16d7 (`CAN_BUN_ARCHIVE=/private/tmp/bun-dl/bun-darwin-aarch64.zip`)
- Required engines: Chromium 140.0.7339.186, WebKit 26.0
- Server: 344 assertions, 344 real-can
- Grid browser build da892b8195f0 (336 roots) paired 237540fe575f,
  entry `/__can/assets/cc59f68ced3b35b555f1145141de39e18fb9ff23754223c147dfaf19ad88f13b.js`
- Fixture browser build 50650e5a94dc (39 roots) paired 2bc6974aa164,
  entry `/__can/assets/b9265cc703eb27a0a1e0c4e6864a3a18466168866148a815bdd2c918fc317e17.js`
- Empty browser build 7b235fb5d4be (1 root) paired f833c34861e9,
  entry `/__can/assets/da58c0c3ee36a3cacd148fa048d0fc10411862c748043bbbb532f9f95a6307f3.js`

Paired entry digests are stable across stagings (content-addressed over
emitted bytes); build IDs incorporate the toolchain input hash, so a
fresh toolchain rebuild renames builds without changing served bytes.

## Legs

| directory | suite | checks | final state |
|---|---|---|---|
| grid-chromium / grid-webkit | invoice grid, served pairing | 32 / 32 | rev 16, 15 replay rows (revs 2–16), 32 ledger calls |
| conformance-chromium / conformance-webkit | query fixture, served pairing | 17 / 17 | database untouched (rev 1, 0 replays) |
| empty-chromium / empty-webkit | minimal app, served pairing | 4 / 4 | database untouched; DOM untouched |
| invoice-chromium | form guards, grid-paired server | 19 / 19 | rev 5 "Guard Final", 4 replays, 5 occurrences |
| invoice-webkit | form guards, grid-paired server | 19 / 19 | rev 5 "Guard Final", 4 replays, 4 occurrences |
| codec-chromium / codec-webkit | wire-codec parity vs bun reference | all vectors agree | sealed-overlay bundle, no node edges |

Each directory holds `report.json` (machine-readable checks, requests,
ledger/occurrences, console/page faults) and `screenshot.png`, except the
codec directories which hold `parity.json` plus the executed `vectors.js`.

Firefox is best-effort only and not installed on this host (Playwright
firefox-1490 missing); the codec subtest records the skip. Both required
engines ran every leg.

## Pinned limitations

- L-keystroke-eaten, L-flight, L-double-render (grid, both engines)
- L-redirect-webkit (invoice webkit leg only; redirect branch qualified
  end to end on Chromium)

## Reproduce

```sh
CAN_BUN_ARCHIVE=/private/tmp/bun-dl/bun-darwin-aarch64.zip \
  go test ./tests/integration/ -run 'TestGate5ServedMatrix' -count=1 -timeout=55m
CAN_BUN_ARCHIVE=/private/tmp/bun-dl/bun-darwin-aarch64.zip \
  go test ./tests/integration/ -run 'TestGate5GridStatic|TestBrowserBuildTarget|TestBrowserWireCodecParity' -count=1 -timeout=50m
```

Set `CAN_BROWSER_EVIDENCE_DIR` to re-archive the reports. Full verbose
logs for this record live outside the repo (`/private/tmp/up23matrix.log`,
`/private/tmp/up23adjacent.log`); the committed reports above are the
preserved evidence for UP24/25.
