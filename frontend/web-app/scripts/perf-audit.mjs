import { spawn } from "node:child_process";
import { existsSync } from "node:fs";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import net from "node:net";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { gzipSync } from "node:zlib";
import { webcrypto } from "node:crypto";
import { chromium } from "playwright-core";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const appDir = path.resolve(scriptDir, "..");
const distDir = path.join(appDir, "dist");
const manifestPath = path.join(distDir, ".vite", "manifest.json");
const reportDir = path.join(appDir, "reports");
const reportJsonPath = path.join(reportDir, "frontend-perf-report.json");
const reportCsvPath = path.join(reportDir, "frontend-perf-report.csv");
const npmCommand = process.platform === "win32" ? "npm.cmd" : "npm";
const previewHost = "127.0.0.1";
const previewStartPort = Number(process.env.PREVIEW_PORT || 4183);
const browserCandidates = process.platform === "win32"
  ? [
      process.env.BROWSER_EXECUTABLE_PATH,
      process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
      "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe",
      "C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe",
      "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
      "C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
    ]
  : [
      process.env.BROWSER_EXECUTABLE_PATH,
      process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
      "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
      "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
      "/usr/bin/microsoft-edge",
      "/usr/bin/google-chrome",
      "/usr/bin/chromium",
      "/usr/bin/chromium-browser",
    ];

const budgets = [];

function addBudget(name, passed, actual, expected, details = "") {
  budgets.push({ name, passed, actual, expected, details });
}

function base64Url(bytes) {
  return Buffer.from(bytes).toString("base64url");
}

function toNumber(value) {
  return typeof value === "number" && Number.isFinite(value) ? value : null;
}

function isIgnoredRequest(url) {
  return /favicon\.ico($|\?)/.test(url) || url.startsWith("chrome-extension://");
}

async function findFreePort(startPort) {
  for (let port = startPort; port < startPort + 20; port += 1) {
    // eslint-disable-next-line no-await-in-loop
    const free = await isPortFree(port);
    if (free) {
      return port;
    }
  }

  throw new Error(`Could not find a free port starting at ${startPort}.`);
}

function isPortFree(port) {
  return new Promise((resolve) => {
    const server = net.createServer();

    server.unref();
    server.once("error", () => resolve(false));
    server.once("listening", () => {
      server.close(() => resolve(true));
    });
    server.listen(port, previewHost);
  });
}

async function waitForServer(url, timeoutMs = 20_000) {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    try {
      const response = await fetch(url, { method: "GET" });
      if (response.ok) {
        return;
      }
    } catch {
      // retry
    }

    // eslint-disable-next-line no-await-in-loop
    await new Promise((resolve) => setTimeout(resolve, 250));
  }

  throw new Error(`Preview server did not become ready at ${url} within ${timeoutMs}ms.`);
}

function findEntryKey(manifest) {
  if (manifest["index.html"]?.isEntry) {
    return "index.html";
  }

  for (const [key, entry] of Object.entries(manifest)) {
    if (entry?.isEntry && entry?.src?.endsWith("src/main.tsx")) {
      return key;
    }
  }

  for (const [key, entry] of Object.entries(manifest)) {
    if (entry?.isEntry && entry?.src?.endsWith("main.tsx")) {
      return key;
    }
  }

  throw new Error("Could not locate the main entry in the Vite manifest.");
}

function collectInitialAssets(manifest, entryKey) {
  const visited = new Set();
  const js = new Set();
  const css = new Set();

  const visit = (key) => {
    if (visited.has(key)) {
      return;
    }

    visited.add(key);
    const entry = manifest[key];
    if (!entry) {
      return;
    }

    if (entry.file && entry.file.endsWith(".js")) {
      js.add(entry.file);
    }

    for (const cssFile of entry.css ?? []) {
      css.add(cssFile);
    }

    for (const importKey of entry.imports ?? []) {
      visit(importKey);
    }
  };

  visit(entryKey);

  for (const dynamicImportKey of manifest[entryKey]?.dynamicImports ?? []) {
    if (typeof dynamicImportKey === "string" && !dynamicImportKey.startsWith("src/pages/")) {
      visit(dynamicImportKey);
    }
  }

  return {
    js: [...js],
    css: [...css],
  };
}

function collectAllAssetFiles(manifest) {
  const files = new Set();

  for (const entry of Object.values(manifest)) {
    if (entry?.file) {
      files.add(entry.file);
    }

    for (const cssFile of entry?.css ?? []) {
      files.add(cssFile);
    }
  }

  return [...files];
}

async function gzipSize(filePath) {
  const bytes = await readFile(filePath);
  return gzipSync(bytes).byteLength;
}

async function buildAssetStats() {
  if (!existsSync(manifestPath)) {
    throw new Error(`Missing Vite manifest at ${manifestPath}. Run the build first.`);
  }

  const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
  const entryKey = findEntryKey(manifest);
  const initialAssets = collectInitialAssets(manifest, entryKey);
  const allAssetFiles = collectAllAssetFiles(manifest);

  const jsAssets = await Promise.all(
    initialAssets.js.map(async (file) => ({
      file,
      path: path.join(distDir, file),
      gzipBytes: await gzipSize(path.join(distDir, file)),
    })),
  );

  const cssAssets = await Promise.all(
    initialAssets.css.map(async (file) => ({
      file,
      path: path.join(distDir, file),
      gzipBytes: await gzipSize(path.join(distDir, file)),
    })),
  );

  return {
    entryKey,
    jsAssets,
    cssAssets,
    homeJsGzipBytes: jsAssets.reduce((sum, asset) => sum + asset.gzipBytes, 0),
    homeCssGzipBytes: cssAssets.reduce((sum, asset) => sum + asset.gzipBytes, 0),
    allAssetFiles,
  };
}

async function startPreviewServer(port) {
  const child = spawn(
    npmCommand,
    [
      "run",
      "preview",
      "--",
      "--host",
      previewHost,
      "--port",
      String(port),
      "--strictPort",
    ],
    {
      cwd: appDir,
      stdio: ["ignore", "pipe", "pipe"],
      shell: process.platform === "win32",
      windowsHide: true,
    },
  );

  let output = "";
  child.stdout?.on("data", (chunk) => {
    output += chunk.toString();
  });
  child.stderr?.on("data", (chunk) => {
    output += chunk.toString();
  });

  try {
    await waitForServer(`http://${previewHost}:${port}`);
  } catch (error) {
    child.kill();
    throw new Error(`${String(error)}\n${output}`);
  }

  return child;
}

async function launchBrowser() {
  const executablePath = browserCandidates.find((candidate) => candidate && existsSync(candidate));
  if (!executablePath) {
    throw new Error(
      "Could not find a browser executable. Set BROWSER_EXECUTABLE_PATH or install Edge/Chrome.",
    );
  }

  return chromium.launch({
    executablePath,
    headless: true,
  });
}

async function warmPreviewAssets(baseUrl, assetFiles) {
  const warmTargets = [baseUrl, ...assetFiles.map((file) => `${baseUrl}/${file}`)];
  for (const target of warmTargets) {
    // eslint-disable-next-line no-await-in-loop
    await fetch(target).catch(() => null);
  }
}

function makeRouteResponse(body, status = 200) {
  return {
    status,
    headers: corsHeaders({
      "Content-Type": "application/json",
    }),
    body: JSON.stringify(body),
  };
}

function makePreflightResponse() {
  return {
    status: 204,
    headers: corsHeaders(),
    body: "",
  };
}

function corsHeaders(extra = {}) {
  return {
    "Access-Control-Allow-Origin": "*",
    "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
    "Access-Control-Allow-Headers": "Content-Type, Accept, X-Request-ID",
    "Access-Control-Max-Age": "600",
    ...extra,
  };
}

async function createEncryptedSecret(plaintext) {
  const keyBytes = webcrypto.getRandomValues(new Uint8Array(32));
  const nonce = webcrypto.getRandomValues(new Uint8Array(12));
  const key = await webcrypto.subtle.importKey(
    "raw",
    keyBytes,
    {
      name: "AES-GCM",
      length: 256,
    },
    true,
    ["encrypt", "decrypt"],
  );

  const ciphertextBuffer = await webcrypto.subtle.encrypt(
    {
      name: "AES-GCM",
      iv: nonce,
    },
    key,
    new TextEncoder().encode(plaintext),
  );

  return {
    fragmentKey: base64Url(keyBytes),
    ciphertext: base64Url(new Uint8Array(ciphertextBuffer)),
    nonce: base64Url(nonce),
  };
}

function performanceSnapshot(entries) {
  return {
    domContentLoadedMs: toNumber(entries.navigation?.domContentLoadedEventEnd),
    loadMs: toNumber(entries.navigation?.loadEventEnd),
    fcpMs: toNumber(entries.fcp?.startTime),
  };
}

async function readPerformance(page) {
  return page.evaluate(() => {
    const navigation = performance.getEntriesByType("navigation")[0];
    const fcp =
      performance.getEntriesByName("first-contentful-paint")[0] ||
      performance.getEntriesByType("paint").find((entry) => entry.name === "first-contentful-paint");

    return {
      navigation: navigation
        ? {
            domContentLoadedEventEnd: navigation.domContentLoadedEventEnd,
            loadEventEnd: navigation.loadEventEnd,
          }
        : null,
      fcp: fcp
        ? {
            startTime: fcp.startTime,
          }
        : null,
    };
  });
}

async function createPage(browser, options = {}) {
  const context = await browser.newContext({
    viewport: options.viewport ?? { width: 1440, height: 900 },
  });

  const page = await context.newPage();
  const requests = [];
  const consoleMessages = [];
  const pageErrors = [];

  page.on("request", (request) => {
    if (isIgnoredRequest(request.url())) {
      return;
    }

    requests.push({
      url: request.url(),
      resourceType: request.resourceType(),
      method: request.method(),
    });
  });

  page.on("console", (message) => {
    consoleMessages.push({
      type: message.type(),
      text: message.text(),
    });
  });

  page.on("pageerror", (error) => {
    pageErrors.push(error.message);
  });

  return {
    context,
    page,
    requests,
    consoleMessages,
    pageErrors,
  };
}

async function measureHome(browser, baseUrl, viewport, name) {
  const { context, page, requests } = await createPage(browser, {
    viewport,
    isMobile: viewport.width < 700,
    hasTouch: viewport.width < 700,
  });

  try {
    const startedAt = Date.now();
    await page.goto(baseUrl, { waitUntil: "domcontentloaded" });
    await page.waitForLoadState("networkidle");
    await page.waitForFunction(() => {
      return performance.getEntriesByName("first-contentful-paint").length > 0;
    });

    const perf = performanceSnapshot(await readPerformance(page));
    const requestCount = requests.length;
    const scriptRequests = requests.filter((request) => request.resourceType === "script").length;
    const assetTypes = requests.reduce((acc, request) => {
      acc[request.resourceType] = (acc[request.resourceType] || 0) + 1;
      return acc;
    }, {});

    return {
      name,
      requestCount,
      scriptRequests,
      assetTypes,
      elapsedMs: Date.now() - startedAt,
      ...perf,
    };
  } finally {
    await context.close();
  }
}

async function measureQrLazyLoad(browser, baseUrl) {
  const { context, page, requests } = await createPage(browser, {
    viewport: { width: 1440, height: 900 },
  });

  try {
    await page.route("**/api/secrets", async (route) => {
      if (route.request().method() === "OPTIONS") {
        await route.fulfill(makePreflightResponse());
        return;
      }

      await route.fulfill(
        makeRouteResponse({
          secretId: "qr-demo-secret",
          token: "qr-demo-secret",
          expiresAt: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
        }),
      );
    });

    await page.goto(baseUrl, { waitUntil: "domcontentloaded" });
    await page.waitForSelector("textarea#plaintext");

    const payload = "Tin nhan kiem thu QR";
    await page.fill("textarea#plaintext", payload);
    await page.getByRole("button", { name: "Tạo link bí mật" }).click();
    await page.waitForSelector(".parchment-input");

    const scriptsBefore = requests
      .filter((request) => request.resourceType === "script")
      .map((request) => request.url);

    await page.getByTitle("Mã QR").click();
    await page.waitForSelector(".otl-qrcode-canvas-wrapper canvas");
    await page.waitForLoadState("networkidle");

    const scriptsAfter = requests
      .filter((request) => request.resourceType === "script")
      .map((request) => request.url);

    const newScriptRequests = scriptsAfter.filter((url) => !scriptsBefore.includes(url));

    return {
      beforeScriptRequests: scriptsBefore.length,
      afterScriptRequests: scriptsAfter.length,
      newScriptRequests,
      requestCount: requests.length,
    };
  } finally {
    await context.close();
  }
}

async function measureRevealScenario(browser, baseUrl, scenario) {
  const { context, page, requests, consoleMessages, pageErrors } = await createPage(browser, {
    viewport: { width: 1440, height: 900 },
  });

  const plaintext = `Tin kiem thu ${scenario.name}`;
  const secret = await createEncryptedSecret(plaintext);
  const secretUrl = `${baseUrl}/s/demo-secret#${secret.fragmentKey}`;
  const routeState = {
    statusRequestStartedAt: null,
    openRequestStartedAt: null,
  };

  try {
    if (scenario.blockFonts) {
      await page.route("**/fonts.googleapis.com/**", (route) => route.abort());
      await page.route("**/fonts.gstatic.com/**", (route) => route.abort());
    }

    await page.route("**/api/secrets/demo-secret/status", async (route) => {
      if (route.request().method() === "OPTIONS") {
        await route.fulfill(makePreflightResponse());
        return;
      }

      routeState.statusRequestStartedAt = Date.now();

      if (scenario.statusMode === "offline") {
        await route.abort("failed");
        return;
      }

      if (scenario.statusDelayMs) {
        await new Promise((resolve) => setTimeout(resolve, scenario.statusDelayMs));
      }

      if (scenario.statusMode === "server-error") {
        await route.fulfill(
          makeRouteResponse(
            {
              message: "Status service unavailable",
            },
            500,
          ),
        );
        return;
      }

      await route.fulfill(
        makeRouteResponse({
          secretId: "demo-secret",
          status: "active",
          expiresAt: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
        }),
      );
    });

    await page.route("**/api/secrets/demo-secret/open", async (route) => {
      if (route.request().method() === "OPTIONS") {
        await route.fulfill(makePreflightResponse());
        return;
      }

      routeState.openRequestStartedAt = Date.now();

      if (scenario.openMode === "offline") {
        await route.abort("failed");
        return;
      }

      if (scenario.openDelayMs) {
        await new Promise((resolve) => setTimeout(resolve, scenario.openDelayMs));
      }

      if (scenario.openMode === "server-error") {
        await route.fulfill(
          makeRouteResponse(
            {
              message: "Open service unavailable",
            },
            500,
          ),
        );
        return;
      }

      await route.fulfill(
        makeRouteResponse({
          secretId: "demo-secret",
          ciphertext: secret.ciphertext,
          nonce: secret.nonce,
          algorithm: "AES-GCM",
          consumedAt: new Date().toISOString(),
        }),
      );
    });

    const startedAt = Date.now();
    await page.goto(secretUrl, { waitUntil: "domcontentloaded" });

    if (scenario.statusMode === "offline") {
      await page.waitForSelector(".secret-status-error", { timeout: 15_000 });
      return {
        requestCount: requests.length,
        statusError: true,
        errorText: await page.locator(".status-body-text").textContent(),
        statusRequestStartedAt: routeState.statusRequestStartedAt,
        openRequestStartedAt: routeState.openRequestStartedAt,
        elapsedMs: Date.now() - startedAt,
      };
    }

    const sealedEnvelope = page.locator(".secret-envelope-sealed");
    try {
      await sealedEnvelope.waitFor({ state: "attached", timeout: 15_000 });
    } catch (error) {
      const bodyText = await page.locator("body").innerText().catch(() => "");
      throw new Error(
        `${String(error)}\n\nReveal body snapshot:\n${bodyText}\n\nURL: ${page.url()}\n\nPage errors:\n${pageErrors.join("\n")}\n\nConsole:\n${consoleMessages.map((entry) => `${entry.type}: ${entry.text}`).join("\n")}\n\nRequests:\n${requests.map((request) => `${request.method} ${request.url}`).join("\n")}`,
      );
    }
    await page.waitForFunction(() => {
      const element = document.querySelector(".secret-envelope-sealed");
      return Boolean(element && element.getBoundingClientRect().width > 0 && element.getBoundingClientRect().height > 0);
    }, null, { timeout: 15_000 });
    const sealedVisibleAt = Date.now();

    await page.getByRole("button", { name: "Tôi hiểu rồi, tiếp tục tới phong bì" }).click();
    const envelope = page.locator(".secret-envelope-sealed");
    const box = await envelope.boundingBox();
    if (!box) {
      throw new Error("Could not locate the sealed envelope box for the hold interaction.");
    }

    const endStatePromise = scenario.openMode === "server-error"
      ? page.waitForSelector(".secret-status-error", { timeout: 15_000 })
      : page.waitForSelector(".secret-envelope-revealed", { timeout: 15_000 });
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    await page.mouse.down();
    await endStatePromise;
    await page.mouse.up();
    const revealedVisibleAt = Date.now();

    if (scenario.openMode === "server-error") {
      return {
        requestCount: requests.length,
        statusRequestStartedAt: routeState.statusRequestStartedAt,
        openRequestStartedAt: routeState.openRequestStartedAt,
        openToErrorMs: routeState.openRequestStartedAt
          ? revealedVisibleAt - routeState.openRequestStartedAt
          : null,
        statusToSealedMs: sealedVisibleAt - routeState.statusRequestStartedAt,
        statusError: true,
        errorText: await page.locator(".status-body-text").textContent(),
        elapsedMs: Date.now() - startedAt,
        scriptRequests: requests.filter((request) => request.resourceType === "script").length,
      };
    }

    await page.waitForFunction(
      (expectedText) => {
        const text = document.querySelector(".secret-paper pre")?.textContent;
        return Boolean(text && text.includes(expectedText));
      },
      plaintext,
      { timeout: 15_000 },
    );

    const preText = await page.locator(".secret-paper pre").textContent();

    return {
      requestCount: requests.length,
      statusRequestStartedAt: routeState.statusRequestStartedAt,
      openRequestStartedAt: routeState.openRequestStartedAt,
      sealedVisibleMs: sealedVisibleAt - startedAt,
      revealedVisibleMs: revealedVisibleAt - startedAt,
      openToRevealedMs: routeState.openRequestStartedAt
        ? revealedVisibleAt - routeState.openRequestStartedAt
        : null,
      statusToSealedMs: routeState.statusRequestStartedAt
        ? sealedVisibleAt - routeState.statusRequestStartedAt
        : null,
      preText,
      elapsedMs: Date.now() - startedAt,
      scriptRequests: requests.filter((request) => request.resourceType === "script").length,
    };
  } finally {
    await context.close();
  }
}

async function main() {
  await mkdir(reportDir, { recursive: true });

  const assetStats = await buildAssetStats();
  const port = await findFreePort(previewStartPort);
  const previewProcess = await startPreviewServer(port);
  const baseUrl = `http://${previewHost}:${port}`;
  const report = {
    generatedAt: new Date().toISOString(),
    baseUrl,
    assets: assetStats,
    browser: {},
    budgets,
  };

  try {
    const browser = await launchBrowser();
    const homeDesktop = await measureHome(browser, baseUrl, { width: 1440, height: 900 }, "desktop");
    const homeMobile = await measureHome(browser, baseUrl, { width: 390, height: 844 }, "mobile");
    const qrLazyLoad = await measureQrLazyLoad(browser, baseUrl);
    await browser.close();

    const revealBrowser = await launchBrowser();
    const revealSlow = await measureRevealScenario(revealBrowser, baseUrl, {
      name: "slow",
      statusDelayMs: 1200,
      openDelayMs: 700,
      statusMode: "active",
      openMode: "active",
    });
    const revealOffline = await measureRevealScenario(revealBrowser, baseUrl, {
      name: "offline",
      statusMode: "offline",
      openMode: "offline",
    });
    const revealServerError = await measureRevealScenario(revealBrowser, baseUrl, {
      name: "server-error",
      statusDelayMs: 150,
      openDelayMs: 150,
      statusMode: "active",
      openMode: "server-error",
    });
    const revealFontBlocked = await measureRevealScenario(revealBrowser, baseUrl, {
      name: "font-blocked",
      statusDelayMs: 150,
      openDelayMs: 120,
      statusMode: "active",
      openMode: "active",
      blockFonts: true,
    });
    await revealBrowser.close();

    report.browser = {
      homeDesktop,
      homeMobile,
      qrLazyLoad,
      revealSlow,
      revealOffline,
      revealServerError,
      revealFontBlocked,
    };

    addBudget(
      "home-js-gzip",
      assetStats.homeJsGzipBytes <= 100 * 1024,
      assetStats.homeJsGzipBytes,
      "<= 100 KiB",
      "Initial home JS bundle should stay below the target budget.",
    );

    addBudget(
      "home-fcp-desktop",
      homeDesktop.fcpMs !== null && homeDesktop.fcpMs <= 1000,
      homeDesktop.fcpMs,
      "<= 1000 ms",
      "Desktop home FCP should land within the target.",
    );

    addBudget(
      "home-fcp-mobile",
      homeMobile.fcpMs !== null && homeMobile.fcpMs <= 1800,
      homeMobile.fcpMs,
      "<= 1800 ms",
      "Mobile home FCP should stay inside the mobile target.",
    );

    addBudget(
      "home-request-count",
      homeDesktop.requestCount <= 4,
      homeDesktop.requestCount,
      "<= 4 requests",
      "Home should avoid pulling reveal assets or remote fonts on first paint.",
    );

    addBudget(
      "qr-lazy-script-load",
      qrLazyLoad.newScriptRequests.length >= 1,
      qrLazyLoad.newScriptRequests.length,
      ">= 1 new script request",
      "Opening the QR modal should lazily fetch the QR code module.",
    );

    addBudget(
      "reveal-slow-status-to-sealed",
      revealSlow.statusToSealedMs !== null && revealSlow.statusToSealedMs <= 1700,
      revealSlow.statusToSealedMs,
      "<= 1700 ms after status request",
      "Sealed state should appear shortly after the status response starts.",
    );

    addBudget(
      "reveal-slow-open-to-revealed",
      revealSlow.openToRevealedMs !== null && revealSlow.openToRevealedMs <= 1700,
      revealSlow.openToRevealedMs,
      "<= 1700 ms after open request",
      "Reveal should complete quickly after the open request begins.",
    );

    addBudget(
      "reveal-offline-error",
      revealOffline.statusError === true,
      revealOffline.errorText,
      "error state",
      "Offline status requests should surface an error screen, not a spinner.",
    );

    addBudget(
      "reveal-server-error",
      revealServerError.statusError === true &&
        revealServerError.openToErrorMs !== null &&
        revealServerError.openToErrorMs <= 1150,
      revealServerError.openToErrorMs,
      "<= 1150 ms to error after open request",
      "Server-side open failures should still be surfaced without hanging.",
    );

    addBudget(
      "reveal-font-blocked",
      revealFontBlocked.preText?.includes("Tin kiem thu font-blocked") ?? false,
      revealFontBlocked.preText,
      "revealed text present",
      "Blocking font CDNs should not block the reveal flow.",
    );

    const csvRows = [
      ["section", "metric", "value", "unit", "details"],
      ["assets", "home_js_gzip", assetStats.homeJsGzipBytes, "bytes", assetStats.jsAssets.map((asset) => asset.file).join("; ")],
      ["assets", "home_css_gzip", assetStats.homeCssGzipBytes, "bytes", assetStats.cssAssets.map((asset) => asset.file).join("; ")],
      ["home_desktop", "dom_content_loaded", homeDesktop.domContentLoadedMs, "ms", ""],
      ["home_desktop", "first_contentful_paint", homeDesktop.fcpMs, "ms", ""],
      ["home_desktop", "requests", homeDesktop.requestCount, "count", JSON.stringify(homeDesktop.assetTypes)],
      ["home_mobile", "dom_content_loaded", homeMobile.domContentLoadedMs, "ms", ""],
      ["home_mobile", "first_contentful_paint", homeMobile.fcpMs, "ms", ""],
      ["home_mobile", "requests", homeMobile.requestCount, "count", JSON.stringify(homeMobile.assetTypes)],
      ["qr_lazy", "new_script_requests", qrLazyLoad.newScriptRequests.length, "count", qrLazyLoad.newScriptRequests.join("; ")],
      ["reveal_slow", "status_to_sealed", revealSlow.statusToSealedMs, "ms", ""],
      ["reveal_slow", "open_to_revealed", revealSlow.openToRevealedMs, "ms", ""],
      ["reveal_slow", "requests", revealSlow.requestCount, "count", ""],
      ["reveal_offline", "error", revealOffline.statusError ? 1 : 0, "boolean", revealOffline.errorText ?? ""],
      ["reveal_server_error", "open_to_error", revealServerError.openToErrorMs, "ms", revealServerError.errorText ?? ""],
      ["reveal_font_blocked", "revealed", revealFontBlocked.preText ? 1 : 0, "boolean", ""],
    ];

    await writeFile(reportJsonPath, `${JSON.stringify(report, null, 2)}\n`, "utf8");
    await writeFile(
      reportCsvPath,
      `${csvRows.map((row) => row.map((cell) => JSON.stringify(cell ?? "")).join(",")).join("\n")}\n`,
      "utf8",
    );

    const failedBudgets = budgets.filter((budget) => !budget.passed);
    const summaryLines = [
      `Preview: ${baseUrl}`,
      `Home JS gzip: ${assetStats.homeJsGzipBytes} bytes`,
      `Home FCP desktop: ${homeDesktop.fcpMs ?? "n/a"} ms`,
      `Home FCP mobile: ${homeMobile.fcpMs ?? "n/a"} ms`,
      `QR lazy-load new scripts: ${qrLazyLoad.newScriptRequests.length}`,
      `Reveal open-to-revealed: ${revealSlow.openToRevealedMs ?? "n/a"} ms`,
      `Budgets passed: ${budgets.length - failedBudgets.length}/${budgets.length}`,
      `Report JSON: ${reportJsonPath}`,
      `Report CSV: ${reportCsvPath}`,
    ];

    console.log(summaryLines.join("\n"));

    if (failedBudgets.length > 0) {
      console.error("\nBudget failures:");
      for (const failure of failedBudgets) {
        console.error(`- ${failure.name}: ${failure.actual} (expected ${failure.expected})`);
      }
      process.exitCode = 1;
    }
  } finally {
    previewProcess.kill();
  }
}

await main();
