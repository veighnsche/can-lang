// Actual Bun.serve `routes` behavior, separate from the proposed Can adapter.
const observations: unknown[] = [];
const server = Bun.serve({
  hostname: "127.0.0.1", port: 0,
  routes: {
    "/invoices/new": {
      GET: (req) => Response.json({ route: "static", method: req.method, params: req.params, url: req.url }),
      PUT: (req) => Response.json({ route: "static", method: req.method, params: req.params, url: req.url }),
    },
    "/invoices/:slug": {
      POST: (req) => Response.json({ route: "slug", method: req.method, params: req.params, url: req.url }),
      PUT: (req) => Response.json({ route: "slug", method: req.method, params: req.params, url: req.url }),
    },
    "/tenants/:tenant_id/invoices/:invoice_id": { POST: (req) => Response.json({ route: "invoice", method: req.method, params: req.params, url: req.url }) },
  },
  fetch(req) { return Response.json({ route: "fallback", method: req.method, url: req.url }, { status: 404 }); },
});
const inputs: [string, string][] = [
  ["POST", "/tenants/1/invoices/7"],
  ["PATCH", "/tenants/1/invoices/7"],
  ["POST", "/tenants/01/invoices/7"],
  ["POST", "/tenants/%ZZ/invoices/7"],
  ["POST", "/tenants/%2F/invoices/7"],
  ["POST", "/tenants/%5C/invoices/7"],
  ["POST", "/tenants/%ED%A0%80/invoices/7"],
  ["POST", "/tenants/%31/invoices/7"],
  ["POST", "/tenants/%2E/invoices/7"],
  ["POST", "/tenants/%2E%2E/invoices/7"],
  ["POST", "/tenants/a\\b/invoices/7"],
  ["POST", "/tenants/1/invoices/7/"],
  ["POST", "/tenants/1/invoices/7/extra"],
  ["POST", "/invoices/new"],
  ["PUT", "/invoices/new"],
  ["GET", "/invoices/new"],
  ["POST", "/invoices/A%20B%25%E2%82%AC"],
];
try {
  for (const [method, path] of inputs) {
    const child = Bun.spawn(["curl", "--silent", "--show-error", "--path-as-is", "--globoff", "--max-time", "2", "--include", "--request", method, `http://127.0.0.1:${server.port}${path}`], { stdout: "pipe", stderr: "pipe" });
    const body = await new Response(child.stdout).text();
    observations.push({ method, path, exit: await child.exited, response: body });
  }
} finally { server.stop(true); }
console.log(JSON.stringify({ bun: Bun.version, revision: Bun.revision, observations }, null, 2));
