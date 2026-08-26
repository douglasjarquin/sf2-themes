import { createRequire } from "node:module";
import { promises as fs } from "node:fs";
import http from "node:http";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { spawn } from "node:child_process";

const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const webRoot = path.join(projectRoot, "web");
const outputDirectory = path.join(webRoot, "public", "screenshots");
const manifestPath = path.join(outputDirectory, "manifest.json");
const astroCli = path.join(webRoot, "node_modules", "astro", "bin", "astro.mjs");
const viewport = { width: 1280, height: 720 };
const captureMoments = ["fight", "hadouken", "counter", "ko"];
const captureDelays = [100, 580, 1060, 1540];
const requireFromWeb = createRequire(path.join(webRoot, "package.json"));
const { chromium } = requireFromWeb("@playwright/test");

const sleep = (milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds));

function run(command, args, options = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      cwd: webRoot,
      env: { ...process.env, ...options.env },
      stdio: ["ignore", "pipe", "pipe"],
    });
    let stdout = "";
    let stderr = "";
    child.stdout.on("data", (chunk) => {
      stdout += chunk;
    });
    child.stderr.on("data", (chunk) => {
      stderr += chunk;
    });
    child.once("error", reject);
    child.once("close", (code, signal) => {
      if (code === 0) {
        resolve({ stdout, stderr });
        return;
      }
      reject(new Error(`${command} ${args.join(" ")} exited with ${signal ?? code}\n${stderr || stdout}`));
    });
  });
}

async function waitForServer(url) {
  const deadline = Date.now() + 15000;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(url);
      if (response.ok) return;
    } catch {
      // The preview is still starting.
    }
    await sleep(100);
  }
  throw new Error(`Capture server did not become ready at ${url}`);
}

async function startStaticServer() {
  const distRoot = path.join(webRoot, "dist");
  const contentTypes = {
    ".css": "text/css; charset=utf-8",
    ".html": "text/html; charset=utf-8",
    ".js": "text/javascript; charset=utf-8",
    ".json": "application/json; charset=utf-8",
    ".png": "image/png",
    ".svg": "image/svg+xml",
    ".woff2": "font/woff2",
  };
  const server = http.createServer(async (request, response) => {
    try {
      const requestUrl = new URL(request.url ?? "/", "http://127.0.0.1");
      if (!requestUrl.pathname.startsWith("/sf2-themes/")) {
        response.writeHead(404).end();
        return;
      }
      let relativePath = decodeURIComponent(requestUrl.pathname.slice("/sf2-themes/".length));
      if (relativePath.length === 0 || relativePath.endsWith("/")) relativePath += "index.html";
      const filePath = path.resolve(distRoot, relativePath);
      if (filePath !== distRoot && !filePath.startsWith(`${distRoot}${path.sep}`)) {
        response.writeHead(400).end();
        return;
      }
      const file = await fs.readFile(filePath);
      response.writeHead(200, {
        "Content-Type": contentTypes[path.extname(filePath)] ?? "application/octet-stream",
      });
      response.end(file);
    } catch {
      response.writeHead(404).end();
    }
  });
  await new Promise((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", resolve);
  });
  const address = server.address();
  if (address === null || typeof address === "string") throw new Error("Unable to start the capture server");
  return { port: address.port, server };
}

async function readExistingImages() {
  try {
    const manifest = JSON.parse(await fs.readFile(manifestPath, "utf8"));
    if (!Array.isArray(manifest.entries)) return;
    for (const entry of manifest.entries) {
      if (typeof entry.image !== "string" || path.basename(entry.image) !== entry.image) continue;
      if (path.extname(entry.image) !== ".png") continue;
      await fs.rm(path.join(outputDirectory, entry.image), { force: true });
    }
  } catch (error) {
    if (error.code !== "ENOENT" && !(error instanceof SyntaxError)) throw error;
  }
}

async function writeManifest(entries) {
  const manifest = {
    viewport,
    generatedAt: new Date().toISOString(),
    source: "ASCII cabinet",
    entries,
  };
  await fs.writeFile(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`, "utf8");
}

function pngDimensions(buffer) {
  if (buffer.length < 24 || buffer.toString("ascii", 1, 4) !== "PNG") return null;
  return { width: buffer.readUInt32BE(16), height: buffer.readUInt32BE(20) };
}

async function main() {
  process.chdir(webRoot);
  const { characterVariants } = await import(pathToFileURL(path.join(webRoot, "src", "data", "theme-data.mjs")));
  const entries = characterVariants.map((fighter, index) => ({
    id: fighter.id,
    name: fighter.character,
    image: `${fighter.id}.png`,
    moment: captureMoments[index],
  }));

  await fs.mkdir(outputDirectory, { recursive: true });
  await readExistingImages();
  await writeManifest(entries);
  await run(process.execPath, [astroCli, "build"]);

  const { port, server } = await startStaticServer();

  let browser;
  try {
    browser = await chromium.launch({ channel: "chrome", headless: true });
    const page = await browser.newPage({ viewport });
    const pageUrl = `http://127.0.0.1:${port}/sf2-themes/`;
    await waitForServer(pageUrl);
    await page.evaluate(() => document.querySelector(".home-page")?.setAttribute("data-capture-mode", "cabinet"));
    await page.evaluate(() => document.documentElement.setAttribute("data-capture-mode", "cabinet"));

    for (const [index, entry] of entries.entries()) {
      await page.goto(pageUrl, { waitUntil: "networkidle" });
      await page.evaluate(() => document.querySelector(".home-page")?.setAttribute("data-capture-mode", "cabinet"));
      await page.evaluate(() => document.documentElement.setAttribute("data-capture-mode", "cabinet"));
      await page.getByRole("button", { name: entry.name.toUpperCase(), exact: true }).click();
      await page.getByRole("button", { name: "INSERT COIN", exact: true }).click();
      await page.waitForTimeout(captureDelays[index]);

      const imagePath = path.join(outputDirectory, entry.image);
      await page.screenshot({ path: imagePath, fullPage: false });
      const dimensions = pngDimensions(await fs.readFile(imagePath));
      if (dimensions?.width !== viewport.width || dimensions.height !== viewport.height) {
        throw new Error(`${entry.image} has unexpected dimensions ${JSON.stringify(dimensions)}`);
      }
    }
  } finally {
    await browser?.close();
    await new Promise((resolve) => server.close(resolve));
  }

  await writeManifest(entries);
  await run(process.execPath, [astroCli, "build"]);
  console.log(`Captured ${entries.length} cabinet screenshots at ${outputDirectory}`);
}

main().catch((error) => {
  console.error(error instanceof Error ? error.stack : error);
  process.exitCode = 1;
});
