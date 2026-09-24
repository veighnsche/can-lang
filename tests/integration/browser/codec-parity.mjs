// Loads one browser-target wire-codec bundle in a named Playwright browser
// and reports the vector results. Usage:
//   node codec-parity.mjs <chromium|firefox|webkit> <bundle.js>
// Prints one JSON object: {browser, version, userAgent, rawJSON, results}.
const wanted = process.argv[2];
const bundle = process.argv[3];
if (wanted !== "chromium" && wanted !== "firefox" && wanted !== "webkit") {
  console.error(`unknown browser ${wanted}`);
  process.exit(2);
}
if (!bundle) {
  console.error("usage: node codec-parity.mjs <chromium|firefox|webkit> <bundle.js>");
  process.exit(2);
}
const playwright = await import("playwright");
const browser = await playwright[wanted].launch({ timeout: 60000 });
try {
  const page = await browser.newPage();
  await page.goto("about:blank");
  const rawJSON = await page.evaluate(() => typeof JSON.rawJSON === "function");
  if (!rawJSON) {
    console.log(
      JSON.stringify({
        browser: wanted,
        version: browser.version(),
        userAgent: await page.evaluate(() => navigator.userAgent),
        rawJSON: false,
        results: [],
      }),
    );
    process.exit(0);
  }
  await page.addScriptTag({ path: bundle });
  const results = await page.evaluate(() => globalThis.__canWireResults ?? null);
  console.log(
    JSON.stringify({
      browser: wanted,
      version: browser.version(),
      userAgent: await page.evaluate(() => navigator.userAgent),
      rawJSON: true,
      results,
    }),
  );
} finally {
  await browser.close();
}
