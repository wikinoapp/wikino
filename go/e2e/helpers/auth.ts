import { type Page } from "@playwright/test";
import { type TestUser } from "./database";

/**
 * テストユーザーでサインインする
 * Go版のサインインフォーム経由でログインし、セッションCookieを取得する
 */
export async function signIn(page: Page, user: TestUser): Promise<void> {
  await page.goto("/sign_in");

  await page.locator('input[name="email"]').fill(user.email);
  await page.locator('input[name="password"]').fill(user.password);
  await page.locator('button[type="submit"]').click();

  // サインイン後のリダイレクトを待機
  await page.waitForURL((url) => !url.pathname.includes("/sign_in"));
}
