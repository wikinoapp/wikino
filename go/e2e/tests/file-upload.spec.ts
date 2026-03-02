import { test, expect } from "@playwright/test";
import {
  createTestUser,
  createTestSpace,
  createTestSpaceMember,
  createTestTopic,
  createTestTopicMember,
  createTestPage,
  cleanupTestData,
  type TestUser,
  type TestSpace,
  type TestPage,
} from "../helpers/database";
import { signIn } from "../helpers/auth";
import { clearEditor, fillInEditor, getEditorContent } from "../helpers/editor";

let user: TestUser;
let space: TestSpace;

test.beforeAll(async () => {
  user = await createTestUser();
  space = await createTestSpace();
  const spaceMemberId = await createTestSpaceMember(space.id, user.id);
  const topic = await createTestTopic(space.id);
  await createTestTopicMember(space.id, topic.id, spaceMemberId);
});

test.afterAll(async () => {
  await cleanupTestData([user.id]);
});

async function visitPageEditor(pwPage: import("@playwright/test").Page): Promise<TestPage> {
  const topic = await createTestTopic(space.id);
  const spaceMemberResult = await import("../helpers/database").then((db) =>
    db.query("SELECT id FROM space_members WHERE space_id = $1 AND user_id = $2 LIMIT 1", [space.id, user.id]),
  );
  const spaceMemberId = spaceMemberResult.rows[0].id;
  await createTestTopicMember(space.id, topic.id, spaceMemberId);
  const page_ = await createTestPage(space.id, topic.id);

  await signIn(pwPage, user);
  await pwPage.goto(`/s/${space.identifier}/pages/${page_.number}/edit`);
  await pwPage.waitForSelector(".cm-content");
  return page_;
}

test.describe("ドラッグ&ドロップによるファイルアップロード", () => {
  test("ファイルをドラッグするとドロップゾーンが表示されること", async ({ page }) => {
    await visitPageEditor(page);

    const editor = page.locator(".cm-editor");

    // dragenterイベントをディスパッチしてドロップゾーンを表示
    await editor.evaluate((el) => {
      const event = new DragEvent("dragenter", {
        bubbles: true,
        cancelable: true,
        dataTransfer: new DataTransfer(),
      });
      Object.defineProperty(event, "dataTransfer", {
        value: { types: ["Files"], dropEffect: "none" },
      });
      el.dispatchEvent(event);
    });

    // ドロップゾーンが表示されることを確認
    await expect(page.locator(".cm-drop-zone")).toBeVisible();
  });

  test("ドラッグがエディタから離れるとドロップゾーンが非表示になること", async ({ page }) => {
    await visitPageEditor(page);

    const editor = page.locator(".cm-editor");

    // dragenterイベントをディスパッチ
    await editor.evaluate((el) => {
      const enterEvent = new DragEvent("dragenter", {
        bubbles: true,
        cancelable: true,
      });
      Object.defineProperty(enterEvent, "dataTransfer", {
        value: { types: ["Files"], dropEffect: "none" },
      });
      el.dispatchEvent(enterEvent);
    });

    await expect(page.locator(".cm-drop-zone")).toBeVisible();

    // dragleaveイベントをディスパッチ
    await editor.evaluate((el) => {
      const leaveEvent = new DragEvent("dragleave", {
        bubbles: true,
        cancelable: true,
      });
      el.dispatchEvent(leaveEvent);
    });

    // ドロップゾーンが非表示になることを確認
    await expect(page.locator(".cm-drop-zone")).toBeHidden();
  });

  test("ファイルをドロップするとfile-dropカスタムイベントがディスパッチされること", async ({ page }) => {
    await visitPageEditor(page);

    // file-dropイベントのリスナーを設定
    const eventFired = await page.evaluate(() => {
      return new Promise<boolean>((resolve) => {
        const editor = document.querySelector(".cm-editor");
        if (!editor) {
          resolve(false);
          return;
        }

        editor.addEventListener("file-drop", () => {
          resolve(true);
        });

        // dropイベントをディスパッチ
        const dataTransfer = new DataTransfer();
        const file = new File(["test content"], "test.txt", { type: "text/plain" });
        dataTransfer.items.add(file);

        const dropEvent = new DragEvent("drop", {
          bubbles: true,
          cancelable: true,
          dataTransfer,
        });
        editor.dispatchEvent(dropEvent);
      });
    });

    expect(eventFired).toBe(true);
  });

  test("ファイルをドロップするとアップロードプレースホルダーが挿入されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);

    // dropイベントをディスパッチ
    await page.evaluate(() => {
      const editor = document.querySelector(".cm-editor");
      if (!editor) return;

      const dataTransfer = new DataTransfer();
      const file = new File(["test content"], "test.txt", { type: "text/plain" });
      dataTransfer.items.add(file);

      const dropEvent = new DragEvent("drop", {
        bubbles: true,
        cancelable: true,
        dataTransfer,
      });
      editor.dispatchEvent(dropEvent);
    });

    // プレースホルダーが挿入されることを確認（ファイル名を含む）
    const content = await getEditorContent(page);
    expect(content).toContain("test.txt");
  });

  test("許可されていないファイル形式をドロップするとエラーメッセージが表示されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);

    // flash-toast:showイベントをリッスン
    const errorMessagePromise = page.evaluate(() => {
      return new Promise<string>((resolve) => {
        document.addEventListener("flash-toast:show", ((e: CustomEvent) => {
          resolve(e.detail.messageHtml as string);
        }) as EventListener);

        // .exe ファイルをドロップ
        const editor = document.querySelector(".cm-editor");
        if (!editor) {
          resolve("");
          return;
        }

        const dataTransfer = new DataTransfer();
        const file = new File(["MZ"], "malware.exe", { type: "application/x-msdownload" });
        dataTransfer.items.add(file);

        const dropEvent = new DragEvent("drop", {
          bubbles: true,
          cancelable: true,
          dataTransfer,
        });
        editor.dispatchEvent(dropEvent);
      });
    });

    const errorMessage = await errorMessagePromise;
    expect(errorMessage).toContain("ファイル形式");
  });

  test("ファイルサイズが制限を超えるとエラーメッセージが表示されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);

    // flash-toast:showイベントをリッスン
    const errorMessagePromise = page.evaluate(() => {
      return new Promise<string>((resolve) => {
        document.addEventListener("flash-toast:show", ((e: CustomEvent) => {
          resolve(e.detail.messageHtml as string);
        }) as EventListener);

        // 11MBの画像ファイルを作成
        const editor = document.querySelector(".cm-editor");
        if (!editor) {
          resolve("");
          return;
        }

        const size = 11 * 1024 * 1024;
        const buffer = new ArrayBuffer(size);
        const file = new File([buffer], "large-image.png", { type: "image/png" });

        const dataTransfer = new DataTransfer();
        dataTransfer.items.add(file);

        const dropEvent = new DragEvent("drop", {
          bubbles: true,
          cancelable: true,
          dataTransfer,
        });
        editor.dispatchEvent(dropEvent);
      });
    });

    const errorMessage = await errorMessagePromise;
    expect(errorMessage).toContain("サイズ");
  });

  test("既存のテキストがある位置にファイルをドロップしてもテキストが保持されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await fillInEditor(page, "既存のテキスト");

    // dropイベントをディスパッチ
    await page.evaluate(() => {
      const editor = document.querySelector(".cm-editor");
      if (!editor) return;

      const dataTransfer = new DataTransfer();
      const file = new File(["content"], "document.pdf", { type: "application/pdf" });
      dataTransfer.items.add(file);

      const dropEvent = new DragEvent("drop", {
        bubbles: true,
        cancelable: true,
        dataTransfer,
      });
      editor.dispatchEvent(dropEvent);
    });

    const content = await getEditorContent(page);
    expect(content).toContain("既存のテキスト");
    expect(content).toContain("document.pdf");
  });
});
