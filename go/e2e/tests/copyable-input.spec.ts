import { expect, test } from "@playwright/test";

import { signIn } from "../helpers/auth";
import {
  cleanupTestData,
  createTestFeatureFlag,
  createTestSpace,
  createTestSpaceMember,
  createTestUser,
  type TestSpace,
  type TestUser,
} from "../helpers/database";

const TOKEN_NAME = "e2eのCLI";

let user: TestUser | undefined;
let space: TestSpace | undefined;

test.beforeAll(async () => {
  user = await createTestUser();
  space = await createTestSpace();
  await createTestSpaceMember(space.id, user.id, "admin");
  await createTestFeatureFlag(user.id, "go_public_api");
});

test.afterAll(async () => {
  if (user) {
    await cleanupTestData([user.id]);
  }
});

test.describe("発行した直後に示す秘密の値", () => {
  test("個人用アクセストークンを、欄のクリックで全体を選び、ボタンでコピーできること", async ({ page, context }) => {
    if (!user || !space) {
      throw new Error("テストデータが作成されていません");
    }
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    await page.context().clearCookies();
    await signIn(page, user);
    await page.goto(`/s/${space.identifier}/settings/personal_access_tokens/new`);
    await page.locator("#name").fill(TOKEN_NAME);
    await page.getByLabel("トピックの読み取り").check();
    await page.getByRole("button", { name: "発行する" }).click();

    const field = page.locator("#token_value");
    await expect(field).toHaveValue(/^wkp_/);
    const tokenValue = await field.inputValue();

    // 値はbase64urlで `-` を含むため、ダブルクリックの単語選択では全体が選ばれない。クリックで全体を選ぶ
    await field.click();
    const selected = await field.evaluate((element: HTMLInputElement) =>
      element.value.slice(element.selectionStart ?? 0, element.selectionEnd ?? 0),
    );
    expect(selected).toBe(tokenValue);

    const message = `トークン (${TOKEN_NAME})をコピーしました`;
    await page.getByRole("button", { name: `トークン (${TOKEN_NAME})をコピー` }).click();

    await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toBe(tokenValue);
    await expect(page.locator("#toaster .toast", { hasText: message })).toBeVisible();
    await expect(page.locator("[data-copyable-input-status]")).toHaveText(message);
  });
});
