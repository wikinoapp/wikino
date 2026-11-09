import { expect, test } from "@playwright/test";

import { signIn } from "../helpers/auth";
import {
  cleanupTestData,
  createTestPage,
  createTestSpace,
  createTestSpaceMember,
  createTestTopic,
  createTestTopicMember,
  createTestUser,
  type TestSpace,
  type TestTopic,
  type TestUser,
} from "../helpers/database";

let user: TestUser | undefined;
let space: TestSpace | undefined;
let topic: TestTopic | undefined;

test.beforeAll(async () => {
  user = await createTestUser();
  space = await createTestSpace();
  const memberId = await createTestSpaceMember(space.id, user.id, "editor");
  topic = await createTestTopic(space.id);
  await createTestTopicMember(space.id, topic.id, memberId);
});

test.afterAll(async () => {
  if (user) {
    await cleanupTestData([user.id]);
  }
});

test.describe("ページのピン留めのキーボード操作", () => {
  for (const key of ["Enter", "Space"]) {
    test(`${key}キーでピンを打ち、外せること`, async ({ page }) => {
      if (!user || !space || !topic) {
        throw new Error("テストデータが作成されていません");
      }
      const target = await createTestPage(space.id, topic.id, { title: `ピン留めの確認 (${key})` });

      await page.context().clearCookies();
      await signIn(page, user);
      await page.goto(`/s/${space.identifier}/pages/${target.number}`);

      const trigger = page.locator("#page-actions-dropdown-trigger");
      // Basecoatはトリガーにフォーカスを残して項目を選び、選択中のmenuitemを実行する。
      await trigger.press("ArrowDown");
      await trigger.press("ArrowDown");
      await expect(page.getByRole("menuitem", { name: "ピン留めする", exact: true })).toHaveClass(/active/);
      await trigger.press(key);
      await expect(page.locator("#toaster .toast")).toContainText("ページをピン留めしました");

      await trigger.press("ArrowDown");
      await trigger.press("ArrowDown");
      await expect(page.getByRole("menuitem", { name: "ピン留めを外す", exact: true })).toHaveClass(/active/);
      await trigger.press(key);
      await expect(page.locator("#toaster .toast")).toContainText("ページのピン留めを外しました");

      await trigger.press("ArrowDown");
      await expect(page.getByRole("menuitem", { name: "ピン留めする", exact: true })).toBeVisible();
    });
  }
});
