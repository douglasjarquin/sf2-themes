import assert from "node:assert/strict";
import { after, test as nodeTest } from "node:test";
import { localBrowser, Stagehand } from "@browserbasehq/stagehand";
import { chromium } from "playwright-core";

// Stagehand has no auto-waiting or expect(); every assertion here re-reads the DOM until it passes or the
// deadline, like Playwright's web-first assertions did.
const TIMEOUT = 5000;
// A hung CDP call fails its test instead of stalling the suite.
const TEST_TIMEOUT = 120000;
const DESKTOP = { width: 1280, height: 720 };

export const baseURL =
  process.env.SF2_E2E_BASE_URL ?? `http://127.0.0.1:${process.env.E2E_PORT ?? "4321"}/sf2-themes/`;

export const url = (path, base = baseURL) => new URL(path, base).href;

// playwright-core only provisions Chromium (Linux arm64 included); Stagehand drives it over CDP.
const chromePath = () => process.env.CHROME_PATH || chromium.executablePath();

// Docker caps /dev/shm at 64 MB, and Chrome aborts at startup once three suites' browsers fill it (font_data_service
// "No space left on device"). Playwright passed this flag on Linux for us; Stagehand's localBrowser does not.
const CHROME_ARGS = ["--disable-dev-shm-usage"];

// node:test fails a timed-out test but never aborts its pending awaits, so a wedged CDP call would skip the finally
// below and leave Chrome (its own process group) running. On abort, close the browser directly: browser.close() is
// bounded (a 5 s Browser.close, then SIGTERM and SIGKILL). after() holds the file open until those closes settle.
const REAP_TIMEOUT = 15000;
const reaping = new Set();
after(() => Promise.allSettled(reaping));

function reap(browser, stagehand) {
  let timer;
  const closed = Promise.race([
    Promise.allSettled([browser?.close(), stagehand?.close()]),
    new Promise((resolve) => {
      timer = setTimeout(resolve, REAP_TIMEOUT);
    }),
  ]).finally(() => {
    clearTimeout(timer);
    reaping.delete(closed);
  });
  reaping.add(closed);
}

// Each test gets its own browser, so localStorage and init scripts never leak between tests.
export function test(name, fn, { viewport = DESKTOP, initScript } = {}) {
  nodeTest(name, { timeout: TEST_TIMEOUT }, async (t) => {
    let browser;
    let stagehand;
    const onAbort = () => reap(browser, stagehand);
    t.signal.addEventListener("abort", onAbort, { once: true });
    try {
      browser = await localBrowser.launch({ headless: true, executablePath: chromePath(), viewport, args: CHROME_ARGS });
      t.signal.throwIfAborted();
      stagehand = await Stagehand.create({ browser, logging: { level: "warn" } });
      t.signal.throwIfAborted();
      const page = await browser.context.newPage();
      if (initScript) {
        await page.addInitScript(initScript);
      }
      await fn(page);
    } finally {
      t.signal.removeEventListener("abort", onAbort);
      if (t.signal.aborted) {
        // The abort may have fired before launch or Stagehand.create resolved, so reap what exists now.
        reap(browser, stagehand);
      } else {
        await stagehand?.close();
        await browser?.close();
      }
    }
  });
}

export const open = (page, path, base) => page.goto(url(path, base), { waitUntil: "load", timeout: 30000 });

export async function click(page, selector, index = 0) {
  await expectVisible(page, by(selector, undefined, { index }));
  await page.locator(selector).nth(index).click();
}

export async function eventually(read, check) {
  const deadline = Date.now() + TIMEOUT;
  for (;;) {
    try {
      check(await read());
      return;
    } catch (error) {
      if (Date.now() > deadline) {
        throw error;
      }
    }
    await new Promise((resolve) => setTimeout(resolve, 50));
  }
}

// by() narrows a CSS selector the way getByRole's name option did: aria-label or text without aria-hidden
// parts, a case-insensitive substring for a string and a test for a RegExp.
export const by = (css, name, { index = 0, visible = false } = {}) => ({ css, name, index, visible });

const serialize = (target) => {
  const { css, name, index = 0, visible = false } = typeof target === "string" ? { css: target } : target;
  const pattern = name instanceof RegExp ? { source: name.source, flags: name.flags } : name === undefined ? null : { text: name };
  return { css, pattern, index, visible };
};

function inspect([{ css, pattern, index, visible }, kind, arg]) {
  const normalize = (value) => value.replace(/\s+/g, " ").trim();
  const accessibleName = (el) => {
    const label = el.getAttribute("aria-label");
    if (label !== null) {
      return normalize(label);
    }
    const copy = el.cloneNode(true);
    copy.querySelectorAll("[aria-hidden='true']").forEach((hidden) => hidden.remove());
    return normalize(copy.textContent);
  };
  const shown = (el) => {
    const box = el.getBoundingClientRect();
    return el.checkVisibility({ visibilityProperty: true }) && box.width > 0 && box.height > 0;
  };
  let found = [...document.querySelectorAll(css)];
  if (pattern) {
    found = found.filter((el) => {
      const value = accessibleName(el);
      return pattern.text === undefined
        ? new RegExp(pattern.source, pattern.flags).test(value)
        : value.toLowerCase().includes(pattern.text.toLowerCase());
    });
  }
  if (visible) {
    found = found.filter(shown);
  }
  const el = found[index];
  switch (kind) {
    case "count":
      return found.length;
    case "text":
      return el ? normalize(el.textContent) : null;
    case "attr":
      return el ? el.getAttribute(arg) : null;
    case "value":
      return el ? el.value : null;
    case "css":
      return el ? getComputedStyle(el).getPropertyValue(arg) : null;
    case "visible":
      return Boolean(el) && shown(el);
    case "focused":
      return Boolean(el) && el === document.activeElement;
    case "inViewport": {
      if (!el) {
        return false;
      }
      const box = el.getBoundingClientRect();
      return box.bottom > 0 && box.right > 0 && box.top < innerHeight && box.left < innerWidth;
    }
  }
  throw new Error(`unknown read ${kind}`);
}

export const read = (page, target, kind, arg = null) => page.evaluate(inspect, [serialize(target), kind, arg]);

const expectRead = (page, target, kind, arg, check) => eventually(() => read(page, target, kind, arg), check);

export const expectText = (page, target, want) => expectRead(page, target, "text", null, (v) => assert.equal(v, want));
export const expectContainsText = (page, target, want) =>
  expectRead(page, target, "text", null, (v) =>
    want instanceof RegExp ? assert.match(v ?? "", want) : assert.ok(v?.includes(want), `${JSON.stringify(v)} lacks ${JSON.stringify(want)}`),
  );
export const expectLacksText = (page, target, unwanted) =>
  expectRead(page, target, "text", null, (v) => {
    assert.notEqual(v, null, "element not found");
    assert.doesNotMatch(v, unwanted instanceof RegExp ? unwanted : new RegExp(RegExp.escape(unwanted)));
  });
export const expectAttr = (page, target, name, want) => expectRead(page, target, "attr", name, (v) => assert.equal(v, want));
export const expectValue = (page, target, want) => expectRead(page, target, "value", null, (v) => assert.equal(v, want));
export const expectCSS = (page, target, property, want) => expectRead(page, target, "css", property, (v) => assert.equal(v, want));
export const expectCount = (page, target, want) => expectRead(page, target, "count", null, (v) => assert.equal(v, want));
export const expectVisible = (page, target) => expectRead(page, target, "visible", null, (v) => assert.equal(v, true));
export const expectHidden = (page, target) => expectRead(page, target, "visible", null, (v) => assert.equal(v, false));
export const expectFocused = (page, target) => expectRead(page, target, "focused", null, (v) => assert.equal(v, true));
export const expectInViewport = (page, target) => expectRead(page, target, "inViewport", null, (v) => assert.equal(v, true));
export const expectTitle = (page, want) => eventually(() => page.title(), (v) => assert.equal(v, want));
export const expectURLContains = (page, want) => eventually(() => page.url(), (v) => assert.ok(v.includes(want), `${v} lacks ${want}`));
