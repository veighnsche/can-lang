// Browser entry for the shared wire-codec vectors. Bundled with
// `bun build --target browser`, loaded once per named browser by
// codec-parity.mjs, and read back through `__canWireResults`.
import { runBrowserWireVectors } from "../../../runtime/test/browser-wire-vectors.ts";

(globalThis as Record<string, unknown>).__canWireResults = runBrowserWireVectors();
