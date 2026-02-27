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
import {
  clearEditor,
  setEditorContent,
  getEditorContent,
  pressEnterInEditor,
  setCursorPosition,
  getCursorPosition,
  fillInEditor,
} from "../helpers/editor";

let user: TestUser;
let space: TestSpace;
let page_: TestPage;

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

async function visitPageEditor(pwPage: import("@playwright/test").Page) {
  const topic = await createTestTopic(space.id);
  const spaceMemberResult = await import("../helpers/database").then((db) =>
    db.query("SELECT id FROM space_members WHERE space_id = $1 AND user_id = $2 LIMIT 1", [space.id, user.id]),
  );
  const spaceMemberId = spaceMemberResult.rows[0].id;
  await createTestTopicMember(space.id, topic.id, spaceMemberId);
  page_ = await createTestPage(space.id, topic.id);

  await signIn(pwPage, user);
  await pwPage.goto(`/go/s/${space.identifier}/pages/${page_.number}/edit`);
  await pwPage.waitForSelector(".cm-content");
}

test.describe("Enter押下時のリスト継続機能", () => {
  test("順序なしリストの末尾でEnterを押すと次行にリストマーカーが挿入されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- aaa");
    await setCursorPosition(page, 1, 5);
    await pressEnterInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- aaa\n- ");

    const cursor = await getCursorPosition(page);
    expect(cursor.line).toBe(2);
    expect(cursor.column).toBe(2);
  });

  test("順序付きリストの末尾でEnterを押すと番号がインクリメントされたマーカーが挿入されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "1. aaa");
    await setCursorPosition(page, 1, 6);
    await pressEnterInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("1. aaa\n2. ");

    const cursor = await getCursorPosition(page);
    expect(cursor.line).toBe(2);
    expect(cursor.column).toBe(3);
  });

  test("タスクリストの末尾でEnterを押すと未完了のタスクマーカーが挿入されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- [ ] タスク1");
    await setCursorPosition(page, 1, 10);
    await pressEnterInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- [ ] タスク1\n- [ ] ");

    const cursor = await getCursorPosition(page);
    expect(cursor.line).toBe(2);
    expect(cursor.column).toBe(6);
  });

  test("完了タスクリストの末尾でEnterを押すと未完了のタスクマーカーが挿入されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- [x] 完了タスク");
    await setCursorPosition(page, 1, 11);
    await pressEnterInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- [x] 完了タスク\n- [ ] ");

    const cursor = await getCursorPosition(page);
    expect(cursor.line).toBe(2);
    expect(cursor.column).toBe(6);
  });

  test("空の順序なしリスト項目でEnterを押すとリストマーカーが削除されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- aaa\n- ");
    await setCursorPosition(page, 2, 2);
    await pressEnterInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- aaa\n");
  });

  test("空の順序付きリスト項目でEnterを押すとリストマーカーが削除されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "1. aaa\n2. ");
    await setCursorPosition(page, 2, 3);
    await pressEnterInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("1. aaa\n");
  });

  test("空のタスクリスト項目でEnterを押すとリストマーカーが削除されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- [ ] タスク1\n- [ ] ");
    await setCursorPosition(page, 2, 6);
    await pressEnterInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- [ ] タスク1\n");
  });

  test("空のインデント付きリスト項目でEnterを押すとインデントが1段減ること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- aaa\n  - ");
    await setCursorPosition(page, 2, 4);
    await pressEnterInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- aaa\n- ");
  });

  test("空のインデント付きタスクリスト項目でEnterを押すとインデントが1段減ること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- [ ] aaa\n  - [ ] ");
    await setCursorPosition(page, 2, 8);
    await pressEnterInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- [ ] aaa\n- [ ] ");
  });

  test("インデント付きリストの末尾でEnterを押すとインデントが維持されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- aaa\n  - bbb");
    await setCursorPosition(page, 2, 7);
    await pressEnterInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- aaa\n  - bbb\n  - ");

    const cursor = await getCursorPosition(page);
    expect(cursor.line).toBe(3);
    expect(cursor.column).toBe(4);
  });

  test("リスト項目のテキスト途中でEnterを押すとカーソル以降のテキストが次行に移動すること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- abcdef");
    await setCursorPosition(page, 1, 5);
    await pressEnterInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- abc\n- def");

    const cursor = await getCursorPosition(page);
    expect(cursor.line).toBe(2);
    expect(cursor.column).toBe(2);
  });

  test("タスクリスト項目のテキスト途中でEnterを押すとカーソル以降のテキストが次行に移動すること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- [ ] abcdef");
    await setCursorPosition(page, 1, 9);
    await pressEnterInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- [ ] abc\n- [ ] def");

    const cursor = await getCursorPosition(page);
    expect(cursor.line).toBe(2);
    expect(cursor.column).toBe(6);
  });

  test("順序付きリスト項目のテキスト途中でEnterを押すとカーソル以降のテキストが次行に移動すること", async ({
    page,
  }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "1. abcdef");
    await setCursorPosition(page, 1, 6);
    await pressEnterInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("1. abc\n2. def");

    const cursor = await getCursorPosition(page);
    expect(cursor.line).toBe(2);
    expect(cursor.column).toBe(3);
  });

  test("リスト以外の行でEnterを押すと通常の改行が挿入されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await fillInEditor(page, "通常のテキスト");
    await pressEnterInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("通常のテキスト\n");
  });

  test("複数のリスト項目が続く場合にEnterを押すとマーカーが継続されること", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "- aaa\n- bbb");
    await setCursorPosition(page, 2, 5);
    await pressEnterInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("- aaa\n- bbb\n- ");

    const cursor = await getCursorPosition(page);
    expect(cursor.line).toBe(3);
    expect(cursor.column).toBe(2);
  });

  test("順序付きリストで連番のインクリメントが正しいこと", async ({ page }) => {
    await visitPageEditor(page);
    await clearEditor(page);
    await setEditorContent(page, "1. aaa\n2. bbb\n3. ccc");
    await setCursorPosition(page, 3, 6);
    await pressEnterInEditor(page);

    const content = await getEditorContent(page);
    expect(content).toBe("1. aaa\n2. bbb\n3. ccc\n4. ");

    const cursor = await getCursorPosition(page);
    expect(cursor.line).toBe(4);
    expect(cursor.column).toBe(3);
  });
});
