// Browser-profile asset server. The browser serves no routes; asset bytes
// arrive over HTTP from the verified server instead of disk. The stub
// below only satisfies linking and fails closed if serving is ever
// attempted.
import type { AssetServer, AssetTable } from "../platform/assets.ts";

export function createAssets(_table: AssetTable, _root: URL): AssetServer {
  return Object.freeze({
    async serve(_request: Request): Promise<Response | undefined> {
      throw new TypeError("asset serving is unavailable in the browser profile");
    },
  });
}
