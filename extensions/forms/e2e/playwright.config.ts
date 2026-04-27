import { defineConfig, devices } from "@playwright/test";

/**
 * Forms extension E2E test configuration.
 *
 * Dev server: `make dev` from the project root (starts on :3000 by default).
 * Run:        npx playwright test --config=extensions/forms/playwright.config.ts
 */
export default defineConfig({
  testDir: ".",
  timeout: 30_000,
  retries: 0,
  reporter: [["list"]],

  use: {
    baseURL: process.env.BASE_URL ?? "http://localhost:3000",
    headless: true,
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
  },

  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],

  // Uncomment to auto-start the dev server when running E2E locally:
  // webServer: {
  //   command: "make dev",
  //   cwd: "../../..",
  //   port: 3000,
  //   reuseExistingServer: true,
  //   timeout: 60_000,
  // },
});
