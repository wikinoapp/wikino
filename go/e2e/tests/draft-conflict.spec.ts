import { test, expect, type Page } from "@playwright/test";
import {
  createTestTopic,
  createTestTopicMember,
  createTestPage,
  loadSharedTestData,
  query,
  type TestSpace,
  type TestPage,
} from "../helpers/database";
import { getEditorContent, setEditorContent, waitForEditorReady } from "../helpers/editor";

let space: TestSpace;
let spaceMemberId: string;

test.beforeAll(async () => {
  const shared = loadSharedTestData();
  space = shared.space;
  spaceMemberId = shared.spaceMemberId;
});

async function visitPageEditor(pwPage: Page): Promise<TestPage> {
  const topic = await createTestTopic(space.id);
  await createTestTopicMember(space.id, topic.id, spaceMemberId);
  const page_ = await createTestPage(space.id, topic.id, { title: "Draft Conflict Page" });

  await pwPage.goto(`/s/${space.identifier}/pages/${page_.number}/edit`);
  await waitForEditorReady(pwPage);
  return page_;
}

// autosaveは本文を書き換え、自動保存の応答を待って、その状態を返す
async function autosave(pwPage: Page, body: string): Promise<number> {
  const response = pwPage.waitForResponse(
    (res) => res.request().method() === "PATCH" && new URL(res.url()).pathname.endsWith("/draft_page"),
  );
  await setEditorContent(pwPage, body);
  return (await response).status();
}

// updateDraftElsewhereは、APIでの更新のように編集画面の外で下書きを書き換え、下書きの版を変える
async function updateDraftElsewhere(page_: TestPage, body: string): Promise<void> {
  const result = await query(
    `UPDATE draft_pages SET body = $1, modified_at = NOW(), updated_at = NOW() WHERE page_id = $2 AND space_id = $3`,
    [body, page_.id, space.id],
  );
  expect(result.rowCount).toBe(1);
}

async function draftBody(page_: TestPage): Promise<string> {
  const result = await query(`SELECT body FROM draft_pages WHERE page_id = $1 AND space_id = $2`, [page_.id, space.id]);
  return result.rows[0].body;
}

async function pageBody(page_: TestPage): Promise<string> {
  const result = await query(`SELECT body FROM pages WHERE id = $1 AND space_id = $2`, [page_.id, space.id]);
  return result.rows[0].body;
}

// publishは公開のボタンを押し、公開の応答の状態を返す
async function publish(pwPage: Page, button: string): Promise<number> {
  const response = pwPage.waitForResponse(
    (res) => res.request().method() === "POST" && new URL(res.url()).pathname.match(/\/pages\/\d+$/) !== null,
  );
  await pwPage.getByRole("button", { name: button }).click();
  return (await response).status();
}

test.describe("下書きの競合", () => {
  test("別の場所で下書きが更新されたら上書きせずに知らせ、「このまま上書きする」で画面の内容を保存すること", async ({
    page,
  }) => {
    const page_ = await visitPageEditor(page);
    expect(await autosave(page, "画面で書いた本文")).toBe(204);
    await updateDraftElsewhere(page_, "APIで書いた本文");

    expect(await autosave(page, "画面で書き足した本文")).toBe(409);

    const alert = page.getByRole("alert").filter({ hasText: "下書きが別の場所で更新されています" });
    await expect(alert).toBeVisible();
    expect(await draftBody(page_)).toBe("APIで書いた本文");
    expect(await getEditorContent(page)).toBe("画面で書き足した本文");

    const overwritten = page.waitForResponse(
      (res) => res.request().method() === "PATCH" && new URL(res.url()).pathname.endsWith("/draft_page"),
    );
    await page.getByRole("button", { name: "このまま上書きする" }).click();
    expect((await overwritten).status()).toBe(204);

    await expect(alert).toBeHidden();
    expect(await draftBody(page_)).toBe("画面で書き足した本文");

    // 上書きのあとは自動保存を再開する
    expect(await autosave(page, "上書きのあとに書いた本文")).toBe(204);
    expect(await draftBody(page_)).toBe("上書きのあとに書いた本文");
  });

  test("「最新の内容を読み込む」で、別の場所で更新された下書きを表示すること", async ({ page }) => {
    const page_ = await visitPageEditor(page);
    expect(await autosave(page, "画面で書いた本文")).toBe(204);
    await updateDraftElsewhere(page_, "APIで書いた本文");
    expect(await autosave(page, "画面で書き足した本文")).toBe(409);

    page.once("dialog", (dialog) => void dialog.accept());
    const reloaded = page.waitForEvent("framenavigated", (frame) => frame === page.mainFrame());
    await page.getByRole("button", { name: "最新の内容を読み込む" }).click();

    await reloaded;
    await waitForEditorReady(page);
    await expect.poll(() => getEditorContent(page)).toBe("APIで書いた本文");
    await expect(page.getByRole("alert").filter({ hasText: "下書きが別の場所で更新されています" })).toBeHidden();
  });

  test("別の場所で下書きが更新されたら公開せずに知らせ、「このまま上書きする」で画面の内容を公開すること", async ({
    page,
  }) => {
    const page_ = await visitPageEditor(page);
    expect(await autosave(page, "画面で書いた本文")).toBe(204);
    await updateDraftElsewhere(page_, "APIで書いた本文");

    expect(await publish(page, "トピックに公開")).toBe(409);

    // 送信した内容を残した編集画面で、競合を知らせる
    await waitForEditorReady(page);
    const alert = page.getByRole("alert").filter({ hasText: "下書きが別の場所で更新されています" });
    await expect(alert).toBeVisible();
    expect(await getEditorContent(page)).toBe("画面で書いた本文");
    expect(await draftBody(page_)).toBe("APIで書いた本文");
    expect(await pageBody(page_)).not.toBe("画面で書いた本文");

    const published = page.waitForURL(new RegExp(`/s/${space.identifier}/pages/${page_.number}$`));
    await page.getByRole("button", { name: "このまま上書きする" }).click();
    await published;

    expect(await pageBody(page_)).toBe("画面で書いた本文");
    const drafts = await query(`SELECT 1 FROM draft_pages WHERE page_id = $1 AND space_id = $2`, [page_.id, space.id]);
    expect(drafts.rowCount).toBe(0);
  });

  for (const missingDigest of [false, true]) {
    const trigger = missingDigest ? "版を送らない公開" : "公開の版の不一致";
    test(`${trigger}で競合したあと、「最新の内容を読み込む」で最新の下書きを編集画面に表示すること`, async ({
      page,
    }) => {
      const page_ = await visitPageEditor(page);
      await page.locator("#page_title").fill("画面で書いたタイトル");
      expect(await autosave(page, "画面で書いた本文")).toBe(204);
      await query(
        `UPDATE draft_pages SET title = $1, body = $2, modified_at = NOW(), updated_at = NOW() WHERE page_id = $3 AND space_id = $4`,
        ["APIで書いたタイトル", "APIで書いた本文", page_.id, space.id],
      );
      if (missingDigest) {
        await page.locator('input[name="draft_page_digest"]').evaluate((field) => field.remove());
      }

      expect(await publish(page, "トピックに公開")).toBe(409);
      await waitForEditorReady(page);
      await expect(page).toHaveURL(new RegExp(`/pages/${page_.number}$`));
      const alert = page.getByRole("alert").filter({ hasText: "下書きが別の場所で更新されています" });
      await expect(alert).toBeVisible();

      page.once("dialog", (dialog) => void dialog.accept());
      const reloaded = page.waitForURL(new RegExp(`/pages/${page_.number}/edit$`));
      await page.getByRole("button", { name: "最新の内容を読み込む" }).click();
      await reloaded;
      await waitForEditorReady(page);

      await expect(page.locator("#page_title")).toHaveValue("APIで書いたタイトル");
      await expect.poll(() => getEditorContent(page)).toBe("APIで書いた本文");
      await expect(alert).toBeHidden();
    });
  }
});
