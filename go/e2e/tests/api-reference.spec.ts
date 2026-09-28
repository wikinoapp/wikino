import { expect, test } from "@playwright/test";

test("APIリファレンスがOpenAPI記述を読み込んで操作を描画すること", async ({ page }) => {
  const specResponse = page.waitForResponse((response) => response.url().endsWith("/api/v1/openapi.yaml"));
  await page.goto("/api/reference/v1");

  expect((await specResponse).status()).toBe(200);
  await expect(page.locator("#api-reference")).toContainText("トークンの持ち主のユーザーを取得する");
  await expect(page.locator("#api-reference-fallback")).toHaveCount(0);

  const search = page.getByRole("textbox", { name: "Search" });
  await expect(search).toBeVisible();
  await search.fill("トークンの持ち主のユーザーを取得する");
  await expect(page.locator('[data-role="search:results"]')).toContainText("トークンの持ち主のユーザーを取得する");
});

test("スクリプトを読み込めなくてもOpenAPI記述へ辿れること", async ({ page }) => {
  await page.route("**/static/js/api-reference.js*", (route) => route.abort());
  await page.goto("/api/reference/v1");

  const fallback = page.locator("#api-reference-fallback");
  await expect(fallback.getByRole("heading", { name: "Wikino API V1リファレンス" })).toBeVisible();
  await expect(fallback.getByRole("link", { name: "openapi.yaml" })).toHaveAttribute("href", "/api/v1/openapi.yaml");
});

test("OpenAPI記述を読み込めなくてもOpenAPI記述へのリンクが残ること", async ({ page }) => {
  await page.route("**/api/v1/openapi.yaml", (route) => route.fulfill({ status: 500, body: "" }));
  await page.goto("/api/reference/v1");

  const fallback = page.locator("#api-reference-fallback");
  await expect(fallback.getByRole("link", { name: "openapi.yaml" })).toBeVisible();
  await expect(page.locator("#api-reference")).not.toContainText("トークンの持ち主のユーザーを取得する");
});

test.describe("JavaScriptが無効な場合", () => {
  test.use({ javaScriptEnabled: false });

  test("初期HTMLからAPIの概要とOpenAPI記述を読めること", async ({ page }) => {
    await page.goto("/api/reference/v1");

    const fallback = page.locator("#api-reference-fallback");
    await expect(fallback.getByRole("heading", { name: "Wikino API V1リファレンス" })).toBeVisible();
    await expect(fallback).toContainText("Wikinoの公開Web API V1のリファレンスです。");
    await expect(fallback.getByRole("link", { name: "openapi.yaml" })).toBeVisible();
  });
});
