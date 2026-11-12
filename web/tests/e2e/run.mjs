import { spawn, spawnSync } from "node:child_process";

const port = process.env.E2E_PORT ?? "4321";
const baseURL = `http://127.0.0.1:${port}/sf2-themes/`;

const reachable = async () => {
  try {
    return (await fetch(baseURL)).ok;
  } catch {
    return false;
  }
};

let preview;
if (await reachable()) {
  if (process.env.CI) {
    console.error(`${baseURL} is already serving; CI never reuses a preview server`);
    process.exit(1);
  }
} else {
  const build = spawnSync("aube", ["run", "build"], { stdio: "inherit" });
  if (build.status !== 0) {
    process.exit(build.status ?? 1);
  }
  preview = spawn("aube", ["run", "preview", "--", "--host", "127.0.0.1", "--port", port, "--ignore-lock"], {
    detached: true,
    env: { ...process.env, ASTRO_PREVIEW_BACKGROUND: "0" },
    stdio: "inherit",
  });
  const deadline = Date.now() + 120000;
  while (!(await reachable())) {
    if (Date.now() > deadline || preview.exitCode !== null) {
      console.error(`the preview server never answered at ${baseURL}`);
      process.kill(-preview.pid);
      process.exit(1);
    }
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
}

const tests = spawnSync(process.execPath, ["--test", "tests/e2e/*.spec.mjs"], {
  env: { ...process.env, SF2_E2E_BASE_URL: baseURL },
  stdio: "inherit",
});
if (preview) {
  process.kill(-preview.pid);
}
process.exit(tests.status ?? 1);
