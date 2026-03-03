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
import { fillInEditor } from "../helpers/editor";

let user: TestUser;
let space: TestSpace;
let spaceMemberId: string;
let topic: TestTopic;

test.beforeAll(async () => {
  const shared = loadSharedTestData();
  user = shared.user;
  space = shared.space;
  spaceMemberId = shared.spaceMemberId;

  topic = await createTestTopic(space.id, { name: "TestTopic" });
  await createTestTopicMember(space.id, topic.id, spaceMemberId);

  // 補完候補として表示されるページを作成
  await createTestPage(space.id, topic.id, { title: "Page Alpha" });
  await createTestPage(space.id, topic.id, { title: "Page Beta" });
});

async function visitPageEditor(pwPage: import("@playwright/test").Page) {
  const editTopic = await createTestTopic(space.id);
  await createTestTopicMember(space.id, editTopic.id, spaceMemberId);
  const page_ = await createTestPage(space.id, editTopic.id);

  await pwPage.goto(`/s/${space.identifier}/pages/${page_.number}/edit`);
  await pwPage.waitForSelector(".cm-content");
}

test.describe("Wikiリンク補完", () => {
  test.describe.configure({ mode: "serial" });
  test("[[を入力すると補完候補が表示されること", async ({ page }) => {
    await visitPageEditor(page);

    await fillInEditor(page, "[[Page");

    // 補完候補アイテムが表示されるまで待つ
    // CodeMirrorの非同期補完ではツールチップコンテナが先に表示され、
    // APIレスポンス後に補完候補アイテムがレンダリングされるため、アイテム自体を待つ
    await expect(page.locator(".cm-tooltip-autocomplete .cm-completionLabel").first()).toBeVisible({
      timeout: 5000,
    });

    // 補完候補のテキストを取得して検証
    const labels = await page.locator(".cm-completionLabel").allTextContents();
    expect(labels.length).toBeGreaterThan(0);
    expect(labels.some((label) => label.includes("Page Alpha") || label.includes("Page Beta"))).toBe(true);
  });

  test("補完候補にトピック名/ページタイトルの形式で表示されること", async ({ page }) => {
    await visitPageEditor(page);

    await fillInEditor(page, "[[Page");

    await expect(page.locator(".cm-tooltip-autocomplete .cm-completionLabel").first()).toBeVisible({
      timeout: 5000,
    });

    const labels = await page.locator(".cm-completionLabel").allTextContents();
    // トピック名/ページタイトル の形式であること
    expect(labels.some((label) => label.includes("TestTopic/Page"))).toBe(true);
  });

  test("補完候補を選択するとテキストが挿入されること", async ({ page }) => {
    await visitPageEditor(page);

    await fillInEditor(page, "[[Page");

    await expect(page.locator(".cm-tooltip-autocomplete .cm-completionLabel").first()).toBeVisible({
      timeout: 5000,
    });

    // 最初の補完候補をクリックして選択
    await page.locator(".cm-tooltip-autocomplete li[role='option']").first().click();

    // 補完候補が閉じること
    await expect(page.locator(".cm-tooltip-autocomplete")).toBeHidden();

    // テキストが挿入されていること（[[トピック名/ページタイトル の形式）
    const textarea = page.locator("#page_body");
    const value = await textarea.inputValue();
    expect(value).toContain("[[TestTopic/Page");
  });
});
