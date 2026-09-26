// Remote-Firefox helper (C01): native Firefox cannot launch on macOS
// 27, so the pinned container runner serves Firefox 141 over a
// Playwright launchServer websocket. When CAN_FIREFOX_WS is set,
// firefox legs connect to it instead of launching locally; otherwise
// they launch natively (the CI macos-15 path). Chromium and WebKit
// always launch natively. Container Firefox reaches the Mac leg
// servers through the provisioned loopback forwarders, so firefox
// legs keep the exact loopback base and guards: nothing else here
// differs between connect and launch mode.
export async function launchWanted(playwright, wanted) {
  if (wanted === "firefox" && process.env.CAN_FIREFOX_WS) {
    return { browser: await playwright.firefox.connect(process.env.CAN_FIREFOX_WS), remote: true };
  }
  return { browser: await playwright[wanted].launch({ timeout: 120000 }), remote: false };
}

// A connected leg must never close the browser: browser.close() would
// terminate the shared server for the legs after it. Each leg opens a
// fresh browser context instead, so legs stay isolated on the shared
// server; harnesses check the remote flag in their finally block.
