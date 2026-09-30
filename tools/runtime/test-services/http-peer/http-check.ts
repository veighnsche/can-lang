// K20 bounded self-check: typed in-memory HTTP operations only.
//
// Run with: bun tools/runtime/test-services/http-peer/http-check.ts
// Local controls only. No fetch, no network, no timers, no files. Every
// loop below is bounded by a small constant.

import { strict as assert } from "node:assert";
import { checkFactsInert } from "../native-values/schema.ts";
import { checkHttpLimits, HttpTestService, type HttpLimits } from "./http.ts";
import { HttpPeerError, HTTP_PEER_SCHEMA_VERSION } from "./peer.ts";

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectPeerError(body: () => unknown, kind: string): void {
  try {
    body();
  } catch (error) {
    if (!(error instanceof HttpPeerError)) {
      assert.fail("expected a HttpPeerError");
    }
    assert.equal(error.kind, kind);
    return;
  }
  assert.fail(`expected rejection with kind ${kind}`);
}

function serviceLimits(): HttpLimits {
  return checkHttpLimits({
    max_requests: 16,
    max_headers: 8,
    max_header_bytes: 512,
    max_header_value_bytes: 128,
    max_body_bytes: 64,
    max_chunk_bytes: 16,
    max_reissues: 3,
    max_target_bytes: 128,
  });
}

function tinyLimits(): HttpLimits {
  return checkHttpLimits({
    max_requests: 4,
    max_headers: 2,
    max_header_bytes: 64,
    max_header_value_bytes: 16,
    max_body_bytes: 4,
    max_chunk_bytes: 4,
    max_reissues: 1,
    max_target_bytes: 32,
  });
}

// --- Admission -----------------------------------------------------------------

check("declare-and-request", () => {
  const service = new HttpTestService(["https://svc"], serviceLimits());
  const handle = service.request("owner-a", "https://svc", "POST", "/upload", "1.1");
  assert.equal(handle.kind, "http-request");
  assert(Object.isFrozen(handle));
  // Foreign origins reject before any state changes.
  expectPeerError(
    () => service.request("owner-a", "https://evil", "POST", "/upload", "1.1"),
    "foreign-destination",
  );
  assert.equal(service.request_count, 1);
  // Closed method/version vocabularies.
  expectPeerError(
    () => service.request("owner-a", "https://svc", "DELETE", "/x", "1.1"),
    "unsupported-capability",
  );
  expectPeerError(
    () => service.request("owner-a", "https://svc", "GET", "/x", "2"),
    "unsupported-capability",
  );
  // Malformed targets.
  expectPeerError(
    () => service.request("owner-a", "https://svc", "GET", "no-slash", "1.1"),
    "invalid-request",
  );
  expectPeerError(
    () => service.request("owner-a", "https://svc", "GET", "/has-\u00e9", "1.1"),
    "invalid-request",
  );
  assert.equal(service.request_count, 1);
});

// --- Repeated headers ------------------------------------------------------------

check("repeated-headers-ordered", () => {
  const service = new HttpTestService(["https://svc"], serviceLimits());
  const handle = service.request("owner-a", "https://svc", "POST", "/t", "1.1");
  service.addHeader("owner-a", handle.id, "set-cookie", "a=1");
  service.addHeader("owner-a", handle.id, "x-trace", "t1");
  service.addHeader("owner-a", handle.id, "set-cookie", "b=2");
  const facts = service.requestFacts("owner-a", handle.id);
  // Repeats preserved in order, never folded.
  assert.deepEqual(
    facts.headers.map(([name, value]) => [name, value]),
    [
      ["set-cookie", "a=1"],
      ["x-trace", "t1"],
      ["set-cookie", "b=2"],
    ],
  );
  assert.equal(facts.header_count, 3);
  assert.equal(facts.headers_truncated, false);
  assert(Object.isFrozen(facts.headers));
});

check("header-value-rules", () => {
  const service = new HttpTestService(["https://svc"], serviceLimits());
  const handle = service.request("owner-a", "https://svc", "POST", "/t", "1.1");
  expectPeerError(
    () => service.addHeader("owner-a", handle.id, "x-custom", "v"),
    "unsupported-capability",
  );
  expectPeerError(
    () => service.addHeader("owner-a", handle.id, "x-trace", "has-\u00e9"),
    "invalid-request",
  );
  expectPeerError(() => service.addHeader("owner-a", handle.id, "x-trace", 42), "invalid-request");
  expectPeerError(
    () => service.addHeader("owner-a", handle.id, "x-trace", "v".repeat(129)),
    "invalid-request",
  );
  assert.equal(service.requestFacts("owner-a", handle.id).header_count, 0);
});

check("header-truncation-explicit", () => {
  const service = new HttpTestService(["https://svc"], tinyLimits());
  const handle = service.request("owner-a", "https://svc", "POST", "/t", "1.1");
  service.addHeader("owner-a", handle.id, "x-trace", "one");
  service.addHeader("owner-a", handle.id, "x-trace", "two");
  // Third header rejects AND sets the sticky truncated fact; the loss is
  // observable in facts rather than swallowed by the error.
  expectPeerError(
    () => service.addHeader("owner-a", handle.id, "x-trace", "three"),
    "header-limit",
  );
  const facts = service.requestFacts("owner-a", handle.id);
  assert.equal(facts.headers_truncated, true);
  assert.equal(facts.header_count, 2);
  // The request stays usable with its explicit partial state.
  service.endUpload("owner-a", handle.id);
  assert.equal(service.requestFacts("owner-a", handle.id).state, "upload_complete");
});

// --- Upload body -------------------------------------------------------------------

check("upload-body-and-get-rule", () => {
  const service = new HttpTestService(["https://svc"], serviceLimits());
  const post = service.request("owner-a", "https://svc", "POST", "/u", "1.1");
  const first = service.sendBodyChunk("owner-a", post.id, [1, 2, 3]);
  assert.equal(first.accepted, 3);
  assert.equal(first.body_accepted_total, 3);
  const second = service.sendBodyChunk("owner-a", post.id, [4]);
  assert.equal(second.body_accepted_total, 4);
  expectPeerError(() => service.sendBodyChunk("owner-a", post.id, []), "invalid-request");
  const get = service.request("owner-a", "https://svc", "GET", "/g", "1.1");
  expectPeerError(() => service.sendBodyChunk("owner-a", get.id, [1]), "invalid-state");
  service.endUpload("owner-a", post.id);
  expectPeerError(() => service.sendBodyChunk("owner-a", post.id, [5]), "invalid-state");
  expectPeerError(() => service.addHeader("owner-a", post.id, "x-trace", "late"), "invalid-state");
  expectPeerError(() => service.endUpload("owner-a", post.id), "invalid-state");
});

check("upload-truncation-explicit", () => {
  const service = new HttpTestService(["https://svc"], tinyLimits());
  const handle = service.request("owner-a", "https://svc", "POST", "/u", "1.1");
  service.sendBodyChunk("owner-a", handle.id, [1, 2, 3, 4]);
  expectPeerError(() => service.sendBodyChunk("owner-a", handle.id, [5]), "body-limit");
  const facts = service.requestFacts("owner-a", handle.id);
  assert.equal(facts.upload_truncated, true);
  assert.equal(facts.upload_accepted, 4);
  service.endUpload("owner-a", handle.id);
});

check("upload-length-mismatch", () => {
  const service = new HttpTestService(["https://svc"], serviceLimits());
  const handle = service.request("owner-a", "https://svc", "POST", "/u", "1.1");
  service.addHeader("owner-a", handle.id, "content-length", "10");
  service.sendBodyChunk("owner-a", handle.id, [1, 2, 3, 4]);
  // Short upload completes with an explicit mismatch fact, not a swallow.
  const facts = service.endUpload("owner-a", handle.id);
  assert.equal(facts.state, "upload_complete");
  assert.equal(facts.upload_length_mismatch, true);
  const bad = service.request("owner-a", "https://svc", "POST", "/u", "1.1");
  service.addHeader("owner-a", bad.id, "content-length", "many");
  service.sendBodyChunk("owner-a", bad.id, [1]);
  expectPeerError(() => service.endUpload("owner-a", bad.id), "invalid-request");
});

// --- No implicit redirect or retry --------------------------------------------------

check("no-implicit-redirect", () => {
  const service = new HttpTestService(["https://svc"], serviceLimits());
  const handle = service.request("owner-a", "https://svc", "GET", "/old", "1.1");
  service.endUpload("owner-a", handle.id);
  const before = service.request_count;
  const response = service.deliverResponse("owner-a", handle.id, 302, [["location", "/new"]], []);
  assert.deepEqual(response.redirect, { status: 302, location: "/new" });
  assert.equal(response.redirect_incomplete, false);
  // The redirect surfaces as a fact only: no follow-up request exists.
  assert.equal(service.request_count, before);
  const facts = service.requestFacts("owner-a", handle.id);
  assert.deepEqual(facts.redirect, { status: 302, location: "/new" });
  // Only an explicit reissue mints the follow-up, linked and counted.
  const follow = service.reissue("owner-a", handle.id);
  assert.equal(service.request_count, before + 1);
  const followFacts = service.requestFacts("owner-a", follow.id);
  assert.equal(followFacts.supersedes, handle.id);
  assert.equal(followFacts.reissue_count, 1);
  assert.equal(followFacts.state, "upload_open");
  // Nothing is copied over implicitly: fresh headers and body.
  assert.equal(followFacts.header_count, 0);
  assert.equal(followFacts.upload_accepted, 0);
});

check("redirect-incomplete", () => {
  const service = new HttpTestService(["https://svc"], serviceLimits());
  const handle = service.request("owner-a", "https://svc", "GET", "/old", "1.1");
  service.endUpload("owner-a", handle.id);
  const response = service.deliverResponse("owner-a", handle.id, 302, [], []);
  assert.equal(response.redirect, null);
  assert.equal(response.redirect_incomplete, true);
});

check("no-implicit-retry", () => {
  const service = new HttpTestService(["https://svc"], serviceLimits());
  const handle = service.request("owner-a", "https://svc", "GET", "/flaky", "1.1");
  service.endUpload("owner-a", handle.id);
  service.deliverResponse("owner-a", handle.id, 500, [], [9]);
  // A failed response never reissues by itself.
  assert.equal(service.request_count, 1);
  assert.equal(service.requestFacts("owner-a", handle.id).state, "responded");
  const retry = service.reissue("owner-a", handle.id);
  assert.equal(service.requestFacts("owner-a", retry.id).reissue_count, 1);
});

check("reissue-counted-capped", () => {
  const service = new HttpTestService(["https://svc"], tinyLimits());
  const first = service.request("owner-a", "https://svc", "GET", "/a", "1.1");
  // Cannot reissue an open upload: finish explicitly first.
  expectPeerError(() => service.reissue("owner-a", first.id), "invalid-state");
  service.endUpload("owner-a", first.id);
  service.deliverResponse("owner-a", first.id, 302, [["location", "/b"]], []);
  const second = service.reissue("owner-a", first.id);
  service.endUpload("owner-a", second.id);
  service.deliverResponse("owner-a", second.id, 302, [["location", "/c"]], []);
  // Budget of 1 exhausted.
  expectPeerError(() => service.reissue("owner-a", second.id), "resource-limit");
  assert.equal(service.requestFacts("owner-a", second.id).reissue_count, 1);
});

// --- Response body: accepted vs consumed, EOF, partial -------------------------------

check("response-body-accepted-vs-consumed", () => {
  const service = new HttpTestService(["https://svc"], serviceLimits());
  const handle = service.request("owner-a", "https://svc", "GET", "/r", "1.1");
  // Reads before delivery: empty with eof:false, never implied EOF.
  const pending = service.readBodyChunk("owner-a", handle.id, 16);
  assert.deepEqual([...pending.bytes], []);
  assert.equal(pending.eof, false);
  assert.equal(service.responseFacts("owner-a", handle.id), null);
  service.endUpload("owner-a", handle.id);
  service.deliverResponse("owner-a", handle.id, 200, [["content-type", "bytes"]], [1, 2, 3, 4, 5]);
  const facts = service.requestFacts("owner-a", handle.id);
  assert.equal(facts.response_body_accepted, 5);
  assert.equal(facts.response_body_consumed, 0);
  const first = service.readBodyChunk("owner-a", handle.id, 2);
  assert.deepEqual([...first.bytes], [1, 2]);
  assert.equal(first.truncated, true);
  assert.equal(first.eof, false);
  assert.equal(first.consumed_total, 2);
  assert.equal(first.accepted_total, 5);
  const rest = service.readBodyChunk("owner-a", handle.id, 16);
  assert.deepEqual([...rest.bytes], [3, 4, 5]);
  assert.equal(rest.truncated, false);
  assert.equal(rest.eof, true);
  assert(Object.isFrozen(rest.bytes));
});

check("response-length-mismatch", () => {
  const service = new HttpTestService(["https://svc"], serviceLimits());
  const handle = service.request("owner-a", "https://svc", "GET", "/r", "1.1");
  service.endUpload("owner-a", handle.id);
  // Declared 99, delivered 3: delivered with an explicit mismatch fact.
  const response = service.deliverResponse(
    "owner-a",
    handle.id,
    200,
    [["content-length", "99"]],
    [1, 2, 3],
  );
  assert.equal(response.length_mismatch, true);
  assert.equal(response.body_accepted, 3);
  assert.equal(service.requestFacts("owner-a", handle.id).response_length_mismatch, true);
  const bad = service.request("owner-a", "https://svc", "GET", "/r", "1.1");
  service.endUpload("owner-a", bad.id);
  expectPeerError(
    () => service.deliverResponse("owner-a", bad.id, 200, [["content-length", "lots"]], [1]),
    "invalid-request",
  );
  // Malformed delivery stores nothing: still re-deliverable.
  assert.equal(service.requestFacts("owner-a", bad.id).response_delivered, false);
});

check("response-limits", () => {
  const service = new HttpTestService(["https://svc"], tinyLimits());
  const handle = service.request("owner-a", "https://svc", "GET", "/r", "1.1");
  service.endUpload("owner-a", handle.id);
  // Over-cap body rejects AND sets the sticky fact; the request stays
  // pending so a smaller response may be re-delivered.
  expectPeerError(
    () => service.deliverResponse("owner-a", handle.id, 200, [], [1, 2, 3, 4, 5]),
    "body-limit",
  );
  const facts = service.requestFacts("owner-a", handle.id);
  assert.equal(facts.response_body_truncated, true);
  assert.equal(facts.response_delivered, false);
  assert.equal(facts.state, "upload_complete");
  service.deliverResponse("owner-a", handle.id, 200, [], [1, 2]);
  // Over-cap response headers follow the same explicit pattern.
  const headed = service.request("owner-a", "https://svc", "GET", "/h", "1.1");
  service.endUpload("owner-a", headed.id);
  expectPeerError(
    () =>
      service.deliverResponse(
        "owner-a",
        headed.id,
        200,
        [
          ["x-trace", "a"],
          ["x-trace", "b"],
          ["x-trace", "c"],
        ],
        [],
      ),
    "header-limit",
  );
  assert.equal(service.requestFacts("owner-a", headed.id).response_headers_truncated, true);
  expectPeerError(
    () => service.deliverResponse("owner-a", headed.id, 418, [], []),
    "unsupported-capability",
  );
  expectPeerError(
    () => service.deliverResponse("owner-a", headed.id, 200, [["x-custom", "v"]], []),
    "unsupported-capability",
  );
  expectPeerError(
    () => service.deliverResponse("owner-a", headed.id, 200, [], "body"),
    "invalid-request",
  );
});

// --- Negative controls and cleanup ----------------------------------------------------

check("wrong-owner-closed-unknown", () => {
  const service = new HttpTestService(["https://svc"], serviceLimits());
  const handle = service.request("owner-a", "https://svc", "GET", "/r", "1.1");
  expectPeerError(() => service.addHeader("owner-b", handle.id, "x-trace", "v"), "wrong-owner");
  expectPeerError(() => service.endUpload("owner-b", handle.id), "wrong-owner");
  expectPeerError(() => service.requestFacts("owner-a", "q999"), "invalid-request");
  expectPeerError(() => service.deliverResponse("owner-a", "q999", 200, [], []), "invalid-request");
  expectPeerError(() => service.readBodyChunk("owner-a", handle.id, 0), "invalid-request");
  service.endUpload("owner-a", handle.id);
  service.deliverResponse("owner-a", handle.id, 200, [], [1, 2, 3]);
  expectPeerError(
    () => service.deliverResponse("owner-a", handle.id, 200, [], []),
    "invalid-state",
  );
  service.readBodyChunk("owner-a", handle.id, 1);
  // Close records the unread remainder in the receipt.
  const receipt = service.close("owner-a", handle.id);
  assert.equal(receipt.response_body_accepted, 3);
  assert.equal(receipt.response_body_consumed, 1);
  assert.equal(receipt.response_body_unread, 2);
  // Facts stay readable after close; operations reject as closed.
  assert.equal(service.requestFacts("owner-a", handle.id).state, "closed");
  expectPeerError(() => service.readBodyChunk("owner-a", handle.id, 4), "closed-handle");
  expectPeerError(() => service.reissue("owner-a", handle.id), "closed-handle");
  expectPeerError(() => service.close("owner-a", handle.id), "closed-handle");
});

check("caps-and-limits", () => {
  const service = new HttpTestService(["https://svc"], tinyLimits());
  for (let n = 0; n < 4; n += 1) {
    service.request("owner-a", "https://svc", "GET", `/r${n}`, "1.1");
  }
  expectPeerError(
    () => service.request("owner-a", "https://svc", "GET", "/full", "1.1"),
    "resource-limit",
  );
  expectPeerError(() => checkHttpLimits(null), "invalid-request");
  expectPeerError(() => checkHttpLimits({ ...tinyLimits(), max_requests: 0 }), "invalid-request");
  expectPeerError(
    () => checkHttpLimits({ ...tinyLimits(), max_chunk_bytes: 8, max_body_bytes: 4 }),
    "invalid-request",
  );
  expectPeerError(
    () => checkHttpLimits({ ...tinyLimits(), max_header_value_bytes: 65, max_header_bytes: 64 }),
    "invalid-request",
  );
  expectPeerError(() => new HttpTestService([], tinyLimits()), "invalid-request");
});

check("facts-inert", () => {
  const service = new HttpTestService(["https://svc"], serviceLimits());
  const handle = service.request("owner-a", "https://svc", "POST", "/u", "1.1");
  service.addHeader("owner-a", handle.id, "set-cookie", "a=1");
  service.addHeader("owner-a", handle.id, "set-cookie", "b=2");
  service.sendBodyChunk("owner-a", handle.id, [1, 2]);
  service.endUpload("owner-a", handle.id);
  service.deliverResponse("owner-a", handle.id, 302, [["location", "/new"]], [7]);
  checkFactsInert(service.requestFacts("owner-a", handle.id), "request facts");
  const response = service.responseFacts("owner-a", handle.id);
  assert(response !== null);
  checkFactsInert(response, "response facts");
  checkFactsInert(service.readBodyChunk("owner-a", handle.id, 16), "read result");
});

console.log(
  JSON.stringify({
    kind: "can.http-peer-http-check",
    schema_version: HTTP_PEER_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
