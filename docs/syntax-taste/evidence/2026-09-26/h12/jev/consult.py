"""Three fresh Jev consultations on H12 paired-deploy staging decisions.

Every explanatory state field, instruction and option description is rewritten
in full across the three rounds. Technical identifiers, task IDs, numeric
facts and code spellings stay exact. No prior Jev answer appears in any
request. Usage: python3 consult.py (generate) ; python3 consult.py --send (3 calls).
"""
from pathlib import Path
import json
import os
import sys
import urllib.request
import datetime
import hashlib
import re

P = Path(__file__).resolve().parent

state = {
'program': [
'Can is an agent-oriented programming language with zero external users, so no backwards compatibility is owed to old syntax, ABI, spellings, generated TypeScript layouts, or goldens. Task H12 (requirement R13 with F-R13-01b/02/03, workload leg W6-deploy, shared interface C-H, owned by release engineering) qualifies the paired deployment on the user native x86 machine (UP25): package, install, network-denied smoke, live PG roundtrip, credentials, health, retained assets, old-browser scenario, rollout and rollback. UP25 machine access is not granted yet, so this turn is staging-only: the runbook, pins, and procedures are written and checked here without executing any linux/amd64 binary or container. Unavailable live evidence never counts as a pass, and no emulated run may support a native claim. Decide from the supplied facts without asking the user.',
'Judge staging choices for Can, a programming language whose consumers are coding agents rather than people. Nobody outside the project uses it, which removes any duty to preserve earlier grammars, binary interfaces, source spellings, emitted TypeScript arrangements, or recorded goldens. Work item H12, drawn from R13 with F-R13-01b/02/03 under workload leg W6-deploy and the C-H interface and held by release engineering, proves the paired server-plus-browser deployment on native x86 hardware (UP25): building, installing, offline smoke, a live PostgreSQL roundtrip, credential handling, health, asset retention, the aged-browser case, plus rollout and rollback. The UP25 grant has not arrived, so current work only stages: procedures and pins are authored and syntax-checked while every linux/amd64 executable and image stays unrunnable on this machine. Missing live proof is never scored as success, and emulation can never back a bare-metal assertion. Reach every selection below from this material alone, posing no question to the user.',
'This choice concerns a language built for AI coding agents, where dependable behavior outranks every other goal. The project has no outside adopters, so keeping prior syntax, interfaces, spellings, generated file shapes, or goldens carries no weight. H12 (sourced from R13 with F-R13-01b/02/03 through W6-deploy and interface C-H, release-engineering owned) closes native x86 qualification (UP25) of paired deployment: release packaging, installation, smoke with networking denied, a live PG roundtrip, secrets, service health, retained routes, the stale-browser scenario, and both rollout and rollback directions. Because the user has not yet designated the machine or granted the window, this turn prepares only: the runbook is written and its static checks pass, but no linux/amd64 program or container starts here. Absent live evidence cannot become a passing leg, and no emulated execution counts toward UP25. Make each call from the evidence given here, without further questions.',
],
'mechanism': [
'The C-H contract (distribution/paired-deploy.md) is fixed: every served document carries its serving generation in data-can-generation; every action sends can-generation; the server pins its manifest buildID at startup and refuses missing, malformed, or unequal values with status 409 and the exact body {"schemaVersion":1,"kind":"can.generation-mismatch","serverGeneration":"<id>"} before any action logic runs; the app answers with a blocking refresh prompt; replaced digest routes stay servable for seven days via dist/assets.json; staging links identical bytes from dist/cas as read-only 0400 hardlinks while manifests stay unique 0600 writes. C06 implemented the runtime slices: the mismatch lowers to http::transport_failed with phase "generation", enforced on JSON plus form mounts of paired builds only, with the blocking prompt live in the invoice-grid app; the invoice-compare app makes no server calls so it has no mismatch path, and the htmx form page refuses with the exact 409 but renders no prompt. The T19 lane (distribution/linux/) packages Bun 1.4.2 for target bun-1.4.2-linux-amd64-v1 on Debian 13+ amd64 with glibc: build.sh releases, smoke.sh installs plus smokes, postgres.sh roundtrips PG 17, and the Go suite (TestLinuxInstalledArtifactSmoke, TestLinuxPostgresRoundtrip) mirrors the matrix using CAN_BUN_ARCHIVE, CAN_LINUX_INSTALL_ROOT, and DATABASE_URL. qualify.py denies networking with unshare -n when available, else honors a caller --network-isolation claim only after verifying no non-loopback interface is up, and fails closed otherwise.',
'Shared interface C-H already pins the handshake shape: rendered pages stamp the serving generation into data-can-generation, each action request repeats it as can-generation, and the service compares against the buildID pinned from its own manifest at boot, rejecting absent, corrupt, or differing values ahead of handler code with a 409 whose body is exactly {"schemaVersion":1,"kind":"can.generation-mismatch","serverGeneration":"<id>"}. Applications respond through a blocking refresh prompt, superseded digest URLs keep serving seven days under the dist/assets.json ledger, staged bytes arrive as 0400 hardlinks out of dist/cas, and manifests remain private 0600 files. Lane C qualification placed the behavior: mismatch outcomes surface as http::transport_failed carrying phase "generation" on JSON and form mounts of paired builds, the invoice-grid app shows the blocking prompt, invoice-compare issues zero server requests so mismatch cannot arise there, and the htmx form surface returns the precise 409 without any prompt. Packaging for bun-1.4.2-linux-amd64-v1 on Debian 13+ amd64 with glibc lives in distribution/linux/: build.sh cuts the release, smoke.sh installs and smokes, postgres.sh exercises PG 17, while TestLinuxInstalledArtifactSmoke and TestLinuxPostgresRoundtrip replay the same legs in Go driven by CAN_BUN_ARCHIVE, CAN_LINUX_INSTALL_ROOT, and DATABASE_URL. For offline proof qualify.py prefers unshare -n, otherwise accepts a caller --network-isolation statement exclusively when live interfaces confirm no uplink, refusing to proceed in any other case.',
'The paired-deploy contract is settled law: documents embed their renderer generation in data-can-generation, actions echo it back in can-generation, and the running server holds the manifest buildID from startup, failing closed ahead of all domain code on omitted, ill-formed, or non-equal values with status 409 and body {"schemaVersion":1,"kind":"can.generation-mismatch","serverGeneration":"<id>"} verbatim. A blocking refresh prompt resolves the app side, replaced digest routes survive seven days through dist/assets.json, content-addressed staging shares bytes as read-only 0400 links from dist/cas, and per-build metadata stays single-owner 0600. Implemented scope from C06: the refusal lowers into http::transport_failed with phase "generation" for JSON plus form mounts on paired builds, invoice-grid renders the blocking prompt, invoice-compare never contacts the server so it cannot mismatch, and the htmx form page emits the exact 409 with no prompt follow-up. The Linux lane targets bun-1.4.2-linux-amd64-v1 on Debian 13+ amd64/glibc via distribution/linux/ (build.sh for release, smoke.sh for install plus smoke, postgres.sh for the PG 17 roundtrip) and the Go mirrors TestLinuxInstalledArtifactSmoke and TestLinuxPostgresRoundtrip keyed off CAN_BUN_ARCHIVE, CAN_LINUX_INSTALL_ROOT, and DATABASE_URL. Offline enforcement in qualify.py uses unshare -n where it works, else demands a caller --network-isolation declaration corroborated by dead uplinks on every live interface, and aborts when neither holds.',
],
'limits': [
'Jev is a classifier, not a researcher: it supplies typed choices with probabilities, never prose reasons or new facts. Its distributions are advisory only, never acceptance proof; these calls execute no Can code, start no linux/amd64 process, and calibrate nothing. The UP25 window checklist demands a native Debian 13+ amd64/glibc host verified by uname, os-release, and glibc probes, with no linux/amd64 emulation anywhere; judge the staging procedures against that bar. Treat each round independently: no request carries any prior answer. Decide every question below from the supplied state alone.',
'The model here classifies rather than investigates: answers are typed selections plus distributions, without rationale text or fresh evidence. Such scores guide engineering judgment but prove nothing; this round runs no compiler, no runtime, no container, and no native leg. Window-entry rules insist on a genuine Debian 13+ amd64/glibc machine confirmed through uname, os-release, and glibc checks, while any linux/amd64 emulation stays forbidden; measure each staged procedure against those gates. Rounds stand alone with no earlier verdict leaking into later state. Ground each answer strictly in the material provided.',
'Expect no research from this consultation: the system returns a chosen option with a probability spread, not explanations or discoveries. Those numbers advise the design and certify no behavior; nothing here executes Can programs, launches x86 binaries, or measures domain accuracy. Entry to the UP25 window requires proving a real Debian 13+ amd64/glibc host via uname, os-release, and glibc evidence, and linux/amd64 emulation is excluded throughout; evaluate the candidate runbook shapes by that standard. Each of the three rounds is self-contained; later requests contain no trace of earlier outcomes. Answer only from the context handed to this call.',
],
}

questions = {
'netdenial': {
'instructions': [
'How should the H12 runbook deny networking for the smoke legs on the bare-metal UP25 host? Judge denial strength, independence of the verification, and honesty when the mechanism is unavailable.',
'Choose the offline-smoke networking shape for a host with no container tooling assumed. Weigh how completely traffic is blocked, what independently confirms the block, and what happens if the tool is missing.',
'Decide the network-denial procedure the runbook prescribes on native x86. Favor the path that blocks all traffic by construction, checks the block separately, and fails closed without quiet fallback.',
],
'criteria': {
'outer_unshare': [
'Wrap the entire smoke invocation in unshare -n at the runbook level, so the Go suite, canlc runs, and qualify.py probes all inherit a namespace with loopback only. qualify.py then independently corroborates the claim by observing no live uplink interface, and its fail-closed branch aborts the leg when neither unshare nor a verified claim holds. When the host forbids user namespaces the leg stays BLOCKED with that cause recorded; it never silently runs on the open host. One mechanism covers every leg process instead of trusting each probe to isolate itself.',
'Execute the whole offline battery inside a single outer unshare -n namespace created by the runbook command itself, giving every child from the test binary down to the Bun probes a loopback-only view. Separately, qualify.py inspects live interfaces and refuses any --network-isolation declaration unless uplinks are verifiably down, halting rather than degrading. If the kernel or policy denies namespace creation, the runbook records BLOCKED for offline legs instead of proceeding connected. Denial therefore precedes the suite and confirmation follows it, with no per-probe self-isolation to trust.',
'Start the smoke suite under one runbook-owned unshare -n wrapper so networking disappears before any leg process exists and every descendant sees loopback alone. The independent check comes after: qualify.py reads the live interface states and rejects the isolation claim unless each non-loopback link is down, failing the leg outright on doubt. Hosts that cannot create the namespace leave the offline legs explicitly BLOCKED with the reason logged, never quietly executed with connectivity intact. Coverage is uniform because the boundary sits above the suite rather than inside individual probes.',
],
'container_native': [
'Reuse the existing T19 container path on the native host: docker run --platform linux/amd64 --network none around build, smoke, and postgres legs. The amd64 image on amd64 hardware involves no emulation, and --network none is a strong, familiar denial. But it contradicts the window preference for no Docker in the qualification path, adds a daemon dependency to the user machine, and risks arch-spoofing doubt in later review despite native execution.',
'Run the documented Docker invocations directly on the x86 box, relying on --network none for denial and native amd64-on-amd64 execution to avoid emulation. Strength comes from a well-worn flag, yet the checklist prefers zero Docker involvement, the host must provide a daemon, and future readers must re-verify that no image-arch mismatch hid under the platform flag. Convenience trades against the explicit bare-metal posture.',
'Keep the T19 Docker orchestration unchanged and execute it on the UP25 machine, taking --network none as the isolation boundary with natively matching architectures. Denial is robust and the scripts already fit it, but the window record favors keeping Docker out entirely, demands a container runtime on user hardware, and invites later suspicion about which architecture truly ran. The path is easy and legible, though estranged from the stated no-container preference.',
],
'open_host_observe': [
'Run the smoke on the open host and rely on qualify.py interface observation plus log review to argue nothing dialed out. This inverts the burden: connectivity stays available throughout, the live-interface check cannot pass on a connected machine so the caller claim must be weakened or skipped, and absence of observed traffic is a far weaker statement than impossibility of traffic. A passing run proves the legs tolerate connectivity, not that they survive without it.',
'Execute offline legs with uplinks up and afterwards claim isolation from clean logs and probe reports. The interface gate then fails by design on any connected host, forcing the runbook to dilute or drop the claim it was meant to verify, while reviewers receive behavioral absence instead of structural denial. Success under these terms shows the suite happens not to phone home, which is not the offline property H12 must qualify.',
'Leave host networking intact during the smoke and substitute post-hoc observation — log scans and probe output — for real denial. Because qualify.py fails closed when interfaces are live, the runbook would have to bypass or soften the very check that guards the leg, and the resulting evidence attests that no connection was noticed rather than that none was possible. The leg would pass in exactly the configuration it was supposed to exclude.',
],
},
},
'browserscenario': {
'instructions': [
'Which vehicle should carry the old-browser scenario on the UP25 box? Judge end-to-end honesty, fit with implemented app behavior, and whether each claimed observation is actually executed on x86.',
'Choose how the stale-browser leg runs natively. Weigh genuine browser execution against replay fidelity, match each candidate to the app that truly exhibits the behavior, and reject any leg that borrows observations from another machine.',
'Decide what executes the aged-page case during the window. Prefer real interactions on the built x86 pair, align the scenario with the app that owns the prompt, and refuse credit for behavior proven elsewhere.',
],
'criteria': {
'grid_playwright': [
'Drive the real invoice-grid pair on the UP25 host with a pinned Playwright Chromium: open the grid page from generation N, roll out generation N+1, trigger a save from the stale page, observe the exact 409 can.generation-mismatch refusal and the blocking refresh prompt, refresh, and confirm the action then succeeds. Retained digest URLs from N are fetched directly and must still return 200. Every observation — refusal shape, prompt, retained bytes — executes against x86-built generations in a genuine browser. Cost is a browser download plus a new provision row during the window, which the runbook stages explicitly.',
'Run the full stale-page story in an actual pinned Chromium on the native box against the invoice-grid deployment: load generation N in the browser, promote to N+1 underneath, act from the aged tab, watch the typed 409 refusal surface as the blocking refresh, reload into the new generation, and re-run the action to success. Replaced asset URLs from the old generation are requested raw and must answer 200. Refusal bytes, prompt behavior, and retention all come from live x86-built artifacts under real browser execution. The runbook pays openly for this with a staged browser fetch and provision entry inside the exclusive window.',
'Execute the aged-browser narrative end to end on UP25 with Playwright Chromium pinned: render generation N of invoice-grid, roll the server to N+1, submit from the outdated page, verify the exact can.generation-mismatch 409 becomes the blocking refresh prompt, refresh the tab, and see the retried action pass. Old digest routes are curled straight and must serve 200. No observation is imported: mismatch bytes, prompt rendering, and retained assets are all witnessed on the native pair through a true browser. Browser acquisition and provisioning are declared runbook steps inside the window, not hidden prerequisites.',
],
'header_replay': [
'Prove the server-side legs with header replay instead of a browser: fetch the served page, read data-can-generation, issue the action with a stale can-generation value via curl, assert the byte-exact 409 can.generation-mismatch body, and fetch replaced digest URLs for 200s. This covers refusal shape and retention honestly on x86, but the blocking refresh prompt — a browser-executed behavior — stays unobserved on the native pair. As the sole scenario it silently drops the app half of the C-H contract.',
'Replace browser execution with curl replays of the wire contract: extract the generation slot from served markup, send actions carrying an outdated can-generation header, compare the 409 body byte for byte, and confirm retained assets serve. Server refusal and retention earn genuine native evidence, yet the prompt that users actually see never renders during the window. Standing alone, this vehicle qualifies the wire while leaving the documented app behavior unexercised on x86.',
'Exercise only the HTTP surface: read data-can-generation out of a fetched document, replay actions with a superseded can-generation header, check the 409 payload exactly, and verify old digest URLs still return bytes. Refusal and retention legs gain honest x86 proof, but the blocking refresh — which exists solely as rendered app behavior — never appears in the evidence. Adopted alone, the replay path quietly abandons the client half of the handshake it claims to qualify.',
],
'split_claim': [
'Run header replay on x86 for refusal and retention, then cite the C06 Mac-matrix prompt evidence for the blocking refresh and mark the whole scenario green. The prompt code that would render on UP25 ships inside x86-built bundles that C06 never executed, so the borrowed observation covers different bytes on a different target. Combining legs across machines into one verdict hides exactly the native gap H12 exists to close.',
'Qualify the wire natively and import the prompt verdict from C06 runs on other hardware, presenting the merged rows as full UP25 coverage. C06 exercised darwin-built bundles, while the UP25 prompt would execute from freshly built x86 artifacts — untested bytes under an untested build. The split verdict papers over the one behavior the native window uniquely owes: the prompt as served by the qualified pair.',
'Merge x86 replay legs with off-target prompt results from the C06 matrix and report the scenario complete. Because bundle bytes and build provenance differ per target, evidence from the Mac matrix cannot vouch for prompt behavior in the native deployment. This stitching converts two partial truths into a whole claim no single machine ever demonstrated, defeating the purpose of a native leg.',
],
},
},
'servicehealth': {
'instructions': [
'What operating shape should serve the lifecycle legs on the user machine? Judge credential safety, cleanup burden on user hardware, and whether health genuinely reflects the paired build.',
'Choose the service posture for rollout, rollback, health, and secrets during the window. Weigh env-only credential discipline, leftover state on the user box, and how directly health proves the serving generation.',
'Decide how the runbook hosts the deployed pair for its lifecycle evidence. Favor secret-free artifacts, full removal afterwards, and a health signal tied to the served generation itself.',
],
'criteria': {
'supervised_process': [
'Serve each generation as a runbook-supervised foreground process under a fresh install root: secrets arrive only through process env (DATABASE_URL and pool names, never files or evidence), a bounded health poll curls the served page and requires the current data-can-generation plus one successful action roundtrip, and rollout/rollback proceed through the paired-deploy recipe of rebuild plus atomic selection. Teardown kills the processes and deletes the run roots; nothing registers with init, and the box keeps no service state. Health attests the live pair, not a proxy or a stale socket.',
'Host the lifecycle legs with plain supervised processes owned by the runbook session: credentials flow exclusively via environment variables into the server process, health means fetching the served document and matching its embedded data-can-generation to the expected build alongside a passing action, and promotion in either direction follows rebuild with atomic selection. When the window ends, the runbook stops every server and removes its directories, leaving no unit files, timers, or enabled services behind. The health check reads the pair own generation stamp, so it cannot pass against the wrong deployment.',
'Run deployed generations as transient children of the runbook shell inside dedicated directories: env-only secrets passed at spawn, a polling health gate that demands the served page stamp the expected data-can-generation and complete one action, and rollout/rollback through the standard rebuild-and-select recipe. Cleanup terminates the servers and wipes the run trees, so user hardware retains no service registrations or residue. Because health inspects the generation embedded by the serving build, a mismatched or ghost process cannot satisfy it.',
],
'systemd_unit': [
'Install a real systemd unit (user or system scope) for the served pair to maximize lifecycle fidelity: enable, start, health-check, roll, roll back, then disable and remove. This exercises genuine service management but mutates user init state, needs privilege decisions and careful teardown, risks leftover units or journal noise on failure paths, and couples the qualification to one init system. Any missed cleanup step leaks production-shaped state onto borrowed hardware.',
'Register the deployment as a systemd service during the window for full-fidelity lifecycle proof, with enable/start/stop/disable bracketing the legs. Fidelity comes at the cost of touching the user init configuration, negotiating user versus system scope and privileges, scrubbing units plus lingering state afterwards, and assuming systemd present. A teardown miss strands service artifacts on a machine the project does not own.',
'Prove lifecycle through an installed systemd unit with the full enable/activate/roll/deactivate/remove cycle. The realism is real, yet so is the intrusion: init state changes on borrowed hardware, privilege and scope choices multiply, failed-path residue (units, symlinks, logs) needs its own recovery procedure, and the evidence binds to systemd specifically. Each lifecycle gain arrives with a cleanup liability.',
],
'no_service': [
'Skip serving entirely: build, install, and smoke binaries without ever running the pair as a service, asserting health implicitly from install success. This leaves credentials handling, health definition, rollout/rollback, and the operator-DDL roundtrip unexercised — the core of F-R13-02. A green report would then certify deployment qualification while the deployment itself never ran.',
'Limit the window to build/install/smoke artifacts and declare lifecycle covered by construction, never starting the paired server or defining health. Credential flow, generation-stamped health, promotion in both directions, and operator DDL against the live pair all stay theoretical. The resulting verdict would bless a deployment leg whose subject never served traffic.',
'Forgo the service legs and treat packaged artifacts as sufficient proof, with no running pair, no health signal, and no secret flow. Rollout, rollback, credential provisioning, health, and live operator DDL — the substance of the lifecycle recipe — would pass unexecuted. Evidence would describe a release that was built but never operated, which is not what H12 promises to qualify.',
],
},
},
}

for entries in state.values():
    assert len(entries) == 3 and len(set(entries)) == 3, "each state field needs 3 distinct phrasings"
for q in questions.values():
    assert len(q['instructions']) == 3 and len(set(q['instructions'])) == 3, "each question needs 3 distinct instructions"
    for entries in q['criteria'].values():
        assert len(entries) == 3 and len(set(entries)) == 3, "each option must have 3 distinct phrasings"


def word_grams(text, n):
    words = text.split()
    return {" ".join(words[i:i + n]) for i in range(max(0, len(words) - n + 1))}


problems = []
groups = []
for name, entries in state.items():
    groups.append((f"state:{name}", entries))
for qname, q in questions.items():
    groups.append((f"question:{qname}:instructions", q['instructions']))
    for oname, entries in q['criteria'].items():
        groups.append((f"question:{qname}:option:{oname}", entries))
for label, entries in groups:
    for a in range(3):
        for b in range(a + 1, 3):
            shared = word_grams(entries[a], 8) & word_grams(entries[b], 8)
            if shared:
                problems.append({"group": label, "pair": [a + 1, b + 1],
                                 "shared_8grams": sorted(shared)[:5]})

for i in range(3):
    payload = {
        'model': 'jev-latest',
        'state': {k: v[i] for k, v in state.items()},
        'questions': {
            k: {
                'type': 'choice',
                'instructions': q['instructions'][i],
                'criteria': {key: v[i] for key, v in q['criteria'].items()},
            } for k, q in questions.items()
        },
    }
    (P / f'request-{i+1}.json').write_text(json.dumps(payload, indent=2) + '\n')

hashes = {}
for i in range(1, 4):
    body = (P / f'request-{i}.json').read_bytes()
    hashes[f'request-{i}.json'] = hashlib.sha256(body).hexdigest()
(P / 'wording-audit.json').write_text(json.dumps({
    'review': 'Manually checked equal facts, constraints and alternatives across all three requests before sending. Each of the 3 state fields, 3 instructions and 9 option descriptions has a distinct complete phrasing (30 fields). Task IDs (H12/UP25/C06/H10), requirement IDs (R13, F-R13-01b/02/03, W6-deploy), interface C-H, file paths, kind strings, function names, test names, env names, the 409 shape, 0400/0600 modes, the seven-day bound and code spellings are intentionally stable. No response or favored recommendation appears in any later request state.',
    'same_facts': [
        'Can agent language; zero external users; no compat for syntax/ABI/spellings/TS layouts/goldens; H12 paired-deploy qualification under R13 F-R13-01b/02/03 W6-deploy C-H, release-engineering owned; package/install/offline smoke/live PG/credentials/health/retained assets/old-browser/rollout/rollback on native UP25 x86; staging-only turn, no linux/amd64 execution here; unavailable live evidence never a pass; no emulated native claim',
        'C-H fixed: data-can-generation in documents, can-generation on actions, startup buildID pin, 409 exact can.generation-mismatch body before action logic, blocking refresh, seven-day dist/assets.json retention, dist/cas 0400 links, metadata unique 0600; C06 scope: http::transport_failed phase generation on JSON+form paired mounts, prompt live in invoice-grid, invoice-compare no server calls, htmx form 409 without prompt; T19 distribution/linux build/smoke/postgres + TestLinuxInstalledArtifactSmoke/TestLinuxPostgresRoundtrip via CAN_BUN_ARCHIVE/CAN_LINUX_INSTALL_ROOT/DATABASE_URL; qualify.py unshare -n else verified --network-isolation claim else fail closed',
        'Limits: Jev classifies, no research; advisory only, no acceptance proof; no Can code runs, no x86 process starts, no calibration; UP25 checklist demands native Debian 13+ amd64/glibc via uname/os-release/glibc, no linux/amd64 emulation; rounds independent, no prior answer leaks; decide from supplied state',
    ],
    'limit': 'Text inequality plus manual semantic review cannot prove absence of framing effects; investigate disagreement and treat agreement as advice, not proof.',
    'shared_8gram_violations': problems,
    'request_sha256': hashes,
}, indent=2) + '\n')
print(json.dumps({'generated': hashes, 'shared_8gram_violations': problems}, indent=2))

if '--send' in sys.argv:
    if problems:
        print(json.dumps({'refused': 'shared 8-gram violations present; reword before sending'}),
              file=sys.stderr)
        sys.exit(1)
    key = os.environ['TYPESAFE_API_KEY']
    for i in range(1, 4):
        req = urllib.request.Request(
            'https://api.typesafe.ai/v1/systemone',
            data=(P / f'request-{i}.json').read_bytes(),
            headers={'Content-Type': 'application/json', 'Authorization': 'Bearer ' + key},
        )
        start = datetime.datetime.now(datetime.timezone.utc).isoformat()
        with urllib.request.urlopen(req, timeout=60) as r:
            body, status = r.read(), r.status
        (P / f'response-{i}.json').write_bytes(body + b'\n')
        (P / f'response-{i}.metadata.json').write_text(json.dumps(
            {'startedAt': start, 'status': status, 'endpoint': 'v1/systemone'}, indent=2) + '\n')
        print(json.dumps({'request': i, **json.loads(body)}))
