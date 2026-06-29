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
  // Guards the preview POST flow: the request must reach the Go handler (not be
  // proxied to Rails) and render the body HTML into the preview panel. With the
  // reverse-proxy whitelist or hx-vals fix missing, the POST is forwarded to
  // Rails (502 in CI, where Rails is not running) and nothing renders, so this
  // test fails.
  //
  // [Ja] プレビュー POST フローのリグレッションガード。リクエストは Rails に転送されず
  // Go ハンドラーに届き、本文 HTML がプレビューパネルに描画される必要がある。リバース
  // プロキシのホワイトリストか hx-vals の修正が欠けると POST が Rails に転送され
  // (CI では Rails 未起動のため 502)、何も描画されないため本テストは失敗する。
  test("プレビュータブを選択すると編集内容がプレビューパネルに描画されること", async ({ page }) => {
    const page_ = await createTestPage(space.id, topic.id, { title: "Preview Tab Page" });

    await page.goto(`/s/${space.identifier}/pages/${page_.number}/edit`);
    await page.waitForSelector(".cm-content");

    // Seed the editor with an identifiable body and sync it into the hidden textarea.
    //
    // [Ja] エディタに識別しやすい本文を入れて hidden textarea に同期させる。
    const marker = "E2E preview marker body";
    await setEditorContent(page, marker);

    // Selecting the preview tab makes htmx POST the form's current values.
    //
    // [Ja] プレビュータブを選択すると htmx がフォームの現在値を POST する。
    await page.locator("#page-edit-tab-preview").click();

    // The body HTML must render into the preview panel (no error).
    //
    // [Ja] プレビューパネルに本文 HTML が描画されること (エラーにならないこと)。
    await expect(page.locator("#page-edit-preview-content")).toContainText(marker, {
      timeout: 5000,
    });
  });
});
