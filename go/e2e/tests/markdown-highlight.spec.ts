import { test, expect } from "@playwright/test";
import { createTestPage, createTestTopic, createTestTopicMember, loadSharedTestData } from "../helpers/database";
import { setEditorContent } from "../helpers/editor";

// エディタと差分が本文の記法に、色を付けるクラス名を当てることを確かめる。記法ごとの解析の細部は単体テスト
// (web/markdown-highlight/) で確かめ、ここではエディタと差分に色付けが組み込まれていることを見る。

const BODY_MARKDOWN = [
  "## 見出し",
  "",
  "Wikinoは**Wiki**アプリ。[[ページ]]と[リンク](https://wikino.app)と`[[コード]]`。",
  "",
  "<code>[[HTMLコード]]</code> と [[ と書く `x` [[後ろ]]",
  "",
  "<details>",
  "<summary>詳細</summary>",
  "[[折りたたみの中]] と <code>[[HTMLブロックのコード]]</code>",
  "</details>",
  "",
  "> 引用の[参考](https://example.com)",
  "",
  "| 表の見出し |",
  "| --- |",
  "| 表の値 |",
].join("\n");

let editPath: string;
let spaceIdentifier: string;
let topicId: string;
let spaceId: string;

test.beforeAll(async () => {
  const shared = loadSharedTestData();
  const topic = await createTestTopic(shared.space.id, { name: "MarkdownHighlightTopic" });
  await createTestTopicMember(shared.space.id, topic.id, shared.spaceMemberId);
  spaceId = shared.space.id;
  spaceIdentifier = shared.space.identifier;
  topicId = topic.id;
  const page_ = await createTestPage(shared.space.id, topic.id, {
    title: "Markdown Highlight Page",
    body: BODY_MARKDOWN,
  });
  editPath = `/s/${shared.space.identifier}/pages/${page_.number}/edit`;
});

test.describe("エディタのMarkdownの色付け", () => {
  test("記法にクラス名が付くこと", async ({ page }) => {
    await page.goto(editPath);
    const content = page.locator(".cm-content");
    await expect(content).toBeVisible();

    await expect(content.locator(".wikino-md-heading", { hasText: "見出し" })).toBeVisible();
    await expect(content.locator(".wikino-md-heading.wikino-md-mark", { hasText: "##" })).toBeVisible();
    await expect(content.locator(".wikino-md-strong", { hasText: /^Wiki$/ })).toBeVisible();
    await expect(content.locator(".wikino-md-wikilink", { hasText: "ページ" })).toBeVisible();
    await expect(content.locator(".wikino-md-link", { hasText: "リンク" })).toBeVisible();
    await expect(content.locator(".wikino-md-url", { hasText: "https://wikino.app" })).toBeVisible();
    await expect(content.locator(".wikino-md-code", { hasText: "[[コード]]" })).toBeVisible();
    // コードとして書いた [[...]] にはWikiリンクのクラス名を付けない
    await expect(content.locator(".wikino-md-wikilink", { hasText: "コード" })).toHaveCount(0);
    await expect(content.locator(".wikino-md-code", { hasText: "[[HTMLコード]]" })).toBeVisible();
    await expect(content.locator(".wikino-md-code", { hasText: /^x$/ })).toBeVisible();
    await expect(content.locator(".wikino-md-wikilink", { hasText: "後ろ" })).toBeVisible();
    await expect(content.locator(".wikino-md-wikilink", { hasText: "と書く" })).toHaveCount(0);
    await expect(content.locator(".wikino-md-wikilink", { hasText: "折りたたみの中" })).toBeVisible();
    await expect(content.locator(".wikino-md-wikilink", { hasText: "HTMLブロックのコード" })).toHaveCount(0);
  });

  test("記法の色が本文の文字の色と違うこと", async ({ page }) => {
    await page.goto(editPath);
    const content = page.locator(".cm-content");
    await expect(content).toBeVisible();

    const colorOf = (selector: string) =>
      content
        .locator(selector)
        .first()
        .evaluate((el) => getComputedStyle(el).color);
    const textColor = await content.evaluate((el) => getComputedStyle(el).color);

    expect(await colorOf(".wikino-md-mark")).not.toBe(textColor);
    expect(await colorOf(".wikino-md-wikilink")).not.toBe(textColor);
    expect(await colorOf(".wikino-md-code")).not.toBe(textColor);
    expect(
      await content
        .locator(".wikino-md-strong")
        .first()
        .evaluate((el) => getComputedStyle(el).fontWeight),
    ).toBe("700");
    expect(
      await content
        .locator(".wikino-md-heading")
        .first()
        .evaluate((el) => getComputedStyle(el).fontWeight),
    ).not.toBe("700");
  });

  test("見出し・太字・リンクの記号が、その記法の文字と同じ色になること", async ({ page }) => {
    await page.goto(editPath);
    const content = page.locator(".cm-content");
    await expect(content).toBeVisible();

    const colorOf = (selector: string) =>
      content
        .locator(selector)
        .first()
        .evaluate((el) => getComputedStyle(el).color);
    const headingColor = await colorOf(".wikino-md-heading:not(.wikino-md-mark)");
    const linkColor = await colorOf(".wikino-md-link:not(.wikino-md-mark):not(.wikino-md-link-destination)");
    const destinationColor = await colorOf(".wikino-md-link-destination.wikino-md-url");

    expect(await colorOf(".wikino-md-heading.wikino-md-mark")).toBe(headingColor);
    expect(await colorOf(".wikino-md-strong:not(.wikino-md-mark)")).toBe(linkColor);
    expect(await colorOf(".wikino-md-strong.wikino-md-mark")).toBe(linkColor);
    expect(await colorOf(".wikino-md-link.wikino-md-mark:not(.wikino-md-link-destination)")).toBe(linkColor);
    expect(await colorOf(".wikino-md-wikilink.wikino-md-mark")).toBe(linkColor);
    // リンクの `(...)` は、括弧も含めてテキストの側と別の色にする
    expect(destinationColor).not.toBe(linkColor);
    expect(await colorOf(".wikino-md-link-destination.wikino-md-mark")).toBe(destinationColor);
  });

  test("表の見出し行が、本文の行と同じ色になること", async ({ page }) => {
    await page.goto(editPath);
    const content = page.locator(".cm-content");
    await expect(content).toBeVisible();

    const headerLine = content.locator(".cm-line", { hasText: "表の見出し" });
    const bodyLine = content.locator(".cm-line", { hasText: "表の値" });
    await expect(headerLine).toBeVisible();
    await expect(headerLine.locator(".wikino-md-heading")).toHaveCount(0);

    const colorOf = (locator: typeof headerLine) => locator.evaluate((el) => getComputedStyle(el).color);
    expect(await colorOf(headerLine.locator(".wikino-md-mark").first())).toBe(
      await colorOf(bodyLine.locator(".wikino-md-mark").first()),
    );
  });

  test("引用の中の記法が、記法を問わず引用の色になること", async ({ page }) => {
    await page.goto(editPath);
    const content = page.locator(".cm-content");
    await expect(content).toBeVisible();

    const colorOf = (selector: string) =>
      content
        .locator(selector)
        .first()
        .evaluate((el) => getComputedStyle(el).color);
    const quoteColor = await colorOf(".wikino-md-quote.wikino-md-mark");

    expect(await colorOf(".wikino-md-link:not(.wikino-md-quote):not(.wikino-md-mark)")).not.toBe(quoteColor);
    expect(await colorOf(".wikino-md-quote.wikino-md-link:not(.wikino-md-mark)")).toBe(quoteColor);
    expect(await colorOf(".wikino-md-quote.wikino-md-link.wikino-md-mark")).toBe(quoteColor);
    expect(await colorOf(".wikino-md-quote.wikino-md-link-destination.wikino-md-url")).toBe(quoteColor);
  });
});

test.describe("差分のMarkdownの色付け", () => {
  for (const insertedBlank of [true, false]) {
    const action = insertedBlank ? "追加" : "削除";
    test(`空行を${action}した後ろの変更行に、編集履歴のモーダルで色が付くこと`, async ({ page }) => {
      const page_ = await createTestPage(spaceId, topicId, {
        title: `Markdown Highlight Diff Page ${action}`,
        body: ["## 見出し", ...(insertedBlank ? [] : [""]), "同じ行", "消す行の*強調*と<b>タグ</b>"].join("\n"),
      });

      await page.goto(`/s/${spaceIdentifier}/pages/${page_.number}/edit`);
      await page.waitForSelector(".cm-content");

      // 手動保存でリビジョンを作り、ページの本文と比べた差分を編集履歴のモーダルで開く
      await setEditorContent(
        page,
        ["## 見出し", ...(insertedBlank ? [""] : []), "同じ行", "足す行の[[ページ]]と`コード`"].join("\n"),
      );
      await page.locator("#page-edit-save-draft-button").click();
      const versionButton = page.locator("#page-revision-list button").first();
      await expect(versionButton).toBeVisible({ timeout: 5000 });
      await versionButton.click();

      const diff = page.locator("#page-edit-revision-diff-content [data-markdown-diff]");
      await expect(diff).toHaveAttribute("data-markdown-diff-highlighted", "", { timeout: 5000 });

      const rows = diff.locator("tr");
      const equalRow = rows.filter({ hasText: "見出し" });
      const deletedRow = diff.locator("tr.bg-red-50").filter({ hasText: "消す行" });
      const insertedRow = diff.locator("tr.bg-green-50").filter({ hasText: "足す行" });

      const blankSource = insertedBlank ? "new" : "old";
      await expect(
        diff.locator(`[data-markdown-diff-source="${blankSource}"][data-markdown-diff-line="2"]`),
      ).toHaveText("");
      await expect(equalRow.locator(".wikino-md-heading", { hasText: "見出し" })).toBeVisible();
      await expect(deletedRow.locator(".wikino-md-emphasis", { hasText: "強調" }).first()).toBeVisible();
      await expect(insertedRow.locator(".wikino-md-wikilink", { hasText: "ページ" }).first()).toBeVisible();
      await expect(insertedRow.locator(".wikino-md-code", { hasText: "コード" }).first()).toBeVisible();
      // 本文に書いたHTMLは要素にならず、文字のまま残る
      await expect(deletedRow).toContainText("<b>タグ</b>");
      await expect(deletedRow.locator("b")).toHaveCount(0);
    });
  }
});
