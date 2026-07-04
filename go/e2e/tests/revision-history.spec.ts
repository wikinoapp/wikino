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
  // Guards the revision diff GET flow: clicking a version in the edit-history column must reach
  // the Go handler (not be proxied to Rails) and render the diff fragment into the modal. With the
  // reverse-proxy whitelist entry missing, the GET is forwarded to Rails (502 in CI, where Rails is
  // not running; RoutingError otherwise) and nothing renders, so this test fails.
  //
  // [Ja] リビジョン差分 GET フローのリグレッションガード。編集履歴カラムのバージョンをクリックすると
  // リクエストは Rails に転送されず Go ハンドラーに届き、差分フラグメントがモーダルに描画される
  // 必要がある。リバースプロキシのホワイトリスト追加が欠けると GET が Rails に転送され
  // (CI では Rails 未起動のため 502、それ以外では RoutingError)、何も描画されないため本テストは失敗する。
  test("編集履歴のバージョンをクリックすると差分がモーダルに描画されること", async ({ page }) => {
    const page_ = await createTestPage(space.id, topic.id, { title: "Revision History Page" });

    await page.goto(`/s/${space.identifier}/pages/${page_.number}/edit`);
    await page.waitForSelector(".cm-content");

    // Generate a revision: seed the editor and save the draft manually. The manual save creates a
    // draft_page_revision and OOB-swaps the edit history column.
    //
    // [Ja] リビジョンを生成する: エディタに本文を入れて手動保存する。手動保存は
    // draft_page_revision を作成し、編集履歴カラムを OOB スワップで更新する。
    await setEditorContent(page, "E2E revision history marker body");
    await page.locator("#page-edit-save-draft-button").click();

    // Wait for the saved revision to appear in the edit history column.
    // [Ja] 保存したリビジョンが編集履歴カラムに現れるのを待つ。
    const versionButton = page.locator("#page-revision-list button").first();
    await expect(versionButton).toBeVisible({ timeout: 5000 });

    // Clicking the version opens the diff modal and htmx GETs the diff fragment into it.
    // [Ja] バージョンをクリックすると差分モーダルが開き、htmx が差分フラグメントをモーダル内へ取得する。
    await versionButton.click();

    // The diff fragment must render into the modal — no routing error. The clicked version is the
    // newest revision (the current one), whose restore button is intentionally hidden, so assert on
    // the diff body instead: the seeded marker text shows up as an added line in the rendered diff.
    //
    // [Ja] 差分フラグメントがモーダルに描画されること (ルーティングエラーにならないこと)。クリックした
    // バージョンは最新リビジョン (現在) で復元ボタンは意図的に隠れるため、復元ボタンではなく差分本文で
    // 確認する: 投入したマーカー文字列が差分の追加行として現れる。
    await expect(page.locator("#page-edit-revision-diff-content")).toContainText("E2E revision history marker body", {
      timeout: 5000,
    });
  });
});
