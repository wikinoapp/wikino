import { test, expect } from "@playwright/test";
import {
  createTestTopic,
  createTestTopicMember,
  createTestPage,
  loadSharedTestData,
  type TestUser,
  type TestSpace,
  type TestTopic,
} from "../helpers/database";
import { setEditorContent } from "../helpers/editor";

let user: TestUser;
let space: TestSpace;
let spaceMemberId: string;
let topic: TestTopic;

test.beforeAll(async () => {
  const shared = loadSharedTestData();
  user = shared.user;
  space = shared.space;
  spaceMemberId = shared.spaceMemberId;

  topic = await createTestTopic(space.id, { name: "PreviewTabTopic" });
  await createTestTopicMember(space.id, topic.id, spaceMemberId);
});

test.describe("プレビュータブ", () => {
  // プレビューPOSTフローのリグレッションガード。リクエストはRailsに転送されず
  // Goハンドラーに届き、本文HTMLがプレビューパネルに描画される必要がある。リバース
  // プロキシのホワイトリストかhx-valsの修正が欠けるとPOSTがRailsに転送され
  // (CIではRails未起動のため502)、何も描画されないため本テストは失敗する。
  test("プレビュータブを選択すると編集内容がプレビューパネルに描画されること", async ({ page }) => {
    const page_ = await createTestPage(space.id, topic.id, { title: "Preview Tab Page" });

    await page.goto(`/s/${space.identifier}/pages/${page_.number}/edit`);
    await page.waitForSelector(".cm-content");

    // エディタに識別しやすい本文を入れてhidden textareaに同期させる。
    const marker = "E2E preview marker body";
    await setEditorContent(page, marker);

    // プレビュータブを選択するとhtmxがフォームの現在値をPOSTする。
    await page.locator("#page-edit-tab-preview").click();

    // プレビューパネルに本文HTMLが描画されること (エラーにならないこと)。
    await expect(page.locator("#page-edit-preview-content")).toContainText(marker, {
      timeout: 5000,
    });
  });
});
