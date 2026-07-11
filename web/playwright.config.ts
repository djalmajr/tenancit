import { defineConfig, devices } from "@playwright/test";

process.env.PLAYWRIGHT_NO_COPY_PROMPT ??= "1";

const baseURL = process.env.TENANCIT_E2E_BASE_URL ?? "http://127.0.0.1:18080";
const retries = Number(process.env.TENANCIT_E2E_RETRIES ?? (process.env.CI ? 1 : 0));
const desktopChrome = { ...devices["Desktop Chrome"] };
const captureFailureScreenshots = process.env.TENANCIT_E2E_EPHEMERAL === "true";
const outputDir = process.env.TENANCIT_E2E_OUTPUT_DIR ?? `../output/playwright/direct-${process.pid}`;
const chromiumIgnoredTests = [
  /bootstrap\.e2e\.test\.ts/,
  ...(process.env.TENANCIT_E2E_AUTH_MODE === "oidc" ? [] : [/oidc-auth\.e2e\.test\.ts/]),
];

export default defineConfig({
  testDir: "./e2e",
  testMatch: /.*\.e2e\.test\.ts/,
  fullyParallel: false,
  workers: 1,
  retries,
  timeout: 90_000,
  expect: { timeout: 5_000 },
  outputDir,
  reporter: "line",
  use: {
    actionTimeout: 10_000,
    baseURL,
    navigationTimeout: 15_000,
    screenshot: captureFailureScreenshots ? "only-on-failure" : "off",
    // Playwright traces persist locator values and Authorization headers.
    // They must stay disabled even for the disposable internal stack.
    trace: "off",
    video: "off",
  },
  projects: [
    {
      name: "bootstrap",
      testMatch: /bootstrap\.e2e\.test\.ts/,
      use: desktopChrome,
    },
    {
      name: "chromium",
      dependencies: ["bootstrap"],
      testIgnore: chromiumIgnoredTests,
      use: desktopChrome,
    },
  ],
});
