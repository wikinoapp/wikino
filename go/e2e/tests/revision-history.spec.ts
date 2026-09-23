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

  topic = await createTestTopic(space.id, { name: "RevisionHistoryTopic" });
  await createTestTopicMember(space.id, topic.id, spaceMemberId);
});

test.describe("編集履歴", () => {
  // リビジョン差分GETフローのリグレッションガード。編集履歴カラムのバージョンをクリックすると
  // リクエストはRailsに転送されずGoハンドラーに届き、差分フラグメントがモーダルに描画される
  // 必要がある。リバースプロキシのホワイトリスト追加が欠けるとGETがRailsに転送され
  // (CIではRails未起動のため502、それ以外ではRoutingError)、何も描画されないため本テストは失敗する。
  test("編集履歴のバージョンをクリックすると差分がモーダルに描画されること", async ({ page }) => {
    const page_ = await createTestPage(space.id, topic.id, { title: "Revision History Page" });

    await page.goto(`/s/${space.identifier}/pages/${page_.number}/edit`);
    await page.waitForSelector(".cm-content");

    // リビジョンを生成する: エディタに本文を入れて手動保存する。手動保存は
    // draft_page_revisionを作成し、編集履歴カラムをOOBスワップで更新する。
    await setEditorContent(page, "E2E revision history marker body");
    await page.locator("#page-edit-save-draft-button").click();

    // 保存したリビジョンが編集履歴カラムに現れるのを待つ。
    const versionButton = page.locator("#page-revision-list button").first();
    await expect(versionButton).toBeVisible({ timeout: 5000 });

    // バージョンをクリックすると差分モーダルが開き、htmxが差分フラグメントをモーダル内へ取得する。
    await versionButton.click();

    // 差分フラグメントがモーダルに描画されること (ルーティングエラーにならないこと)。クリックした
    // バージョンは最新リビジョン (現在) で復元ボタンは意図的に隠れるため、復元ボタンではなく差分本文で
    // 確認する: 投入したマーカー文字列が差分の追加行として現れる。
    await expect(page.locator("#page-edit-revision-diff-content")).toContainText("E2E revision history marker body", {
      timeout: 5000,
    });
  });
});
