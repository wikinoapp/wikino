import { test, expect } from "@playwright/test";
import {
  createTestTopic,
  createTestTopicMember,
  createTestPage,
  loadSharedTestData,
  type TestUser,
  type TestSpace,
  type TestPage,
} from "../helpers/database";
import { clearEditor, fillInEditor, getEditorContent } from "../helpers/editor";

let user: TestUser;
let space: TestSpace;
let spaceMemberId: string;

test.beforeAll(async () => {
  const shared = loadSharedTestData();
  user = shared.user;
  space = shared.space;
  spaceMemberId = shared.spaceMemberId;
});

async function visitPageEditor(pwPage: import("@playwright/test").Page): Promise<TestPage> {
  const topic = await createTestTopic(space.id);
  await createTestTopicMember(space.id, topic.id, spaceMemberId);
  const page_ = await createTestPage(space.id, topic.id);

  await pwPage.goto(`/s/${space.identifier}/pages/${page_.number}/edit`);
  await pwPage.waitForSelector(".cm-content");
  return page_;
}

test.describe("ペーストによるファイルアップロード", () => {
  test("テキストのペーストは通常通り処理されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);

    const editor = page.locator(".cm-content");
    await editor.click();

    // テキストをペースト
    await page.evaluate(() => {
      const cmContent = document.querySelector(".cm-content");
      if (!cmContent) return;

      const clipboardData = new DataTransfer();
      clipboardData.setData("text/plain", "ペーストされたテキスト");

      const pasteEvent = new ClipboardEvent("paste", {
        bubbles: true,
        cancelable: true,
        clipboardData,
      });
      cmContent.dispatchEvent(pasteEvent);
    });

    const content = await getEditorContent(page);
    expect(content).toContain("ペーストされたテキスト");
  });

  test("画像をペーストするとmedia-pasteカスタムイベントがディスパッチされること", async ({ page }) => {
    await visitPageEditor(page);

    const eventFired = await page.evaluate(() => {
      return new Promise<boolean>((resolve) => {
        const editor = document.querySelector(".cm-editor");
        if (!editor) {
          resolve(false);
          return;
        }

        editor.addEventListener("media-paste", () => {
          resolve(true);
        });

        // 画像ファイルのペーストイベントをディスパッチ
        const cmContent = document.querySelector(".cm-content");
        if (!cmContent) {
          resolve(false);
          return;
        }

        const dataTransfer = new DataTransfer();
        const file = new File(["fake-image-data"], "screenshot.png", { type: "image/png" });
        dataTransfer.items.add(file);

        const pasteEvent = new ClipboardEvent("paste", {
          bubbles: true,
          cancelable: true,
          clipboardData: dataTransfer,
        });
        cmContent.dispatchEvent(pasteEvent);
      });
    });

    expect(eventFired).toBe(true);
  });

  test("PDFをペーストするとfile-pasteカスタムイベントがディスパッチされること", async ({ page }) => {
    await visitPageEditor(page);

    const eventFired = await page.evaluate(() => {
      return new Promise<boolean>((resolve) => {
        const editor = document.querySelector(".cm-editor");
        if (!editor) {
          resolve(false);
          return;
        }

        editor.addEventListener("file-paste", () => {
          resolve(true);
        });

        const cmContent = document.querySelector(".cm-content");
        if (!cmContent) {
          resolve(false);
          return;
        }

        const dataTransfer = new DataTransfer();
        const file = new File(["fake-pdf-data"], "document.pdf", { type: "application/pdf" });
        dataTransfer.items.add(file);

        const pasteEvent = new ClipboardEvent("paste", {
          bubbles: true,
          cancelable: true,
          clipboardData: dataTransfer,
        });
        cmContent.dispatchEvent(pasteEvent);
      });
    });

    expect(eventFired).toBe(true);
  });

  test("画像をペーストするとアップロードプレースホルダーが挿入されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);

    await page.evaluate(() => {
      const cmContent = document.querySelector(".cm-content");
      if (!cmContent) return;

      const dataTransfer = new DataTransfer();
      const file = new File(["fake-image"], "photo.jpg", { type: "image/jpeg" });
      dataTransfer.items.add(file);

      const pasteEvent = new ClipboardEvent("paste", {
        bubbles: true,
        cancelable: true,
        clipboardData: dataTransfer,
      });
      cmContent.dispatchEvent(pasteEvent);
    });

    const content = await getEditorContent(page);
    expect(content).toContain("photo.jpg");
  });

  test("既存テキストの後に画像をペーストしてもテキストが保持されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await fillInEditor(page, "先に入力したテキスト");

    await page.evaluate(() => {
      const cmContent = document.querySelector(".cm-content");
      if (!cmContent) return;

      const dataTransfer = new DataTransfer();
      const file = new File(["fake-image"], "screenshot.png", { type: "image/png" });
      dataTransfer.items.add(file);

      const pasteEvent = new ClipboardEvent("paste", {
        bubbles: true,
        cancelable: true,
        clipboardData: dataTransfer,
      });
      cmContent.dispatchEvent(pasteEvent);
    });

    const content = await getEditorContent(page);
    expect(content).toContain("先に入力したテキスト");
    expect(content).toContain("screenshot.png");
  });
});
