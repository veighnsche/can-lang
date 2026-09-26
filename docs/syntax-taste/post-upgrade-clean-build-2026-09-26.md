# Post-upgrade clean-build/run recipe from the installed distribution

26 September 2026 · code candidate `08904291`.

Reproducible operator path: install the release, seed SQLite, build
and serve the invoice server plus grid, exercise both. No test-harness
file is imported or executed anywhere below; every command uses the
installed toolchain, the examples and standard tools (`sqlite3`,
`curl`, a browser).

Recipe status: drafted from the harness staging paths; each command
below was executed against the installed candidate (see the
verification note at the end).

## Prerequisites

- This source tree at the candidate commit.
- The pinned Bun archive for the host target
  (`distribution/target.json` on darwin-arm64; never downloaded by
  any step below).
- Go (to build the launcher), `sqlite3`, `curl`.
- For the browser legs: `node` plus the pinned Playwright browsers
  (`bun ci` in `tests/integration/browser`; Chromium 140.0.7339.186
  and WebKit 26.0).

## 1. Build and install the release

```sh
SRC=/absolute/path/to/can-lang
BUN_ARCHIVE=/absolute/path/bun-darwin-aarch64.zip
go run ./tools/distbuild --archive "$BUN_ARCHIVE" \
  --out /tmp/bundles --version 0.3.0 --release-out /tmp/releases
go run ./tools/distbuild install \
  --archive /tmp/releases/can-0.3.0-*.zip \
  --sha /tmp/releases/can-0.3.0-*.zip.sha256 --root /tmp/can-root
/tmp/can-root/current/bin/canlc runtime-check
```

The installer stages, verifies against the manifest and swings the
`current` symlink atomically; see [distribution instructions](../../distribution/README.md#offline-release-install-and-updates-i39).

## 2. Seed the invoice database

The schema is operator DDL (`examples/invoice/schema.sql`); the seed
rows below are the documented demo tenant (alice/tenant 1/invoice 7
plus a second tenant for nondisclosure checks).

```sh
DB=/tmp/inv.sqlite
rm -f "$DB"
sqlite3 "$DB" < "$SRC/examples/invoice/schema.sql"
sqlite3 "$DB" <<'SQL'
INSERT INTO invoice_session (token, actor, revoked_ms) VALUES ('tok-alice', 'alice', 0), ('tok-bob', 'bob', 0), ('tok-revoked', 'mallory', 1700000000000);
INSERT INTO invoice_membership (actor, tenant) VALUES ('alice', 1), ('bob', 2);
INSERT INTO invoice (tenant, id, revision, seats, details, updated_ms) VALUES (1, 7, 1, 2, 'Acme demo', 900), (2, 8, 1, 1, 'Globex', 800);
INSERT INTO invoice_line (invoice_id, line_key, id, quantity, price_minor, position) VALUES (7, 'k1', 'sku-1', 2, 1999, 0), (7, 'k2', 'plain', 1, 500, 1);
SQL
```

## 3. Build the grid, then the paired server

Build from copies: `build` writes `dist/` into the project directory,
and building the checkout in place would leave outputs behind (and can
confuse later test staging, which requires a pristine source tree).

```sh
CANLC=/tmp/can-root/current/bin/canlc
rm -rf /tmp/invoice-demo && mkdir -p /tmp/invoice-demo
cp -r "$SRC/examples/invoice-grid" /tmp/invoice-demo/invoice-grid
cp -r "$SRC/examples/invoice" /tmp/invoice-demo/invoice
GRID=$(cd /tmp/invoice-demo/invoice-grid && pwd -P)
SERVER=$(cd /tmp/invoice-demo/invoice && pwd -P)
"$CANLC" build --target browser "$GRID" >/tmp/grid-build.json
MANIFEST=$(python3 -c 'import json;print(json.load(open("/tmp/grid-build.json"))["directory"])')/browser/manifest.json
"$CANLC" build --browser-manifest "$MANIFEST" "$SERVER" >/tmp/server-build.json
python3 -c 'import json;r=json.load(open("/tmp/server-build.json"));print(r["buildID"],r["browser"]["browserBuildId"],r["browser"]["entry"])'
```

The second build verifies the browser manifest (build ID, locked
shared-package instances, asset graph and hashes) and records the
exact pairing in its report; a mismatched manifest fails instead of
serving.

## 4. Serve and exercise

Serve the paired server entry directly with the installed Bun: the
entry takes the port as its argument and reads a JSON credential
snapshot on fd 3 (never argv or disk). `canlc run` would rebuild
without the pairing, so it cannot serve the grid page.

```sh
PORT=18501
ENTRY=$(python3 -c 'import json;r=json.load(open("/tmp/server-build.json"));print(r["directory"]+"/entry.ts")')
printf '{"INVOICE_DB":"%s","PUBLIC_ORIGIN":"http://127.0.0.1:%s"}' "$DB" "$PORT" >/tmp/snapshot-paired.json
chmod 600 /tmp/snapshot-paired.json
PATH=/nonexistent HOME=/tmp /tmp/can-root/current/runtime/bun "$ENTRY" "$PORT" 3</tmp/snapshot-paired.json >/tmp/invoice.log 2>&1 &
SRV=$!
sleep 10
curl -sf "http://127.0.0.1:$PORT/health"
```

Form page (server-rendered HTMX; session travels by cookie only):

```sh
curl -sf --cookie 'session=tok-alice' \
  "http://127.0.0.1:$PORT/invoices/form?tenant_id=1&invoice_id=7" | head -c 300
```

JSON round trip (exact `Origin` required; loopback HTTP is the
documented local exception to the HTTPS origin rule):

```sh
curl -sf --cookie 'session=tok-alice' \
  -H "Origin: http://127.0.0.1:$PORT" -H 'Content-Type: application/json' \
  --data '{"operation_id":"op-demo-1","revision":"1","lines":[{"key":"k1","id":"sku-1","quantity":"2","price":"19.99"},{"key":"k2","id":"plain","quantity":"1","price":"5.00"}]}' \
  "http://127.0.0.1:$PORT/api/tenants/1/invoices/7"
```

Grid page (open in a browser; the served script tag is the
report-selected paired asset, CSP `script-src 'self'`):

```text
http://127.0.0.1:18501/invoice-grid?tenant=1&invoice=7
```

Confirm the served entry matches step 3's report before trusting the
page:

```sh
curl -sf --cookie 'session=tok-alice' \
  "http://127.0.0.1:$PORT/invoice-grid?tenant=1&invoice=7" \
  | grep -o -e '<script type="module"[^>]*>'
```

Stop the server with `kill $SRV`. Replaying the same JSON body with
the same `operation_id` returns the recorded outcome without a second
effect; a changed body under the same ID conflicts (409).

## Full staged qualification

The exact UP24 sequence on darwin/arm64 (PostgreSQL 17.11 on
127.0.0.1:5433, pinned Playwright browsers installed), run as six
sequential shards at `-parallel 2` — the shape a 16 GB host
sustains (a single `-parallel 4` tree thrashed into swap
exhaustion with zero completions; the killed attempt is preserved
in the evidence set as history):

```sh
cd "$SRC"
export CAN_BUN_ARCHIVE="$BUN_ARCHIVE"
export CAN_TEST_POSTGRES_URL=postgres://can@127.0.0.1:5433/can_test
export CAN_TSC="$SRC/tscheck/node_modules/typescript/bin/tsc"
export CAN_TEST_HEAVY_SLOTS=2
go vet ./...
go run ./compiler/internal/catalogue/cmd/cataloguegen --check
go run ./tools/modcheck
go run ./tools/gramcheck
CAN_UP23_OUT=/tmp/up23-verdict.json \
  go test ./tests/integration -run TestUP23WriteVerdict -count=1 -timeout 25m
export CAN_UP23_RESULTS=/tmp/up23-verdict.json
go test $(go list ./... | grep -v tests/integration) -count=1 -timeout 30m -v > /tmp/shard0-units.log 2>&1
CAN_BROWSER_EVIDENCE_DIR=/tmp/up24-s1 \
  go test ./tests/integration -run 'TestGate3' -count=1 -timeout 60m -parallel 2 -v > /tmp/shard1-gate3.log 2>&1
CAN_BROWSER_EVIDENCE_DIR=/tmp/up24-s2 \
  go test ./tests/integration -run 'TestInvoiceContract' -count=1 -timeout 60m -parallel 2 -v > /tmp/shard2-contract.log 2>&1
CAN_BROWSER_EVIDENCE_DIR=/tmp/up24-s3 \
  go test ./tests/integration -run 'TestGate4FaultMatrix|TestInvoiceBrowser$|TestInvoiceFormLive|TestInvoiceGridPagePaired|TestInvoiceHTMLFragments|TestInvoiceStartupRefusal|TestInvoiceGridStagedBrowserBuild|TestInvoiceBrowserGuardDOM|TestApplications' -count=1 -timeout 60m -parallel 2 -v > /tmp/shard3-live.log 2>&1
CAN_BROWSER_EVIDENCE_DIR=/tmp/up24-s4 \
  go test ./tests/integration -run 'TestGate5|TestBrowser|TestCurrentBrowserAssets|TestCurrentPairedAssets' -count=1 -timeout 60m -parallel 2 -v > /tmp/shard4-browser.log 2>&1
CAN_BROWSER_EVIDENCE_DIR=/tmp/up24-s5 \
  go test ./tests/integration -run 'TestAssertionFailureLocations|TestBundled|TestChecksMismatchSpan|TestCurrentBundled|TestCurrentCrypto|TestCurrentFormats|TestCurrentMarkdown|TestCurrentMySQL|TestCurrentS3|TestCurrentSQL|TestCurrentSQLite|TestDevelopmentSidecar|TestDistribution|TestFormatPreserves|TestHarness|TestHeavy|TestInstallRoot|TestLinux|TestReleaseInstall|TestStandardSnapshot|TestStdlib|TestUP23|TestWebhook' -count=1 -timeout 60m -parallel 2 -v > /tmp/shard5-rest.log 2>&1
```

Each `go test` must exit 0 before the next starts; together the
six shards cover every package and all 106 `tests/integration`
tests exactly once. The UP23 step stages the paired server and
runs the committed guard legs live in both named engines, binding
the verdict to the exact qualified commit. Residual skips in a
green run: the two Linux installed-artifact tests (deferred UP25,
x86 machine), the live MySQL/S3 tests (unoperated external
services, never claimed), and the `TestUP23WriteVerdict` row
itself, which skips without `CAN_UP23_OUT`. The 26 September logs,
verdict, and browser reports are archived under
[evidence/2026-09-26/](evidence/2026-09-26/up24/README.md).

## Verification note

Executed 26 September 2026 against the installed candidate
(`can-0.3.0-bun-1.4.2-darwin-arm64-v1`): install ok, seed ok (3
sessions, 2 invoices), grid build ok, paired server build ok (browser
`27d76733132b`, entry `/__can/assets/05893185….js`), paired serve ok
(`/health` → `ok`), form page ok (`hx-post="/tenants/1/invoices/7"`),
grid page carries exactly the report-selected entry, JSON save
committed rev 2 then rev 3, identical replay returned the recorded
outcome with no second effect (1 ledger row, revision unchanged).
Copy-based grid rebuild returns the identical build ID
(`b02636607e2c`), confirming same-input identity; project paths must
be canonical (`/tmp` is a symlink on macOS, hence `pwd -P`).
