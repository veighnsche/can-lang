import { originalEnvironment } from "../../runtime/environment.ts";
// Fixed development qualification command, never an arbitrary TS entry point.
console.log(JSON.stringify({
  kind: "can.development-runtime-check", schemaVersion: 1,
  bun: Bun.version, revision: Bun.revision, platform: process.platform,
  architecture: process.arch, executable: process.execPath,
  rawJSON: typeof JSON.rawJSON, fromAsync: typeof Array.fromAsync,
  arguments: Bun.argv.slice(2),
  environmentProbe: originalEnvironment("CAN_DISTRIBUTION_PROBE") ?? null,
  originalHomePresent: originalEnvironment("HOME") !== undefined,
}));
