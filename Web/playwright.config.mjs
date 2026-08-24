import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/e2e",
  use: {
    ...devices["Desktop Chrome"],
    baseURL: "http://127.0.0.1:4321/street-fighter-2-theme/"
  },
  webServer: {
    command: "npm run build && npm run preview -- --host 127.0.0.1",
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
    url: "http://127.0.0.1:4321/street-fighter-2-theme/"
  }
});
