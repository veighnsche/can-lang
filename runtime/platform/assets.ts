export const browserPolicy = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'; navigate-to 'self'";
export type ServedAsset = Readonly<{route: string; digest: string; mediaType: string; file: string; integrity: string}>;
export type AssetTable = Readonly<{htmx: ServedAsset; project: readonly ServedAsset[]}>;
export type AssetServer = Readonly<{serve(request: Request): Promise<Response | undefined>}>;
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
 const headers = new Headers({"content-type": "text/plain; charset=utf-8", "x-content-type-options": "nosniff"});
 if (status === 405) headers.set("allow", "GET, HEAD");
 return new Response(status === 400 ? "Bad Request" : status === 404 ? "Not Found" : "Method Not Allowed", {status, headers});
}
function safeFile(file: string, digest: string, name: string): boolean {
 return file === `assets/${digest}/${name}` && /^[0-9a-f]{64}$/.test(digest) && /^[A-Za-z0-9][A-Za-z0-9._-]*$/.test(name);
}

// createAssets serves only the compiler-built table. Paths are exact matches
// against that table; nothing reads an arbitrary file or lists a directory.
export function createAssets(table: AssetTable, root: URL): AssetServer {
 if (table.htmx.route !== htmxRoute || table.htmx.integrity !== htmxIntegrity || table.htmx.mediaType !== "text/javascript" || !safeFile(table.htmx.file, table.htmx.digest, "htmx-4.0.0.min.js")) {
  throw new TypeError("invalid pinned htmx asset");
 }
 const routes = new Map<string, ServedAsset>([[table.htmx.route, table.htmx]]);
 for (const asset of table.project) {
  const name = asset.route.slice(asset.route.lastIndexOf("/") + 1);
  if (!asset.route.startsWith(`/__can/project/${asset.digest}/`) || asset.integrity !== "" || !safeFile(asset.file, asset.digest, name) || routes.has(asset.route)) {
   throw new TypeError("invalid project asset");
  }
  routes.set(asset.route, asset);
 }
 return Object.freeze({
  async serve(request: Request): Promise<Response | undefined> {
   let path: string;
   try { path = decodeURIComponent(new URL(request.url).pathname); }
   catch { return text(400); }
   if (path !== "/__can" && !path.startsWith("/__can/")) return undefined;
   if (path.includes("\\") || path.split("/").includes("..")) return text(404);
   const asset = routes.get(path);
   if (!asset) return text(404);
   if (request.method !== "GET" && request.method !== "HEAD") return text(405);
   let bytes: Uint8Array<ArrayBuffer>;
   try { bytes = new Uint8Array(await Bun.file(new URL(asset.file, root)).bytes()); }
   catch { return text(404); }
   const digest = hex(await crypto.subtle.digest("SHA-256", bytes));
   if (digest !== asset.digest) return text(404);
   if (asset.integrity !== "" && integrity(await crypto.subtle.digest("SHA-384", bytes)) !== asset.integrity) return text(404);
   const headers = new Headers({"content-type": asset.mediaType, etag: `"${digest}"`, "cache-control": "public, max-age=31536000, immutable", "x-content-type-options": "nosniff"});
   if (request.headers.get("if-none-match") === `"${digest}"`) return new Response(null, {status: 304, headers});
   return new Response(request.method === "HEAD" ? null : bytes, {status: 200, headers});
  }
 });
}
