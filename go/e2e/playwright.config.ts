import { defineConfig, devices } from "@playwright/test";

const baseURL = process.env.E2E_BASE_URL || "http://localhost:4201";

export default defineConfig({
  testDir: "./tests",
  globalTeardown: "./global-teardown.ts",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  // CIに限らず常に直列 (ワーカー1) で実行する。開発コンテナはCPUが制約され、
  // 並列ワーカーだとeditor系スペックがCodeMirrorの非同期初期化 (アップロード
  // ハンドラはビュー生成後に登録される等) とレースし、負荷依存のフレークが出る。
  // 直列実行は決定的で、スイートが小さいため遅延もわずか (~9.7s vs ~4.9s)。
  workers: 1,
  reporter: process.env.CI ? "github" : "list",
  timeout: 30_000,
  expect: {
    timeout: 5_000,
  },
  use: {
    baseURL,
    trace: "on-first-retry",
    locale: "ja",
  },
  projects: [
    {
      name: "setup",
      testMatch: /.*\.setup\.ts/,
    },
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
        storageState: "playwright/.auth/user.json",
      },
      dependencies: ["setup"],
    },
  ],
});
