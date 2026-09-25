export const browserPolicy =
  "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'; navigate-to 'self'";
// Replaced digest URLs stay servable for at least seven days past
// replacement. A route exactly at the bound still serves.
const assetRetentionMs = 7 * 24 * 60 * 60 * 1000;
export type ServedAsset = Readonly<{
  route: string;
  digest: string;
  mediaType: string;
  file: string;
  integrity: string;
}>;
export type BrowserTable = Readonly<{
  buildId: string;
  entry: string;
  table: string;
  files: readonly ServedAsset[];
}>;
export type AssetTable = Readonly<{
  htmx: ServedAsset;
  project: readonly ServedAsset[];
  browser?: BrowserTable;
}>;
export type AssetServer = Readonly<{
  serve(request: Request): Promise<Response | undefined>;
  browserScript?: string;
}>;
type LedgerEntry = Readonly<{
  digest: string;
  route: string;
  mediaType: string;
  file: string;
  replacedAt: number;
}>;
const htmxRoute = "/__can/assets/htmx-4.0.0.min.js";
const htmxIntegrity = "sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc";

function hex(bytes: ArrayBuffer): string {
  let out = "";
  for (const byte of new Uint8Array(bytes)) out += byte.toString(16).padStart(2, "0");
  return out;
}
function integrity(bytes: ArrayBuffer): string {
  let binary = "";
  for (const byte of new Uint8Array(bytes)) binary += String.fromCharCode(byte);
  return "sha384-" + btoa(binary);
}
function text(status: 400 | 404 | 405): Response {
  const headers = new Headers({
    "content-type": "text/plain; charset=utf-8",
    "x-content-type-options": "nosniff",
  });
  if (status === 405) headers.set("allow", "GET, HEAD");
  return new Response(
    status === 400 ? "Bad Request" : status === 404 ? "Not Found" : "Method Not Allowed",
    { status, headers },
  );
}
function safeFile(file: string, digest: string, name: string): boolean {
  return (
    file === `assets/${digest}/${name}` &&
    /^[0-9a-f]{64}$/.test(digest) &&
    /^[A-Za-z0-9][A-Za-z0-9._-]*$/.test(name)
  );
}
function browserExtension(route: string): string | undefined {
  for (const ext of [".js.map", ".js", ".json"]) {
    if (route.endsWith(ext)) return ext;
  }
  return undefined;
}
function safeBrowserFile(asset: ServedAsset): boolean {
  const ext = browserExtension(asset.route);
  if (ext === undefined) return false;
  const digest = asset.route.slice("/__can/assets/".length, -ext.length);
  if (!/^[0-9a-f]{64}$/.test(digest) || digest !== asset.digest) return false;
  if (asset.route !== `/__can/assets/${digest}${ext}`) return false;
  const name = asset.file.slice(asset.file.lastIndexOf("/") + 1);
  const media = ext === ".js" ? "text/javascript" : "application/json";
  return asset.mediaType === media && asset.integrity === "" && safeFile(asset.file, digest, name);
}
function cached(
  request: Request,
  bytes: Uint8Array<ArrayBuffer>,
  mediaType: string,
  digest: string,
): Response {
  const headers = new Headers({
    "content-type": mediaType,
    etag: `"${digest}"`,
    "cache-control": "public, max-age=31536000, immutable",
    "x-content-type-options": "nosniff",
  });
  if (request.headers.get("if-none-match") === `"${digest}"`)
    return new Response(null, { status: 304, headers });
  return new Response(request.method === "HEAD" ? null : bytes, { status: 200, headers });
}

// createAssets serves only the compiler-built table. Paths are exact matches
// against that table; nothing reads an arbitrary file or lists a directory.
// The optional browser section publishes the verified paired build, and the
// durable ledger beside the generations keeps replaced digest URLs servable
// for at least seven days past replacement.
export function createAssets(table: AssetTable, root: URL): AssetServer {
  if (
    table.htmx.route !== htmxRoute ||
    table.htmx.integrity !== htmxIntegrity ||
    table.htmx.mediaType !== "text/javascript" ||
    !safeFile(table.htmx.file, table.htmx.digest, "htmx-4.0.0.min.js")
  ) {
    throw new TypeError("invalid pinned htmx asset");
  }
  const routes = new Map<string, ServedAsset>([[table.htmx.route, table.htmx]]);
  for (const asset of table.project) {
    const name = asset.route.slice(asset.route.lastIndexOf("/") + 1);
    if (
      !asset.route.startsWith(`/__can/project/${asset.digest}/`) ||
      asset.integrity !== "" ||
      !safeFile(asset.file, asset.digest, name) ||
      routes.has(asset.route)
    ) {
      throw new TypeError("invalid project asset");
    }
    routes.set(asset.route, asset);
  }
  let browserScript: string | undefined;
  if (table.browser !== undefined) {
    const section = table.browser;
    if (!/^[0-9a-f]{64}$/.test(section.buildId) || section.files.length === 0) {
      throw new TypeError("invalid paired browser build");
    }
    const served = new Set<string>();
    for (const asset of section.files) {
      if (!safeBrowserFile(asset) || routes.has(asset.route) || served.has(asset.route)) {
        throw new TypeError("invalid paired browser asset");
      }
      served.add(asset.route);
      routes.set(asset.route, asset);
    }
    if (
      !served.has(section.entry) ||
      !section.entry.endsWith(".js") ||
      section.entry.endsWith(".js.map")
    ) {
      throw new TypeError("invalid paired browser entry");
    }
    if (!served.has(section.table) || !section.table.endsWith(".json")) {
      throw new TypeError("invalid paired browser table");
    }
    browserScript = section.entry;
  }
  // The durable retention store lives two levels above the generation
  // (dist/builds/<id>/ -> dist/); a missing ledger means no retention.
  const durable = new URL("../../", root);
  async function retained(path: string): Promise<LedgerEntry | undefined> {
    const ext = browserExtension(path);
    if (ext === undefined || !path.startsWith("/__can/assets/")) return undefined;
    const digest = path.slice("/__can/assets/".length, -ext.length);
    if (!/^[0-9a-f]{64}$/.test(digest)) return undefined;
    let ledger: unknown;
    try {
      ledger = await Bun.file(new URL("assets.json", durable)).json();
    } catch {
      return undefined;
    }
    if (
      typeof ledger !== "object" ||
      ledger === null ||
      (ledger as { schemaVersion?: unknown }).schemaVersion !== 1 ||
      (ledger as { kind?: unknown }).kind !== "can.asset-ledger" ||
      !Array.isArray((ledger as { retained?: unknown }).retained)
    )
      return undefined;
    for (const entry of (ledger as { retained: unknown[] }).retained) {
      if (typeof entry !== "object" || entry === null) continue;
      const candidate = entry as Record<string, unknown>;
      if (
        candidate["digest"] !== digest ||
        candidate["route"] !== path ||
        candidate["file"] !== `assets/${digest}${ext}` ||
        candidate["mediaType"] !== (ext === ".js" ? "text/javascript" : "application/json") ||
        typeof candidate["replacedAt"] !== "number" ||
        !Number.isFinite(candidate["replacedAt"])
      )
        continue;
      return candidate as unknown as LedgerEntry;
    }
    return undefined;
  }
  return Object.freeze({
    ...(browserScript === undefined ? {} : { browserScript }),
    async serve(request: Request): Promise<Response | undefined> {
      let path: string;
      try {
        path = decodeURIComponent(new URL(request.url).pathname);
      } catch {
        return text(400);
      }
      if (path !== "/__can" && !path.startsWith("/__can/")) return undefined;
      if (path.includes("\\") || path.split("/").includes("..")) return text(404);
      const asset = routes.get(path);
      if (asset !== undefined) {
        if (request.method !== "GET" && request.method !== "HEAD") return text(405);
        let bytes: Uint8Array<ArrayBuffer>;
        try {
          bytes = new Uint8Array(await Bun.file(new URL(asset.file, root)).bytes());
        } catch {
          return text(404);
        }
        const digest = hex(await crypto.subtle.digest("SHA-256", bytes));
        if (digest !== asset.digest) return text(404);
        if (
          asset.integrity !== "" &&
          integrity(await crypto.subtle.digest("SHA-384", bytes)) !== asset.integrity
        )
          return text(404);
        return cached(request, bytes, asset.mediaType, digest);
      }
      const entry = await retained(path);
      if (entry === undefined) return text(404);
      if (request.method !== "GET" && request.method !== "HEAD") return text(405);
      if (Date.now() - entry.replacedAt > assetRetentionMs) return text(404);
      let bytes: Uint8Array<ArrayBuffer>;
      try {
        bytes = new Uint8Array(await Bun.file(new URL(entry.file, durable)).bytes());
      } catch {
        return text(404);
      }
      const digest = hex(await crypto.subtle.digest("SHA-256", bytes));
      if (digest !== entry.digest) return text(404);
      return cached(request, bytes, entry.mediaType, digest);
    },
  });
}
