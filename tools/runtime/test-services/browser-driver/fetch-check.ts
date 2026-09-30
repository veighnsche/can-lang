// K14 bounded self-check: typed page-realm Fetch with page policy.
//
// Run with: bun tools/runtime/test-services/browser-driver/fetch-check.ts
// Local controls only. No network, sockets, fetch, files, timers, live
// hosts, or live runtimes. Every loop below is bounded by a small constant.
// Live host-dependent controls wait for the qualified profile and Q task.

import { strict as assert } from "node:assert";
import {
  BROWSER_FETCH_CODES,
  BROWSER_FETCH_INDEPENDENCE_SCOPE,
  BROWSER_FETCH_SCHEMA_VERSION,
  BrowserFetchError,
  checkFetchLimits,
  digestPageFetch,
  FETCH_CREDENTIALS,
  FETCH_DECODING_SCOPES,
  FETCH_METHODS,
  FETCH_REDIRECTS,
  FORBIDDEN_EVAL_KEYS,
  layerOfFetchCode,
  PAGE_FETCH_REALM,
  PageFetchService,
  type BrowserFetchCode,
  type BrowserFetchLayer,
  type BrowserFetchLimits,
} from "./fetch.ts";
import { HTTP_METHODS } from "../http-peer/http.ts";

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectFetchError(
  body: () => unknown,
  code: BrowserFetchCode,
  layer: BrowserFetchLayer,
): void {
  try {
    body();
  } catch (error) {
    if (!(error instanceof BrowserFetchError)) {
      assert.fail(`expected a BrowserFetchError, got ${String(error)}`);
    }
    assert.equal(error.code, code);
    assert.equal(error.layer, layer);
    assert.ok(error.message.includes(`[${layer}:${code}]`));
    return;
  }
  assert.fail(`expected failure ${layer}:${code}`);
}

const BODY_DIGEST = `sha256:${"cd".repeat(32)}`;
const PAGE_ORIGIN = "https://page.example";
const PEER_ORIGIN = "https://peer.example";

function roomyLimits(): BrowserFetchLimits {
  return checkFetchLimits({
    maxPages: 4,
    maxFetchesPerPage: 8,
    maxCookiesPerPage: 8,
    maxBodyBytes: 1024,
    maxRedirectHops: 3,
  });
}

function openService(csp: "connect-self-only" | "deny-all" = "connect-self-only"): {
  service: PageFetchService;
  token: string;
} {
  const service = new PageFetchService(
    [{ owner: "owner-n", page: "page-a", pageOrigin: PAGE_ORIGIN, csp }],
    [PAGE_ORIGIN, PEER_ORIGIN],
    roomyLimits(),
  );
  service.bindPage("owner-n", "page-a");
  const token = service.pageTokenForTest("owner-n", "page-a");
  return { service, token };
}

function fetchDescriptor(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    method: "GET",
    targetOrigin: PAGE_ORIGIN,
    targetPath: "/data",
    credentials: "same-origin",
    redirect: "manual",
    ...overrides,
  };
}

function responseDescriptor(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    status: 200,
    decodingScope: "utf8-strict",
    byteLength: 12,
    bodyDigest: BODY_DIGEST,
    location: null,
    ...overrides,
  };
}

// --- Page admission-first ----------------------------------------------------

check("page-admission-first", () => {
  const service = new PageFetchService(
    [{ owner: "owner-n", page: "page-a", pageOrigin: PAGE_ORIGIN, csp: "connect-self-only" }],
    [PAGE_ORIGIN],
    roomyLimits(),
  );
  const binding = service.bindPage("owner-n", "page-a");
  assert.equal(binding.realm, "page-fetch");
  assert.equal(binding.owner, "owner-n");
  assert.equal(binding.page, "page-a");
  assert.equal(binding.pageOrigin, PAGE_ORIGIN);
  assert.equal(binding.csp, "connect-self-only");
  assert.ok(binding.handleDigest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(binding).sort(), [
    "csp",
    "handleDigest",
    "owner",
    "page",
    "pageOrigin",
    "realm",
  ]);

  // Rebinding joins the same owned page; the digest is stable.
  const again = service.bindPage("owner-n", "page-a");
  assert.equal(again.handleDigest, binding.handleDigest);

  // Pages outside the owned grant reject before any effect.
  for (const foreign of ["page-z", "", "page-a/x"]) {
    expectFetchError(() => service.bindPage("owner-n", foreign), "unknown-page", "page");
  }
  expectFetchError(() => service.bindPage("owner-x", "page-a"), "unknown-page", "page");

  // A missing page blocks Fetch, whatever the reason: never bound,
  // foreign owner, or closed.
  const missing = new PageFetchService(
    [{ owner: "owner-n", page: "page-a", pageOrigin: PAGE_ORIGIN, csp: "connect-self-only" }],
    [PAGE_ORIGIN],
    roomyLimits(),
  );
  expectFetchError(
    () => missing.fetchBegin("owner-n", "page-a", "page-deadbeef", "f1", fetchDescriptor()),
    "no-page",
    "page",
  );
  missing.bindPage("owner-n", "page-a");
  const token = missing.pageTokenForTest("owner-n", "page-a");
  expectFetchError(
    () => missing.fetchBegin("owner-x", "page-a", token, "f1", fetchDescriptor()),
    "no-page",
    "page",
  );
  missing.closePage("owner-n", "page-a");
  expectFetchError(
    () => missing.fetchBegin("owner-n", "page-a", token, "f1", fetchDescriptor()),
    "no-page",
    "page",
  );

  // Foreign owners fail every page operation.
  const { service: live, token: liveToken } = openService();
  expectFetchError(
    () => live.seedPageCookies("owner-x", "page-a", liveToken, 1),
    "wrong-owner",
    "page",
  );
  expectFetchError(
    () => live.deliverFetchResponse("owner-x", "page-a", liveToken, "f1", responseDescriptor()),
    "wrong-owner",
    "page",
  );
  expectFetchError(
    () => live.closeFetch("owner-x", "page-a", liveToken, "f1"),
    "wrong-owner",
    "page",
  );
  // A second identity granted the same page name cannot join it.
  const shared = new PageFetchService(
    [
      { owner: "owner-n", page: "page-a", pageOrigin: PAGE_ORIGIN, csp: "connect-self-only" },
      { owner: "owner-x", page: "page-a", pageOrigin: PAGE_ORIGIN, csp: "connect-self-only" },
    ],
    [PAGE_ORIGIN],
    roomyLimits(),
  );
  shared.bindPage("owner-n", "page-a");
  expectFetchError(() => shared.bindPage("owner-x", "page-a"), "wrong-owner", "page");
});

// --- Tokens are opaque and verified by lookup --------------------------------

check("page-tokens-opaque-and-verified", () => {
  const { service, token } = openService();
  service.seedPageCookies("owner-n", "page-a", token, 2);
  const begun = service.fetchBegin("owner-n", "page-a", token, "f1", fetchDescriptor());
  assert.equal(begun.cspDecision, "allowed");
  // Invented tokens are never authority, on any token-bearing call.
  for (const forged of ["", "page-deadbeef", `${token.slice(0, -1)}x`]) {
    expectFetchError(
      () => service.seedPageCookies("owner-n", "page-a", forged, 1),
      "forged-token",
      "fetch",
    );
    expectFetchError(
      () => service.fetchBegin("owner-n", "page-a", forged, "f9", fetchDescriptor()),
      "forged-token",
      "fetch",
    );
    expectFetchError(
      () => service.deliverFetchResponse("owner-n", "page-a", forged, "f1", responseDescriptor()),
      "forged-token",
      "fetch",
    );
    expectFetchError(
      () => service.followFetchRedirect("owner-n", "page-a", forged, "f1", "f2"),
      "forged-token",
      "fetch",
    );
    expectFetchError(
      () => service.consumeFetchBody("owner-n", "page-a", forged, "f1", 1),
      "forged-token",
      "fetch",
    );
    expectFetchError(
      () => service.closeFetch("owner-n", "page-a", forged, "f1"),
      "forged-token",
      "fetch",
    );
  }
  // Tokens never transfer between pages.
  const two = new PageFetchService(
    [
      { owner: "owner-n", page: "page-a", pageOrigin: PAGE_ORIGIN, csp: "connect-self-only" },
      { owner: "owner-n", page: "page-b", pageOrigin: PAGE_ORIGIN, csp: "connect-self-only" },
    ],
    [PAGE_ORIGIN],
    roomyLimits(),
  );
  two.bindPage("owner-n", "page-a");
  two.bindPage("owner-n", "page-b");
  const tokenA = two.pageTokenForTest("owner-n", "page-a");
  const tokenB = two.pageTokenForTest("owner-n", "page-b");
  assert.notEqual(tokenA, tokenB);
  expectFetchError(
    () => two.fetchBegin("owner-n", "page-b", tokenA, "f1", fetchDescriptor()),
    "forged-token",
    "fetch",
  );
  expectFetchError(
    () => two.fetchBegin("owner-n", "page-a", tokenB, "f1", fetchDescriptor()),
    "forged-token",
    "fetch",
  );
});

// --- Typed bounded Fetch: methods, credentials, redirect ----------------------

check("typed-fetch-vocabulary", () => {
  const service = new PageFetchService(
    [{ owner: "owner-n", page: "page-a", pageOrigin: PAGE_ORIGIN, csp: "connect-self-only" }],
    [PAGE_ORIGIN, PEER_ORIGIN],
    checkFetchLimits({
      maxPages: 4,
      maxFetchesPerPage: 128,
      maxCookiesPerPage: 8,
      maxBodyBytes: 1024,
      maxRedirectHops: 3,
    }),
  );
  service.bindPage("owner-n", "page-a");
  const token = service.pageTokenForTest("owner-n", "page-a");
  assert.deepEqual(
    [...FETCH_METHODS],
    ["GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH"],
  );
  assert.deepEqual([...FETCH_CREDENTIALS], ["omit", "same-origin", "include"]);
  assert.deepEqual([...FETCH_REDIRECTS], ["follow", "error", "manual"]);
  assert.deepEqual([...FETCH_DECODING_SCOPES], ["raw-bytes", "utf8-strict", "utf8-replace"]);
  // Every method, credentials mode, and redirect policy is accepted.
  let seq = 0;
  for (const method of FETCH_METHODS) {
    for (const credentials of FETCH_CREDENTIALS) {
      for (const redirect of FETCH_REDIRECTS) {
        const id = `v${seq++}`;
        const facts = service.fetchBegin(
          "owner-n",
          "page-a",
          token,
          id,
          fetchDescriptor({ method, credentials, redirect }),
        );
        assert.equal(facts.realm, "page-fetch");
        if (facts.cspDecision !== "allowed") {
          assert.fail("same-origin fetch under connect-self-only must allow");
        }
        assert.equal(facts.method, method);
        assert.equal(facts.credentials, credentials);
        assert.equal(facts.redirect, redirect);
        service.deliverFetchResponse("owner-n", "page-a", token, id, responseDescriptor());
        service.closeFetch("owner-n", "page-a", token, id);
      }
    }
  }
  // Vocabulary outside the finite sets rejects before any effect.
  for (const bad of ["FETCH", "", "get", "POST "]) {
    expectFetchError(
      () => service.fetchBegin("owner-n", "page-a", token, "bad", fetchDescriptor({ method: bad })),
      "unsupported-method",
      "fetch",
    );
  }
  for (const bad of ["cookies", "", "SAME-ORIGIN", "none"]) {
    expectFetchError(
      () =>
        service.fetchBegin(
          "owner-n",
          "page-a",
          token,
          "bad",
          fetchDescriptor({ credentials: bad }),
        ),
      "unknown-credentials",
      "policy",
    );
  }
  for (const bad of ["always", "", "FOLLOW", "never"]) {
    expectFetchError(
      () =>
        service.fetchBegin("owner-n", "page-a", token, "bad", fetchDescriptor({ redirect: bad })),
      "unknown-redirect",
      "policy",
    );
  }
  let decSeq = 0;
  for (const bad of ["text", "", "utf8", "UTF8-STRICT"]) {
    const id = `dec${decSeq++}`;
    service.fetchBegin("owner-n", "page-a", token, id, fetchDescriptor());
    expectFetchError(
      () =>
        service.deliverFetchResponse(
          "owner-n",
          "page-a",
          token,
          id,
          responseDescriptor({ decodingScope: bad }),
        ),
      "unknown-fetch",
      "fetch",
    );
    service.deliverFetchResponse("owner-n", "page-a", token, id, responseDescriptor());
    service.closeFetch("owner-n", "page-a", token, id);
  }
  // Unknown fetch ids and double-settled ids refuse.
  expectFetchError(
    () => service.deliverFetchResponse("owner-n", "page-a", token, "ghost", responseDescriptor()),
    "unknown-fetch",
    "fetch",
  );
  expectFetchError(
    () => service.fetchBegin("owner-n", "page-a", token, "v0", fetchDescriptor()),
    "fetch-settled",
    "fetch",
  );
  // Target origins outside the declared set reject; malformed origins too.
  for (const bad of ["https://evil.example", ""]) {
    expectFetchError(
      () =>
        service.fetchBegin(
          "owner-n",
          "page-a",
          token,
          "bad",
          fetchDescriptor({ targetOrigin: bad }),
        ),
      "unknown-origin",
      "policy",
    );
  }
  // Target paths must be absolute paths.
  for (const bad of ["data", "", "https://page.example/data"]) {
    expectFetchError(
      () =>
        service.fetchBegin("owner-n", "page-a", token, "bad", fetchDescriptor({ targetPath: bad })),
      "unknown-fetch",
      "fetch",
    );
  }
});

// --- Cookie facts name the page realm and jar ----------------------------------

check("cookie-facts-name-page-jar", () => {
  const { service, token } = openService();
  const seed = service.seedPageCookies("owner-n", "page-a", token, 3);
  assert.equal(seed.realm, "page-fetch");
  assert.equal(seed.cookieJar, "page-jar:page-a");
  assert.equal(seed.cookies, 3);
  assert.ok(seed.jarDigest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(seed).sort(), [
    "cookieJar",
    "cookies",
    "jarDigest",
    "page",
    "realm",
  ]);

  // omit sends nothing even same-origin with a seeded jar.
  const omitted = service.fetchBegin(
    "owner-n",
    "page-a",
    token,
    "c-omit",
    fetchDescriptor({ credentials: "omit" }),
  );
  assert.equal(omitted.cspDecision, "allowed");
  if (omitted.cspDecision === "allowed") {
    assert.equal(omitted.cookieJar, "page-jar:page-a");
    assert.equal(omitted.cookiesSeeded, 3);
    assert.equal(omitted.cookiesSent, 0);
  }

  // same-origin sends the seeded jar same-origin.
  const same = service.fetchBegin(
    "owner-n",
    "page-a",
    token,
    "c-same",
    fetchDescriptor({ credentials: "same-origin" }),
  );
  if (same.cspDecision === "allowed") {
    assert.equal(same.sameOrigin, true);
    assert.equal(same.cookiesSent, 3);
  } else {
    assert.fail("same-origin fetch must allow");
  }

  // include sends the seeded jar cross-origin under a permissive test CSP.
  // Under connect-self-only a cross-origin fetch blocks instead: the
  // blocked facts still name the page jar and send zero cookies.
  const blocked = service.fetchBegin(
    "owner-n",
    "page-a",
    token,
    "c-cross",
    fetchDescriptor({ targetOrigin: PEER_ORIGIN, credentials: "include" }),
  );
  assert.equal(blocked.cspDecision, "blocked");
  if (blocked.cspDecision === "blocked") {
    assert.equal(blocked.blockedBy, "origin");
    assert.equal(blocked.cookieJar, "page-jar:page-a");
    assert.equal(blocked.cookiesSent, 0);
  }
  for (const id of ["c-omit", "c-same"]) {
    service.deliverFetchResponse("owner-n", "page-a", token, id, responseDescriptor());
    const receipt = service.closeFetch("owner-n", "page-a", token, id);
    assert.ok(!("cookieJar" in receipt));
    assert.equal(receipt.realm, "page-fetch");
  }
  const blockedReceipt = service.closeFetch("owner-n", "page-a", token, "c-cross");
  assert.equal(blockedReceipt.cookiesSent, 0);
  assert.equal(blockedReceipt.cspDecision, "blocked");

  // Seeding past the jar bound refuses without effect.
  expectFetchError(
    () => service.seedPageCookies("owner-n", "page-a", token, 99),
    "capacity-exhausted",
    "page",
  );
});

// --- Origin facts name the page realm and page origin --------------------------

check("origin-facts-name-page-realm", () => {
  const { service, token } = openService();
  const facts = service.fetchBegin("owner-n", "page-a", token, "o1", fetchDescriptor());
  if (facts.cspDecision !== "allowed") {
    assert.fail("same-origin fetch must allow");
  }
  assert.equal(facts.realm, "page-fetch");
  assert.equal(facts.pageOrigin, PAGE_ORIGIN);
  assert.equal(facts.targetOrigin, PAGE_ORIGIN);
  assert.equal(facts.sameOrigin, true);
  // GET carries no Origin header; POST states the page origin.
  assert.equal(facts.originHeader, null);
  const post = service.fetchBegin(
    "owner-n",
    "page-a",
    token,
    "o2",
    fetchDescriptor({ method: "POST" }),
  );
  if (post.cspDecision === "allowed") {
    assert.equal(post.originHeader, PAGE_ORIGIN);
  } else {
    assert.fail("same-origin POST must allow");
  }
  assert.deepEqual(Object.keys(facts).sort(), [
    "cookieJar",
    "cookiesSeeded",
    "cookiesSent",
    "credentials",
    "csp",
    "cspDecision",
    "digest",
    "fetchId",
    "hops",
    "method",
    "originHeader",
    "owner",
    "page",
    "pageOrigin",
    "realm",
    "redirect",
    "sameOrigin",
    "supersedes",
    "targetOrigin",
    "targetPath",
  ]);
  // The digest recomputes from carried fields: presented facts verify.
  assert.equal(
    facts.digest,
    digestPageFetch(
      "o1",
      "owner-n",
      "page-a",
      PAGE_ORIGIN,
      PAGE_ORIGIN,
      "allowed:GET:same-origin:manual:0",
    ),
  );
  service.deliverFetchResponse("owner-n", "page-a", token, "o1", responseDescriptor());
  service.deliverFetchResponse("owner-n", "page-a", token, "o2", responseDescriptor());
  service.closeFetch("owner-n", "page-a", token, "o1");
  service.closeFetch("owner-n", "page-a", token, "o2");
});

// --- CSP facts name the page policy ---------------------------------------------

check("csp-facts-name-page-policy", () => {
  // deny-all blocks every target, same-origin included, with frozen facts
  // proving the decision; delivery and reads refuse the blocked fetch.
  const denied = openService("deny-all");
  const blocked = denied.service.fetchBegin(
    "owner-n",
    "page-a",
    denied.token,
    "s1",
    fetchDescriptor(),
  );
  assert.equal(blocked.realm, "page-fetch");
  assert.equal(blocked.cspDecision, "blocked");
  if (blocked.cspDecision === "blocked") {
    assert.equal(blocked.csp, "deny-all");
    assert.equal(blocked.blockedBy, "csp");
    assert.equal(blocked.pageOrigin, PAGE_ORIGIN);
    assert.deepEqual(Object.keys(blocked).sort(), [
      "blockedBy",
      "cookieJar",
      "cookiesSent",
      "csp",
      "cspDecision",
      "digest",
      "fetchId",
      "owner",
      "page",
      "pageOrigin",
      "realm",
      "sameOrigin",
      "targetOrigin",
    ]);
  }
  expectFetchError(
    () =>
      denied.service.deliverFetchResponse(
        "owner-n",
        "page-a",
        denied.token,
        "s1",
        responseDescriptor(),
      ),
    "csp-blocked",
    "policy",
  );
  expectFetchError(
    () => denied.service.consumeFetchBody("owner-n", "page-a", denied.token, "s1", 1),
    "csp-blocked",
    "policy",
  );
  const deniedReceipt = denied.service.closeFetch("owner-n", "page-a", denied.token, "s1");
  assert.equal(deniedReceipt.cspDecision, "blocked");

  // connect-self-only allows same-origin and blocks cross-origin.
  const { service, token } = openService("connect-self-only");
  const allowed = service.fetchBegin("owner-n", "page-a", token, "s2", fetchDescriptor());
  assert.equal(allowed.cspDecision, "allowed");
  const cross = service.fetchBegin(
    "owner-n",
    "page-a",
    token,
    "s3",
    fetchDescriptor({ targetOrigin: PEER_ORIGIN }),
  );
  assert.equal(cross.cspDecision, "blocked");
  if (cross.cspDecision === "blocked") {
    assert.equal(blocked.csp, "deny-all");
    assert.equal(cross.blockedBy, "origin");
    assert.equal(cross.sameOrigin, false);
  }
  expectFetchError(
    () => service.deliverFetchResponse("owner-n", "page-a", token, "s3", responseDescriptor()),
    "origin-blocked",
    "policy",
  );
  service.deliverFetchResponse("owner-n", "page-a", token, "s2", responseDescriptor());
  service.closeFetch("owner-n", "page-a", token, "s2");
  service.closeFetch("owner-n", "page-a", token, "s3");

  // Grants outside the finite CSP vocabulary reject at construction.
  expectFetchError(
    () =>
      new PageFetchService(
        [{ owner: "owner-n", page: "page-a", pageOrigin: PAGE_ORIGIN, csp: "allow-all" as never }],
        [PAGE_ORIGIN],
        roomyLimits(),
      ),
    "csp-blocked",
    "policy",
  );
});

// --- Redirect policy: explicit follow, never implicit ----------------------------

check("redirect-policy-explicit-follow", () => {
  const { service, token } = openService();
  // follow holds the redirect: no chained fetch exists until the caller
  // follows explicitly, and close refuses while it is held.
  service.fetchBegin("owner-n", "page-a", token, "r1", fetchDescriptor({ redirect: "follow" }));
  const held = service.deliverFetchResponse(
    "owner-n",
    "page-a",
    token,
    "r1",
    responseDescriptor({ status: 302, location: PEER_ORIGIN }),
  );
  assert.equal(held.realm, "page-fetch");
  assert.deepEqual(held.redirect, { status: 302, location: PEER_ORIGIN });
  assert.equal(held.redirectIncomplete, false);
  assert.equal(service.fetchCount("owner-n", "page-a"), 1);
  expectFetchError(
    () => service.closeFetch("owner-n", "page-a", token, "r1"),
    "redirect-open",
    "policy",
  );
  // The explicit follow mints exactly one chained fetch linked via
  // supersedes with a hop count. The chained target is cross-origin, so
  // the page policy blocks it with facts (no silent drop).
  const chained = service.followFetchRedirect("owner-n", "page-a", token, "r1", "r2");
  assert.equal(service.fetchCount("owner-n", "page-a"), 2);
  assert.equal(chained.cspDecision, "blocked");
  service.closeFetch("owner-n", "page-a", token, "r1");
  service.closeFetch("owner-n", "page-a", token, "r2");

  // Same-origin chains allow and count hops.
  service.fetchBegin("owner-n", "page-a", token, "r3", fetchDescriptor({ redirect: "follow" }));
  service.deliverFetchResponse(
    "owner-n",
    "page-a",
    token,
    "r3",
    responseDescriptor({ status: 302, location: PAGE_ORIGIN }),
  );
  const child = service.followFetchRedirect("owner-n", "page-a", token, "r3", "r4");
  if (child.cspDecision === "allowed") {
    assert.equal(child.hops, 1);
    assert.equal(child.supersedes, "r3");
  } else {
    assert.fail("same-origin chain must allow");
  }
  // Following twice refuses: the parent already settled.
  expectFetchError(
    () => service.followFetchRedirect("owner-n", "page-a", token, "r3", "r5"),
    "redirect-open",
    "policy",
  );
  service.deliverFetchResponse("owner-n", "page-a", token, "r4", responseDescriptor());
  service.closeFetch("owner-n", "page-a", token, "r3");
  service.closeFetch("owner-n", "page-a", token, "r4");

  // error and manual settle with the redirect facts carried; following
  // refuses because nothing is held.
  for (const policy of ["error", "manual"] as const) {
    const id = `nr-${policy}`;
    service.fetchBegin("owner-n", "page-a", token, id, fetchDescriptor({ redirect: policy }));
    const settled = service.deliverFetchResponse(
      "owner-n",
      "page-a",
      token,
      id,
      responseDescriptor({ status: 302, location: PAGE_ORIGIN }),
    );
    assert.deepEqual(settled.redirect, { status: 302, location: PAGE_ORIGIN });
    expectFetchError(
      () => service.followFetchRedirect("owner-n", "page-a", token, id, `${id}-next`),
      "redirect-open",
      "policy",
    );
    service.closeFetch("owner-n", "page-a", token, id);
  }

  // A 302 without a location sets redirect_incomplete and settles: there
  // is nothing to follow.
  service.fetchBegin("owner-n", "page-a", token, "ri", fetchDescriptor({ redirect: "follow" }));
  const incomplete = service.deliverFetchResponse(
    "owner-n",
    "page-a",
    token,
    "ri",
    responseDescriptor({ status: 302, location: null }),
  );
  assert.equal(incomplete.redirect, null);
  assert.equal(incomplete.redirectIncomplete, true);
  expectFetchError(
    () => service.followFetchRedirect("owner-n", "page-a", token, "ri", "ri-next"),
    "redirect-open",
    "policy",
  );
  service.closeFetch("owner-n", "page-a", token, "ri");

  // The hop budget bounds chains: a zero-hop service never mints one.
  const zero = new PageFetchService(
    [{ owner: "owner-n", page: "page-a", pageOrigin: PAGE_ORIGIN, csp: "connect-self-only" }],
    [PAGE_ORIGIN],
    checkFetchLimits({
      maxPages: 1,
      maxFetchesPerPage: 4,
      maxCookiesPerPage: 1,
      maxBodyBytes: 64,
      maxRedirectHops: 0,
    }),
  );
  zero.bindPage("owner-n", "page-a");
  const ztoken = zero.pageTokenForTest("owner-n", "page-a");
  zero.fetchBegin("owner-n", "page-a", ztoken, "z1", fetchDescriptor({ redirect: "follow" }));
  zero.deliverFetchResponse(
    "owner-n",
    "page-a",
    ztoken,
    "z1",
    responseDescriptor({ status: 302, location: PAGE_ORIGIN }),
  );
  expectFetchError(
    () => zero.followFetchRedirect("owner-n", "page-a", ztoken, "z1", "z2"),
    "redirect-open",
    "policy",
  );
});

// --- Accepted vs consumed; settlement distinct from close -------------------------

check("accepted-vs-consumed-and-close", () => {
  const { service, token } = openService();
  service.fetchBegin("owner-n", "page-a", token, "b1", fetchDescriptor());
  // No delivered response yet: reads and close refuse, and name the gap.
  expectFetchError(
    () => service.consumeFetchBody("owner-n", "page-a", token, "b1", 4),
    "response-pending",
    "fetch",
  );
  expectFetchError(
    () => service.closeFetch("owner-n", "page-a", token, "b1"),
    "response-pending",
    "fetch",
  );
  const delivered = service.deliverFetchResponse(
    "owner-n",
    "page-a",
    token,
    "b1",
    responseDescriptor({ byteLength: 10 }),
  );
  assert.equal(delivered.bodyAccepted, 10);
  assert.equal(delivered.bodyConsumed, 0);
  // A second delivery refuses: one response joins.
  expectFetchError(
    () => service.deliverFetchResponse("owner-n", "page-a", token, "b1", responseDescriptor()),
    "fetch-settled",
    "fetch",
  );
  // Partial consumes report eof:false; only full consumption reports eof.
  const first = service.consumeFetchBody("owner-n", "page-a", token, "b1", 4);
  assert.equal(first.consumed, 4);
  assert.equal(first.consumedTotal, 4);
  assert.equal(first.acceptedTotal, 10);
  assert.equal(first.eof, false);
  const second = service.consumeFetchBody("owner-n", "page-a", token, "b1", 99);
  assert.equal(second.consumed, 6);
  assert.equal(second.eof, true);
  const receipt = service.closeFetch("owner-n", "page-a", token, "b1");
  assert.equal(receipt.bodyAccepted, 10);
  assert.equal(receipt.bodyConsumed, 10);
  assert.equal(receipt.bodyUnread, 0);
  // Closed fetches refuse every later operation.
  expectFetchError(
    () => service.deliverFetchResponse("owner-n", "page-a", token, "b1", responseDescriptor()),
    "fetch-closed",
    "fetch",
  );
  expectFetchError(
    () => service.consumeFetchBody("owner-n", "page-a", token, "b1", 1),
    "fetch-closed",
    "fetch",
  );
  expectFetchError(
    () => service.closeFetch("owner-n", "page-a", token, "b1"),
    "fetch-closed",
    "fetch",
  );
  // Closing the page with an open fetch refuses and names the fetch.
  service.fetchBegin("owner-n", "page-a", token, "b2", fetchDescriptor());
  try {
    service.closePage("owner-n", "page-a");
    assert.fail("expected fetch-open");
  } catch (error) {
    assert.ok(error instanceof BrowserFetchError);
    assert.equal(error.code, "fetch-open");
    assert.ok(String(error).includes("b2"));
  }
  service.deliverFetchResponse("owner-n", "page-a", token, "b2", responseDescriptor());
  service.closeFetch("owner-n", "page-a", token, "b2");
  service.closePage("owner-n", "page-a");
  // Bodies past the byte cap refuse without effect.
  const { service: capped, token: cappedToken } = openService();
  capped.fetchBegin("owner-n", "page-a", cappedToken, "big", fetchDescriptor());
  expectFetchError(
    () =>
      capped.deliverFetchResponse(
        "owner-n",
        "page-a",
        cappedToken,
        "big",
        responseDescriptor({ byteLength: 99_999 }),
      ),
    "body-limit",
    "fetch",
  );
});

// --- Acceptance: no source eval ----------------------------------------------------

check("no-source-eval", () => {
  assert.deepEqual([...FORBIDDEN_EVAL_KEYS], ["code", "source", "script"]);
  const { service, token } = openService();
  // Every source-eval key rejects on fetchBegin, whatever else rides along.
  for (const key of FORBIDDEN_EVAL_KEYS) {
    expectFetchError(
      () =>
        service.fetchBegin("owner-n", "page-a", token, "e1", { ...fetchDescriptor(), [key]: "x" }),
      "forbidden-eval",
      "policy",
    );
    // Refusal precedes vocabulary validation: an eval key plus a bad
    // method still reports forbidden-eval.
    expectFetchError(
      () =>
        service.fetchBegin("owner-n", "page-a", token, "e1", {
          ...fetchDescriptor(),
          method: "NOPE",
          [key]: "x",
        }),
      "forbidden-eval",
      "policy",
    );
  }
  // Unknown non-eval fields still reject, with the fetch-layer code.
  expectFetchError(
    () => service.fetchBegin("owner-n", "page-a", token, "e1", { ...fetchDescriptor(), extra: 1 }),
    "unknown-fetch",
    "fetch",
  );
  // Deliveries refuse eval keys the same way.
  service.fetchBegin("owner-n", "page-a", token, "e2", fetchDescriptor());
  for (const key of FORBIDDEN_EVAL_KEYS) {
    expectFetchError(
      () =>
        service.deliverFetchResponse("owner-n", "page-a", token, "e2", {
          ...responseDescriptor(),
          [key]: "x",
        }),
      "forbidden-eval",
      "policy",
    );
  }
  // The refused deliveries left no effect: the fetch is still open.
  service.deliverFetchResponse("owner-n", "page-a", token, "e2", responseDescriptor());
  service.closeFetch("owner-n", "page-a", token, "e2");
  assert.equal(service.fetchCount("owner-n", "page-a"), 1);
});

// --- Acceptance: page Fetch facts are never external HTTP facts --------------------

check("page-fetch-never-external-http", () => {
  const { service, token } = openService();
  service.seedPageCookies("owner-n", "page-a", token, 1);
  const begun = service.fetchBegin("owner-n", "page-a", token, "x1", fetchDescriptor());
  const delivered = service.deliverFetchResponse(
    "owner-n",
    "page-a",
    token,
    "x1",
    responseDescriptor(),
  );
  const receipt = service.closeFetch("owner-n", "page-a", token, "x1");
  const blocked = service.fetchBegin(
    "owner-n",
    "page-a",
    token,
    "x2",
    fetchDescriptor({ targetOrigin: PEER_ORIGIN }),
  );
  const binding = service.binding("owner-n", "page-a");
  // Every fact names the page realm, never the external kind.
  for (const facts of [binding, begun, delivered, receipt, blocked]) {
    assert.equal((facts as { realm: string }).realm, PAGE_FETCH_REALM);
    assert.equal((facts as { realm: string }).realm, "page-fetch");
    assert.ok(!("kind" in (facts as Record<string, unknown>)));
  }
  // The realm vocabulary is disjoint from the external HTTP vocabulary:
  // K20 facts carry kind "http-request" with no realm at all.
  assert.ok(!(HTTP_METHODS as readonly string[]).includes(PAGE_FETCH_REALM));
  const encoded = JSON.stringify([binding, begun, delivered, receipt, blocked]);
  assert.ok(!encoded.includes("http-request"));
  assert.ok(!encoded.includes("external"));
  // Cookie, origin, and CSP facts always name the page realm, the page
  // origin, and the page policy together.
  assert.ok(encoded.includes("page-jar:page-a"));
  assert.ok(encoded.includes(PAGE_ORIGIN));
  assert.ok(encoded.includes("connect-self-only"));
  // Facts carry digests only: the raw page token never crosses.
  assert.ok(!encoded.includes(token));
});

// --- Digest-only frozen facts and closed error vocabulary ---------------------------

check("digest-only-facts-and-closed-vocabulary", () => {
  const { service, token } = openService();
  service.seedPageCookies("owner-n", "page-a", token, 1);
  const begun = service.fetchBegin(
    "owner-n",
    "page-a",
    token,
    "d1",
    fetchDescriptor({ method: "POST", credentials: "include", redirect: "follow" }),
  );
  assert.ok(Object.isFrozen(begun));
  const encoded = JSON.stringify(begun);
  assert.ok(!encoded.includes(token));
  // Every code maps to exactly one layer, and every layer owns a code.
  const seen = new Map<BrowserFetchCode, BrowserFetchLayer>();
  for (const code of BROWSER_FETCH_CODES) {
    const layer = layerOfFetchCode(code);
    assert.ok(["page", "fetch", "policy"].includes(layer));
    assert.ok(!seen.has(code));
    seen.set(code, layer);
  }
  const layers = new Set(seen.values());
  assert.deepEqual([...layers].sort(), ["fetch", "page", "policy"]);
  service.deliverFetchResponse("owner-n", "page-a", token, "d1", responseDescriptor());
  service.closeFetch("owner-n", "page-a", token, "d1");
});

// --- Declared sets and finite bounds --------------------------------------------------

check("declared-sets-and-finite-bounds", () => {
  // Empty grants, empty origins, and unknown limit fields reject.
  expectFetchError(
    () => new PageFetchService([], [PAGE_ORIGIN], roomyLimits()),
    "unknown-page",
    "page",
  );
  expectFetchError(
    () =>
      new PageFetchService(
        [{ owner: "owner-n", page: "page-a", pageOrigin: PAGE_ORIGIN, csp: "connect-self-only" }],
        [],
        roomyLimits(),
      ),
    "unknown-origin",
    "policy",
  );
  expectFetchError(
    () =>
      checkFetchLimits({
        maxPages: 1,
        maxFetchesPerPage: 1,
        maxCookiesPerPage: 1,
        maxBodyBytes: 1,
        maxRedirectHops: 0,
        extra: 1,
      }),
    "unknown-page",
    "page",
  );
  // Tiny bounds refuse exactly at the cap.
  const tiny = new PageFetchService(
    [
      { owner: "o", page: "p1", pageOrigin: PAGE_ORIGIN, csp: "connect-self-only" },
      { owner: "o", page: "p2", pageOrigin: PAGE_ORIGIN, csp: "connect-self-only" },
    ],
    [PAGE_ORIGIN],
    checkFetchLimits({
      maxPages: 2,
      maxFetchesPerPage: 1,
      maxCookiesPerPage: 1,
      maxBodyBytes: 8,
      maxRedirectHops: 1,
    }),
  );
  tiny.bindPage("o", "p1");
  tiny.bindPage("o", "p2");
  const tok = tiny.pageTokenForTest("o", "p1");
  tiny.seedPageCookies("o", "p1", tok, 1);
  expectFetchError(() => tiny.seedPageCookies("o", "p1", tok, 1), "capacity-exhausted", "page");
  tiny.fetchBegin("o", "p1", tok, "f1", fetchDescriptor());
  expectFetchError(
    () => tiny.fetchBegin("o", "p1", tok, "f2", fetchDescriptor()),
    "capacity-exhausted",
    "page",
  );
});

check("independence-scope-stated", () => {
  // The scope statement is a fixed export: the service shares no external
  // HTTP code, doubles, receipts, or state, and proves its facts from its
  // own seeded doubles alone.
  assert.ok(BROWSER_FETCH_INDEPENDENCE_SCOPE.startsWith("independent:"));
  assert.ok(BROWSER_FETCH_INDEPENDENCE_SCOPE.includes("no external HTTP code"));
  assert.ok(BROWSER_FETCH_INDEPENDENCE_SCOPE.includes("seeded doubles"));
  assert.ok(BROWSER_FETCH_INDEPENDENCE_SCOPE.includes("page realm"));
});

console.log(
  JSON.stringify({
    kind: "can.browser-driver-fetch-check",
    schema_version: BROWSER_FETCH_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
