import { existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import net from "node:net";
import path from "node:path";
import { spawn } from "node:child_process";
import { chromium } from "playwright-core";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const appDir = path.resolve(scriptDir, "..");
const repoRoot = path.resolve(appDir, "..", "..");
const npmCommand = process.platform === "win32" ? "npm.cmd" : "npm";
const previewHost = "127.0.0.1";
const previewStartPort = Number(process.env.E2E_PREVIEW_PORT || 4183);
const backendStartPort = Number(process.env.E2E_BACKEND_PORT || 18080);
const redisAddr = process.env.E2E_REDIS_ADDR || "127.0.0.1:6379";

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

function log(message) {
  process.stdout.write(`${message}\n`);
}

function spawnCommand(command, args, options = {}) {
  const finalCommand = process.platform === "win32" && command.toLowerCase().endsWith(".cmd")
    ? "cmd.exe"
    : command;
  const finalArgs = process.platform === "win32" && command.toLowerCase().endsWith(".cmd")
    ? ["/d", "/s", "/c", command, ...args]
    : args;
  const child = spawn(finalCommand, finalArgs, {
    cwd: options.cwd ?? appDir,
    env: options.env ?? process.env,
    stdio: ["ignore", "pipe", "pipe"],
    windowsHide: true,
  });

  let output = "";
  child.stdout?.on("data", (chunk) => {
    output += chunk.toString();
  });
  child.stderr?.on("data", (chunk) => {
    output += chunk.toString();
  });

  return { child, outputRef: () => output };
}

function stopCommand(child) {
  if (!child || child.killed) {
    return;
  }

  if (process.platform === "win32" && child.pid) {
    const killer = spawn("taskkill", ["/pid", String(child.pid), "/T", "/F"], {
      cwd: appDir,
      stdio: "ignore",
      windowsHide: true,
    });
    killer.unref();
  } else {
    child.kill("SIGKILL");
  }

  child.stdout?.destroy();
  child.stderr?.destroy();
}

function runCommand(command, args, options = {}) {
  return new Promise((resolve, reject) => {
    const { child, outputRef } = spawnCommand(command, args, options);

    child.once("error", (error) => {
      reject(new Error(`${command} failed to start: ${error.message}`));
    });

    child.once("close", (code) => {
      if (code === 0) {
        resolve(outputRef());
        return;
      }

      reject(new Error(`${command} ${args.join(" ")} exited with code ${code}\n${outputRef()}`));
    });
  });
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

async function findFreePort(startPort) {
  for (let port = startPort; port < startPort + 20; port += 1) {
    // eslint-disable-next-line no-await-in-loop
    if (await isPortFree(port)) {
      return port;
    }
  }

  throw new Error(`Could not find a free port starting at ${startPort}.`);
}

async function waitForServer(url, timeoutMs = 30_000) {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    try {
      const response = await fetch(url, { method: "GET" });
      if (response.ok) {
        return;
      }
    } catch {
      // Retry until the service becomes ready.
    }

    // eslint-disable-next-line no-await-in-loop
    await new Promise((resolve) => setTimeout(resolve, 250));
  }

  throw new Error(`Server did not become ready at ${url} within ${timeoutMs}ms.`);
}

function resolveBrowserExecutablePath() {
  for (const candidate of browserCandidates) {
    if (candidate && existsSync(candidate)) {
      return candidate;
    }
  }

  return undefined;
}

async function startBackend(port, allowedOrigin) {
  const { child, outputRef } = spawnCommand(
    "go",
    ["run", "./backend/cmd/api"],
    {
      cwd: repoRoot,
      env: {
        ...process.env,
        APP_ENV: "test",
        APP_HOST: previewHost,
        APP_PORT: String(port),
        ALLOWED_ORIGIN: allowedOrigin,
        REDIS_ADDR: redisAddr,
        RATE_LIMIT_ENABLED: "false",
      },
    },
  );

  try {
    await waitForServer(`http://${previewHost}:${port}/healthz`);
  } catch (error) {
    stopCommand(child);
    throw new Error(`${String(error)}\n${outputRef()}`);
  }

  return { child, outputRef };
}

async function startPreview(port, backendBaseUrl) {
  const { child, outputRef } = spawnCommand(
    npmCommand,
    ["run", "preview", "--", "--host", previewHost, "--port", String(port), "--strictPort"],
    {
      cwd: appDir,
      env: {
        ...process.env,
        VITE_API_BASE_URL: backendBaseUrl,
      },
    },
  );

  try {
    await waitForServer(`http://${previewHost}:${port}`);
  } catch (error) {
    stopCommand(child);
    throw new Error(`${String(error)}\n${outputRef()}`);
  }

  return { child, outputRef };
}

async function buildFrontend(backendBaseUrl) {
  log("Building frontend for e2e...");
  await runCommand(
    npmCommand,
    ["run", "build"],
    {
      cwd: appDir,
      env: {
        ...process.env,
        VITE_API_BASE_URL: backendBaseUrl,
      },
    },
  );
}

async function run() {
  const backendPort = await findFreePort(backendStartPort);
  const previewPort = await findFreePort(previewStartPort);
  const backendBaseUrl = `http://${previewHost}:${backendPort}`;
  const previewBaseUrl = `http://${previewHost}:${previewPort}`;

  let backendProcess;
  let previewProcess;
  let browser;

  try {
    await buildFrontend(backendBaseUrl);
    backendProcess = await startBackend(backendPort, previewBaseUrl);
    previewProcess = await startPreview(previewPort, backendBaseUrl);

    const launchOptions = { headless: true };
    const browserExecutable = resolveBrowserExecutablePath();
    if (browserExecutable) {
      launchOptions.executablePath = browserExecutable;
    }

    try {
      browser = await chromium.launch(launchOptions);
    } catch (error) {
      throw new Error(
        `${String(error)}\nUnable to launch a browser for the e2e flow. ` +
        "Install Chromium with `npx --yes playwright@1.60.0 install chromium` or set BROWSER_EXECUTABLE_PATH.",
      );
    }
    const context = await browser.newContext({ baseURL: previewBaseUrl });
    const page = await context.newPage();

    const secretText = `E2E secret ${Date.now()}`;

    log("Opening create page...");
    await page.goto("/");
    await page.getByLabel("Nội dung bí mật").fill(secretText);
    await page.getByRole("button", { name: "Tạo link bí mật" }).click();

    const secretLinkInput = page.getByLabel("Secret link");
    await secretLinkInput.waitFor({ state: "visible" });
    const secretLink = await secretLinkInput.inputValue();
    if (!secretLink.includes("#")) {
      throw new Error(`Generated secret link is missing a fragment: ${secretLink}`);
    }

    log("Opening reveal link...");
    await page.goto(secretLink);

    const consentButton = page.getByRole("button", { name: /Tôi hiểu rồi/i });
    await consentButton.click();

    const revealSessionResponsePromise = page.waitForResponse(
      (response) =>
        response.request().method() === "POST" &&
        response.url().includes("/api/reveal-sessions"),
    );

    const openSecretResponsePromise = page.waitForResponse(
      (response) =>
        response.request().method() === "POST" &&
        response.url().includes("/api/secrets/") &&
        response.url().includes("/open"),
    );

    const openEnvelope = page.getByRole("button", { name: "Mở lá thư bí mật" });
    await openEnvelope.waitFor({ state: "visible" });
    await openEnvelope.focus();
    await openEnvelope.press("Enter");

    const revealSessionResponse = await revealSessionResponsePromise;
    if (revealSessionResponse.status() !== 201) {
      throw new Error(`Expected reveal session response 201, got ${revealSessionResponse.status()}`);
    }

    const openSecretResponse = await openSecretResponsePromise;
    if (openSecretResponse.status() !== 200) {
      throw new Error(`Expected open secret response 200, got ${openSecretResponse.status()}`);
    }

    const requestHeaders = openSecretResponse.request().headers();
    if (!requestHeaders["x-reveal-session"]) {
      throw new Error("Expected X-Reveal-Session header to be sent with open request");
    }

    await page.getByText(secretText, { exact: true }).waitFor({ state: "visible" });

    log("Verifying consumed state on a fresh revisit...");
    const consumedPage = await context.newPage();
    try {
      await consumedPage.goto(secretLink);
      await consumedPage.getByText("Liên kết này đã được mở một lần trước đó.").waitFor({ state: "visible" });
    } finally {
      await consumedPage.close().catch(() => {});
    }

    log("E2E flow passed.");
  } finally {
    if (browser) {
      await browser.close().catch(() => {});
    }

    stopCommand(previewProcess?.child);
    stopCommand(backendProcess?.child);
  }
}

run().catch((error) => {
  console.error(error instanceof Error ? error.stack ?? error.message : String(error));
  process.exit(1);
});
