import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, type FailureShape } from "../domain.ts";
import {
  createFormActions,
  createRouter,
  dispatch,
  type FormActionSite,
} from "../platform/router.ts";
import { snapshotRequest } from "../platform/http.ts";
import { createHTML } from "../platform/html.ts";
import { success, value } from "../completion.ts";
import { record, array, dataProperty, dataArray } from "../data.ts";
import type { FormSchema } from "../platform/form.ts";
const hash = (kind: string, name: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, name]))
    .digest("hex");
const shape = (
  kind: string,
  declaration: string,
  fields: { name: string; type: string }[] = [],
): FailureShape => ({
  identity: hash(kind, declaration),
  kind,
  declaration,
  fields,
  arguments: [],
  leaves: [],
  inputs: [],
  errors: [],
});
const str = shape("primitive", "str");
const declarations = catalogue.errors.filter((e) =>
  ["http::invalid_route", "http::duplicate_route", "http::ambiguous_route"].includes(e.name),
);
const errors = declarations.map((e) =>
  shape(
    "error",
    e.identity,
    e.fields.map((f) => ({ name: f.name, type: str.identity })),
  ),
);
const domain = createDomainRuntime({
  declarations: declarations.map((e) => ({ ...e, parameters: 0 })),
  shapes: [str, ...errors],
});
const forms = createFormActions(domain, { invalidRoute: errors[0]!.identity });
const routing = createRouter(domain, {
  invalid: errors[0]!.identity,
  duplicate: errors[1]!.identity,
  ambiguous: errors[2]!.identity,
});
const htmlDecls = catalogue.errors.filter((e) => e.name === "html::invalid_structure");
const htmlShapes = htmlDecls.map((e) =>
  shape(
    "error",
    e.identity,
    e.fields.map((f) => ({ name: f.name, type: str.identity })),
  ),
);
const htmlDomain = createDomainRuntime({
  declarations: htmlDecls.map((e) => ({ ...e, parameters: 0 })),
  shapes: [str, ...htmlShapes],
});
const html = createHTML(htmlDomain, {
  structure: htmlShapes[0]!.identity,
  url: "unused",
  target: "unused",
  interval: "unused",
});
const schema: FormSchema = {
  root: "wire",
  fields: [
    { name: "customer", kind: "str" },
    {
      name: "lines",
      kind: "rows",
      rows: {
        row: "row",
        order: "lines_order",
        collection: "rows",
        item: "item",
        fields: [{ name: "sku", kind: "str" }],
      },
    },
  ],
};
const site: FormActionSite = {
  action: "web::save_invoice",
  method: "POST",
  path: "/invoices/save",
  form: schema,
  cases: [
    { leaf: "saved", status: 200 },
    { leaf: "rejected", status: 422 },
    { leaf: "stale", status: 409 },
  ],
  rejected: "rejected",
  rawEntry: "raw",
  issue: "issue",
};
const fragment = async (text: string) => value(await html.fragment([value(await html.text(text))]));
const calls: string[] = [];
const handler = async (wire: unknown) => {
  calls.push("handler:" + JSON.stringify(dataProperty(dataProperty(wire, "lines"), "order")));
  return success(record("saved", [["label", "ok"]]));
};
const outcome = async (_result: unknown) => {
  calls.push("outcome");
  return success(await fragment("saved!"));
};
const structural = async (bad: unknown) => {
  const raw = dataArray(dataProperty(bad, "raw"))
    .map((entry) => dataProperty(entry, "value") as string)
    .join("|");
  calls.push("structural:" + raw);
  return success(await fragment("bad:" + raw));
};
async function post(text: string, contentType = "application/x-www-form-urlencoded") {
  const route = value(await forms.serve("web::save_invoice", outcome, structural, site, handler));
  const router = value(await routing.make(array([route])));
  const snap = await snapshotRequest(
    new Request("http://test.local/invoices/save", {
      method: "POST",
      headers: { "content-type": contentType },
      body: text,
    }),
    65536,
  );
  if (snap.kind !== "request") throw Error("snapshot rejected");
  const response = value(await dispatch(router, snap.value));
  return response;
}
test("serve binds the three callables into a routed POST", async () => {
  calls.length = 0;
  const response = await post("customer=ann&lines_order=a&lines%5Ba%5D%5Bsku%5D=x");
  expect(response.status).toBe(200);
  expect(response.headers.get("content-type")).toBe("text/html; charset=utf-8");
  expect(await response.text()).toContain("saved!");
  expect(calls).toEqual(['handler:["a"]', "outcome"]);
});
test("outcome leaves map to case statuses", async () => {
  for (const [leaf, status, label] of [
    ["saved", 200, "saved"],
    ["rejected", 422, "rejected"],
    ["stale", 409, "stale"],
  ] as const) {
    const local = async () => success(record(leaf, [["label", label]]));
    const route = value(await forms.serve("web::save_invoice", outcome, structural, site, local));
    const router = value(await routing.make(array([route])));
    const snap = await snapshotRequest(
      new Request("http://test.local/invoices/save", {
        method: "POST",
        headers: { "content-type": "application/x-www-form-urlencoded" },
        body: "customer=ann",
      }),
      65536,
    );
    if (snap.kind !== "request") throw Error("snapshot rejected");
    const response = value(await dispatch(router, snap.value));
    expect(response.status).toBe(status);
  }
});
test("structural violations reach the 422 renderer with raw and issues", async () => {
  calls.length = 0;
  const seen: unknown[] = [];
  const spy = async (bad: unknown) => {
    seen.push(bad);
    return structural(bad);
  };
  const route = value(await forms.serve("web::save_invoice", outcome, spy, site, handler));
  const router = value(await routing.make(array([route])));
  const snap = await snapshotRequest(
    new Request("http://test.local/invoices/save", {
      method: "POST",
      headers: { "content-type": "application/x-www-form-urlencoded" },
      body: "customer=ann&lines_order=a&lines_order=a&lines%5Ba%5D%5Bsku%5D=x",
    }),
    65536,
  );
  if (snap.kind !== "request") throw Error("snapshot rejected");
  const response = value(await dispatch(router, snap.value));
  expect(response.status).toBe(422);
  expect(calls).toEqual(["structural:ann|a|a|x"]);
  const issues = dataArray(dataProperty(seen[0], "issues")).map(
    (entry) => [dataProperty(entry, "name"), dataProperty(entry, "reason")] as [string, string],
  );
  expect(issues).toEqual([["lines_order", "form_repeated"]]);
});
test("ingress failures stay compiler-owned", async () => {
  calls.length = 0;
  const media = await post("customer=ann", "application/json");
  expect(media.status).toBe(415);
  expect(await media.text()).toBe("Unsupported Media Type");
  const encoding = await post("customer=%");
  expect(encoding.status).toBe(400);
  expect(calls).toEqual([]);
});
test("serve validates its contract", async () => {
  await expect(
    forms.serve("web::save_invoice", outcome, structural, null, handler),
  ).rejects.toThrow(TypeError);
  await expect(
    forms.serve("web::save_invoice", outcome, structural, { ...site, method: "GET" }, handler),
  ).rejects.toThrow(TypeError);
  await expect(forms.serve("web::other", outcome, structural, site, handler)).rejects.toThrow(
    TypeError,
  );
  // The frozen site carries the fully qualified action identity while the
  // invocation carries the authored spelling; the tails must agree.
  const bare = await forms.serve("save_invoice", outcome, structural, site, handler);
  expect(bare.kind).toBe("ok");
  await expect(forms.serve("save_other", outcome, structural, site, handler)).rejects.toThrow(
    TypeError,
  );
  await expect(forms.serve("web::save_invoice", outcome, structural, site, "nope")).rejects.toThrow(
    TypeError,
  );
  const badPath = await forms.serve(
    "web::save_invoice",
    outcome,
    structural,
    { ...site, path: "nope" },
    handler,
  );
  expect(badPath.kind).toBe("domain");
  // An unknown result leaf is a compiler/handler defect, not a status.
  const evil = value(
    await forms.serve("web::save_invoice", outcome, structural, site, async () =>
      success(record("bogus", [])),
    ),
  );
  const evilRouter = value(await routing.make(array([evil])));
  const snap = await snapshotRequest(
    new Request("http://test.local/invoices/save", {
      method: "POST",
      headers: { "content-type": "application/x-www-form-urlencoded" },
      body: "customer=ann",
    }),
    65536,
  );
  if (snap.kind !== "request") throw Error("snapshot rejected");
  const evilResult = await dispatch(evilRouter, snap.value);
  expect(evilResult.kind).not.toBe("ok");
});
test("browser observes escaped 422 text in submitted order over live HTTP", async () => {
  const route = value(await forms.serve("web::save_invoice", outcome, structural, site, handler));
  const router = value(await routing.make(array([route])));
  const server = Bun.serve({
    hostname: "127.0.0.1",
    port: 0,
    async fetch(request) {
      const snap = await snapshotRequest(request, 65536);
      if (snap.kind === "rejected") return new Response(null, { status: snap.status });
      return value(await dispatch(router, snap.value));
    },
  });
  try {
    const evil = encodeURIComponent("<script>alert(1)</script>");
    const response = await fetch(new URL("/invoices/save", server.url), {
      method: "POST",
      headers: { "content-type": "application/x-www-form-urlencoded" },
      body: `customer=${evil}&lines_order=b&lines_order=a&lines_order=a&lines%5Bb%5D%5Bsku%5D=x&lines%5Ba%5D%5Bsku%5D=y`,
    });
    expect(response.status).toBe(422);
    expect(response.headers.get("content-type")).toBe("text/html; charset=utf-8");
    const text = await response.text();
    expect(text).not.toContain("<script>");
    expect(text).toContain("&lt;script&gt;");
    // Raw values render escaped in submitted document order.
    const script = text.indexOf("&lt;script&gt;");
    const tail = text.indexOf("|b|a|a|x|y");
    expect(script).toBeGreaterThan(-1);
    expect(text.indexOf("bad:")).toBeGreaterThan(-1);
    expect(tail).toBeGreaterThan(script);
  } finally {
    await server.stop(true);
  }
});
test("browser observes outcome fragments with case statuses over live HTTP", async () => {
  const route = value(await forms.serve("web::save_invoice", outcome, structural, site, handler));
  const router = value(await routing.make(array([route])));
  const server = Bun.serve({
    hostname: "127.0.0.1",
    port: 0,
    async fetch(request) {
      const snap = await snapshotRequest(request, 65536);
      if (snap.kind === "rejected") return new Response(null, { status: snap.status });
      return value(await dispatch(router, snap.value));
    },
  });
  try {
    const response = await fetch(new URL("/invoices/save", server.url), {
      method: "POST",
      headers: { "content-type": "application/x-www-form-urlencoded" },
      body: "customer=<b>ann</b>&lines_order=a&lines%5Ba%5D%5Bsku%5D=x",
    });
    expect(response.status).toBe(200);
    const text = await response.text();
    expect(text).toContain("saved!");
  } finally {
    await server.stop(true);
  }
});
