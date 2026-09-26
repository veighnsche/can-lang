import { chromium } from '/Users/vince/Projects/can-lang/tests/integration/browser/node_modules/playwright/index.mjs';
// Probe the exact native operations used by runtime/platform/browser.ts:320-326,433.
// This is a DOM semantic probe, not a generated Can end-to-end test.
const browser = await chromium.launch({headless:true});
const page = await browser.newPage();
await page.setContent('<input id="text" value="initial"><input id="check" type="checkbox">');
await page.locator('#text').fill('user edit');
const valueResult = await page.evaluate(() => {
  const input = document.getElementById('text');
  input.setAttribute('value', 'normalized');
  return {attribute:input.getAttribute('value'), actual:input.value};
});
await page.evaluate(() => {
  window.snapshots = [];
  document.getElementById('check').addEventListener('change', event => {
    window.snapshots.push({kind:event.type, target:event.target.id, value:event.target.value, key:typeof event.key === 'string' ? event.key : ''});
  });
});
await page.locator('#check').click();
await page.locator('#check').click();
console.log(JSON.stringify({valueResult, checkboxSnapshots:await page.evaluate(() => window.snapshots)},null,2));
await browser.close();
