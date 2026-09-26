// Remote-Firefox helpers (C01): native Firefox cannot launch on macOS
// 27, so the pinned container runner serves Firefox 141 over a
// Playwright launchServer websocket. When CAN_FIREFOX_WS is set,
// firefox legs connect to it instead of launching locally; the
// container reaches the Mac loopback servers through a host alias
// (default host.docker.internal, overridable via
// CAN_FIREFOX_HOST_ALIAS). Chromium and WebKit always launch natively,
// and firefox legs launch natively when the env var is unset (the CI
// macos-15 path).
export const firefoxHostAlias = () => process.env.CAN_FIREFOX_HOST_ALIAS || "host.docker.internal";

// firefoxEndpoint returns the container ws endpoint for firefox legs
// in connect mode, or "" for every native leg.
export const firefoxEndpoint = (wanted) => (wanted === "firefox" ? process.env.CAN_FIREFOX_WS || "" : "");

// launchWanted connects firefox legs to the shared container server
// and launches every other leg natively. A connected leg must never
// close the browser: browser.close() would terminate the shared
// server for the legs after it. Each leg opens a fresh browser
// context instead, so legs stay isolated on the shared server.
export async function launchWanted(playwright, wanted) {
  const endpoint = firefoxEndpoint(wanted);
  if (endpoint) return { browser: await playwright.firefox.connect(endpoint), remote: true };
  return { browser: await playwright[wanted].launch({ timeout: 120000 }), remote: false };
}

// remoteBase rewrites the loopback base the Go driver passes as a CLI
// arg to the host alias, but only for firefox legs in connect mode:
// the container's own 127.0.0.1 is not the Mac's loopback.
export function remoteBase(wanted, base) {
  if (!firefoxEndpoint(wanted)) return base;
  const rewritten = new URL(base);
  if (rewritten.hostname === "127.0.0.1" || rewritten.hostname === "localhost") {
    rewritten.hostname = firefoxHostAlias();
  }
  return rewritten.toString().replace(/\/+$/, "");
}

// originAllowed is the loopback guard shared by the route
// interception and the in-harness network asserts: loopback always
// passes, and the host alias passes only for firefox legs in connect
// mode. Native legs stay exactly as strict as before.
export function originAllowed(wanted, hostname) {
  if (hostname === "127.0.0.1" || hostname === "localhost") return true;
  return !!firefoxEndpoint(wanted) && hostname === firefoxHostAlias();
}
