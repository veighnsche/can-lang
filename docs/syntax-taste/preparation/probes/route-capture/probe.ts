// Disposable Bun 1.4.2 / URL observations for DI-09a. No production adapter.
const samples = [
  "/tenants/1/invoices/7",
  "/tenants/01/invoices/7",
  "/tenants/+1/invoices/7",
  "/tenants/-1/invoices/7",
  "/tenants/9223372036854775808/invoices/7",
  "/tenants/1/invoices/",
  "/tenants/1/invoices/7/extra",
  "/tenants/1/invoices/7/",
  "/tenants/%2F/invoices/7",
  "/tenants/%5C/invoices/7",
  "/tenants/a%252Fb/invoices/7",
  "/tenants/a%20b/invoices/7",
  "/tenants/%E2%82%AC/invoices/7",
  "/tenants/%ED%A0%80/invoices/7",
  "/tenants/%/invoices/7",
  "/tenants/%ZZ/invoices/7",
  "/tenants/./invoices/7",
  "/tenants/%2E/invoices/7",
  "/tenants/%2E%2E/invoices/7",
  "/tenants/a%2Fb/invoices/7",
  "/tenants/a\\b/invoices/7",
  "/tenants/a%00b/invoices/7",
  "/tenants/%25/invoices/7",
];

function observation(input: string) {
  try {
    const url = new URL(input, "http://127.0.0.1:8080");
    let wholeDecode: string;
    try {
      wholeDecode = decodeURIComponent(url.pathname);
    } catch (cause) {
      wholeDecode = `throws:${(cause as Error).name}`;
    }
    const segmentDecodes = url.pathname.split("/").map((part) => {
      try {
        return decodeURIComponent(part);
      } catch (cause) {
        return `throws:${(cause as Error).name}`;
      }
    });
    return { pathname: url.pathname, wholeDecode, segmentDecodes };
  } catch (cause) {
    return { urlError: (cause as Error).name };
  }
}

const parsed = samples.map((input) => ({ input, ...observation(input) }));
const request = samples.map((input) => {
  try {
    return { input, url: new Request(new URL(input, "http://localhost")).url };
  } catch (cause) {
    return { input, requestError: (cause as Error).name };
  }
});

const served: { request: string; method: string; url: string }[] = [];
const server = Bun.serve({
  hostname: "127.0.0.1",
  port: 0,
  fetch(req) {
    served.push({ request: req.headers.get("x-probe-path") ?? "", method: req.method, url: req.url });
    return new Response("ok");
  },
});

try {
  for (const input of samples) {
    const child = Bun.spawn([
      "curl", "--silent", "--show-error", "--output", "/dev/null", "--path-as-is",
      "--globoff", "--max-time", "2", "--header", `X-Probe-Path: ${input}`,
      `http://127.0.0.1:${server.port}${input}`,
    ], { stdout: "ignore", stderr: "pipe" });
    const code = await child.exited;
    if (code !== 0) served.push({ request: input, method: "curl-error", url: await new Response(child.stderr).text() });
  }
  const methodPaths = ["/tenants/1/invoices/7", "/tenants/%ZZ/invoices/7"];
  for (const input of methodPaths) {
    const child = Bun.spawn([
      "curl", "--silent", "--show-error", "--output", "/dev/null", "--path-as-is",
      "--globoff", "--max-time", "2", "--request", "PATCH", "--header", `X-Probe-Path: ${input}`,
      `http://127.0.0.1:${server.port}${input}`,
    ], { stdout: "ignore", stderr: "pipe" });
    const code = await child.exited;
    if (code !== 0) served.push({ request: input, method: "curl-error", url: await new Response(child.stderr).text() });
  }
} finally {
  server.stop(true);
}

console.log(JSON.stringify({ bun: Bun.version, revision: Bun.revision, parsed, request, served }, null, 2));
