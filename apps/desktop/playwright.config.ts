import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  webServer: {
    command: "npm run dev",
    port: 1420,
    reuseExistingServer: true,
  },
  projects: [
    {
      name: "web",
      testIgnore: /electron-smoke\.spec\.ts/,
      use: {
        baseURL: "http://127.0.0.1:1420",
        trace: "on-first-retry",
      },
    },
    {
      name: "electron",
      testMatch: /electron-smoke\.spec\.ts/,
      use: {
        trace: "on-first-retry",
      },
    },
  ],
});
