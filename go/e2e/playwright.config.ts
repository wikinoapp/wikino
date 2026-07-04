import { defineConfig, devices } from "@playwright/test";

const baseURL = process.env.E2E_BASE_URL || "http://localhost:4201";

export default defineConfig({
  testDir: "./tests",
  globalTeardown: "./global-teardown.ts",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  // Run serially (one worker) everywhere, not just in CI. The dev container is
  // CPU-constrained, and with multiple parallel workers the editor specs race
  // CodeMirror's asynchronous initialization (e.g. upload handlers attach after
  // the view mounts), producing load-dependent flakes. Serial execution is
  // deterministic and only marginally slower here because the suite is small
  // (~9.7s vs ~4.9s).
  //
  // [Ja] CI に限らず常に直列 (ワーカー 1) で実行する。開発コンテナは CPU が制約され、
  // 並列ワーカーだと editor 系スペックが CodeMirror の非同期初期化 (アップロード
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
