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
import {
  clearEditor,
  setEditorContent,
  getEditorContent,
  pressTabInEditor,
  pressShiftTabInEditor,
  moveCursorToStart,
  setCursorPosition,
  getCursorPosition,
  selectLineWithNewline,
  selectMultipleLinesWithNewline,
  getSelectionInfo,
  fillInEditor,
} from "../helpers/editor";

let user: TestUser;
let space: TestSpace;
let spaceMemberId: string;
let page_: TestPage;

test.beforeAll(async () => {
  const shared = loadSharedTestData();
  user = shared.user;
  space = shared.space;
  spaceMemberId = shared.spaceMemberId;
});

async function visitPageEditor(pwPage: import("@playwright/test").Page) {
  const topic = await createTestTopic(space.id);
  await createTestTopicMember(space.id, topic.id, spaceMemberId);
  page_ = await createTestPage(space.id, topic.id);

  await pwPage.goto(`/s/${space.identifier}/pages/${page_.number}/edit`);
  await pwPage.waitForSelector(".cm-content");
}

test.describe("タブキーによるインデント機能", () => {
  test("タブキーを押すと半角スペース2つが挿入されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await fillInEditor(page, "テスト");
    await pressTabInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("テスト  ");
  });

  test("行頭でタブキーを押すと行頭に半角スペース2つが挿入されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await fillInEditor(page, "テスト");
    await moveCursorToStart(page);
    await pressTabInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("  テスト");
  });

  test("Shift+タブキーを押すとインデントが削除されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await fillInEditor(page, "  インデント付きテキスト");
    await moveCursorToStart(page);
    await pressShiftTabInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("インデント付きテキスト");
  });

  test("リスト項目の先頭でタブキーを押すとネストされたリスト項目になること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- a\n- ");
    await setCursorPosition(page, 2, 2);
    await pressTabInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- a\n  - ");
  });

  test("タスクリスト項目の先頭でタブキーを押すとネストされたタスクリスト項目になること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- [ ] タスク1\n- [ ] ");
    await setCursorPosition(page, 2, 6);
    await pressTabInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- [ ] タスク1\n  - [ ] ");
  });

  test("既にインデントされたリスト項目でタブキーを押すとさらにネストされること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "  - インデント1\n  - ");
    await setCursorPosition(page, 2, 4);
    await pressTabInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("  - インデント1\n    - ");
  });

  test("行選択でタブキーを押すと選択した行のみインデントが追加されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- a\n- b\n- c");
    await selectLineWithNewline(page, 2);
    await pressTabInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- a\n  - b\n- c");

    const cursor = await getCursorPosition(page);
    expect(cursor.line).toBe(3);
    expect(cursor.column).toBe(0);
  });

  test("行選択でShift+タブキーを押すと選択した行のみインデントが削除されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- a\n  - b\n- c");
    await selectLineWithNewline(page, 2);
    await pressShiftTabInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- a\n- b\n- c");

    const cursor = await getCursorPosition(page);
    expect(cursor.line).toBe(3);
    expect(cursor.column).toBe(0);
  });

  test("複数行選択でタブキーを押すと選択した行のみインデントが追加されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "行1\n行2\n行3\n行4");
    await selectMultipleLinesWithNewline(page, 2, 3);
    await pressTabInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("行1\n  行2\n  行3\n行4");

    const cursor = await getCursorPosition(page);
    expect(cursor.line).toBe(4);
    expect(cursor.column).toBe(0);
  });

  test("行選択でタブキーを押すと選択範囲が維持されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- a\n- b\n- c");
    await selectLineWithNewline(page, 2);
    await pressTabInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- a\n  - b\n- c");

    const selection = await getSelectionInfo(page);
    expect(selection.hasSelection).toBe(true);
    expect(selection.fromLine).toBe(2);
    expect(selection.fromColumn).toBe(0);
    expect(selection.toLine).toBe(3);
    expect(selection.toColumn).toBe(0);
  });

  test("行選択でタブキーを押した後deleteキーで行全体が削除されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- a\n- b\n- c");
    await selectLineWithNewline(page, 2);
    await pressTabInEditor(page);

    await page.locator(".cm-content").press("Delete");

    const content = await getEditorContent(page);
    expect(content).toBe("- a\n- c");
  });

  test("リスト項目のマーカー直後にテキストがある状態でタブキーを押すとネストされたリストアイテムになること", async ({
    page,
  }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- a\n- b");
    await setCursorPosition(page, 2, 2);
    await pressTabInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- a\n  - b");

    const cursor = await getCursorPosition(page);
    expect(cursor.line).toBe(2);
    expect(cursor.column).toBe(4);
  });

  test("タスクリスト項目のマーカー直後にテキストがある状態でタブキーを押すとネストされたタスクリストアイテムになること", async ({
    page,
  }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- [ ] タスク1\n- [ ] タスク2");
    await setCursorPosition(page, 2, 6);
    await pressTabInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- [ ] タスク1\n  - [ ] タスク2");

    const cursor = await getCursorPosition(page);
    expect(cursor.line).toBe(2);
    expect(cursor.column).toBe(8);
  });

  test("完了タスクリスト項目のマーカー直後にテキストがある状態でタブキーを押すとネストされた完了タスクリストアイテムになること", async ({
    page,
  }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- [ ] aaa\n- [x] bbb\n- [ ] ccc");
    await setCursorPosition(page, 2, 6);
    await pressTabInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- [ ] aaa\n  - [x] bbb\n- [ ] ccc");

    const cursor = await getCursorPosition(page);
    expect(cursor.line).toBe(2);
    expect(cursor.column).toBe(8);
  });
});
