import * as fs from "fs";
import * as path from "path";
import { test, expect } from "@playwright/test";
import {
  createTestTopic,
  createTestTopicMember,
  createTestPage,
  loadSharedTestData,
  query,
  type TestUser,
  type TestSpace,
  type TestPage,
} from "../helpers/database";
import { clearEditor, fillInEditor, getEditorContent, waitForEditorReady } from "../helpers/editor";

let user: TestUser;
let space: TestSpace;
let spaceMemberId: string;

test.beforeAll(async () => {
  const shared = loadSharedTestData();
  user = shared.user;
  space = shared.space;
  spaceMemberId = shared.spaceMemberId;
});

// photoWithLocationは、位置情報 (GPSLatitudeRef) と説明 (E2E_SECRET_MARKER) のEXIFを持つJPEG
const photoWithLocation = fs.readFileSync(path.join(__dirname, "../fixtures/photo-with-location.jpg"));
const E2E_SECRET_MARKER = "E2E-SECRET-LOCATION";

// dropFileは、ファイルをエディタへドロップする
async function dropFile(
  pwPage: import("@playwright/test").Page,
  name: string,
  type: string,
  content: Buffer,
): Promise<void> {
  await pwPage.evaluate(
    ({ name, type, base64 }) => {
      const editor = document.querySelector(".cm-editor");
      if (!editor) return;

      const bytes = Uint8Array.from(atob(base64), (c) => c.charCodeAt(0));
      const dataTransfer = new DataTransfer();
      dataTransfer.items.add(new File([bytes], name, { type }));
      editor.dispatchEvent(new DragEvent("drop", { bubbles: true, cancelable: true, dataTransfer }));
    },
    { name, type, base64: content.toString("base64") },
  );
}

async function visitPageEditor(pwPage: import("@playwright/test").Page): Promise<TestPage> {
  const topic = await createTestTopic(space.id);
  await createTestTopicMember(space.id, topic.id, spaceMemberId);
  const page_ = await createTestPage(space.id, topic.id);

  await pwPage.goto(`/s/${space.identifier}/pages/${page_.number}/edit`);
  await pwPage.waitForSelector(".cm-content");
  // 合成ドラッグ&ドロップの前に、アップロード用リスナーの登録完了を待つ。
  await waitForEditorReady(pwPage);
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

    // 楽観的プレースホルダーはdropで同期的に挿入され、検証/アップロードが
    // 決着すると消えるため、drop直後に同期で読む。dropの取りこぼしは
    // visitPageEditorのエディタ準備待ちで防いでいる。
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

        // .exeファイルをドロップ
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

    // drop直後に同期で読む: 楽観的プレースホルダーは同期的に挿入され、
    // 検証/アップロードが決着すると消える。dropの取りこぼしはvisitPageEditorの
    // エディタ準備待ちで防いでいる。
    const content = await getEditorContent(page);
    expect(content).toContain("既存のテキスト");
    expect(content).toContain("document.pdf");
  });
});

test.describe("画像のアップロードと表示", () => {
  test("ドロップした画像が本文に挿入され、プレビューと公開したページに表示され、位置情報が除かれること", async ({
    page,
  }) => {
    test.setTimeout(60_000);
    const page_ = await visitPageEditor(page);
    await page.getByRole("textbox", { name: "タイトル" }).fill("画像のアップロード");
    await clearEditor(page);

    const presigned = page.waitForResponse(
      (res) => res.request().method() === "POST" && new URL(res.url()).pathname.endsWith("/attachments/presign"),
    );
    const uploaded = page.waitForResponse((res) => res.request().method() === "PUT");
    await dropFile(page, "photo.jpg", "image/jpeg", photoWithLocation);
    expect((await presigned).status()).toBe(200);
    expect((await uploaded).status()).toBe(200);

    // アップロードが終わると、プレースホルダーが添付ファイルのMarkdownに置き換わる
    await expect.poll(() => getEditorContent(page)).toMatch(/\/attachments\/[0-9a-f-]{36}/);
    const attachmentId = (await getEditorContent(page)).match(/\/attachments\/([0-9a-f-]{36})/)![1]!;

    // プレビューで、署名付きURLから画像が読み込まれる
    await page.locator("#page-edit-tab-preview").click();
    const previewImage = page.locator(`#page-edit-preview-content img[data-attachment-id="${attachmentId}"]`);
    await expect(previewImage).toBeVisible();
    await expect.poll(() => previewImage.evaluate((img: HTMLImageElement) => img.naturalWidth)).toBe(64);

    // 公開したページの表示画面でも、画像が読み込まれる
    const published = page.waitForURL(new RegExp(`/s/${space.identifier}/pages/${page_.number}$`));
    await page.getByRole("button", { name: "トピックに公開" }).click();
    await published;
    const pageImage = page.locator(`img[data-attachment-id="${attachmentId}"]`);
    await expect.poll(() => pageImage.evaluate((img: HTMLImageElement) => img.naturalWidth)).toBe(64);

    // アップロード後の処理が完了し、取得した画像から位置情報が除かれている
    await expect
      .poll(
        async () => {
          const result = await query(`SELECT processing_status FROM attachments WHERE id = $1 AND space_id = $2`, [
            attachmentId,
            space.id,
          ]);
          return result.rows[0]?.processing_status;
        },
        { timeout: 30_000 },
      )
      .toBe(2);
    const response = await page.request.get(`/attachments/${attachmentId}`);
    expect(response.status()).toBe(200);
    const body = await response.body();
    expect(body.subarray(0, 2).toString("hex")).toBe("ffd8");
    expect(body.includes(E2E_SECRET_MARKER)).toBe(false);
    expect(photoWithLocation.includes(E2E_SECRET_MARKER)).toBe(true);
  });
});
