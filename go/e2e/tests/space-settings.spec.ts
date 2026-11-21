import { expect, test } from "@playwright/test";
import { signIn } from "../helpers/auth";
import { cleanupTestData, createTestSpace, createTestSpaceMember, createTestUser, query } from "../helpers/database";

test.use({ storageState: { cookies: [], origins: [] } });

test("フラグを持たない管理者がスペースの設定画面を開いてログアウトできること", async ({ page }) => {
  // ログアウトで消すセッションはこのテスト専用に作る。
  const user = await createTestUser();
  try {
    const space = await createTestSpace();
    await createTestSpaceMember(space.id, user.id);
    const flags = await query("SELECT name FROM feature_flags WHERE user_id = $1", [user.id]);
    expect(flags.rows).toEqual([]);
    await signIn(page, user);

    for (const setting of ["general", "attachments", "deletion/new"]) {
      const path = `/s/${space.identifier}/settings/${setting}`;
      const response = await page.goto(path);
      expect(response?.status(), path).toBe(200);
      expect(new URL(page.url()).pathname).toBe(path);
    }

    await page.goto("/settings");
    const form = page.locator('form[action="/user_session"]');
    const token = await form.locator('input[name="csrf_token"]').inputValue();
    const method = await form.locator('input[name="_method"]').inputValue();
    expect(token).not.toBe("");

    const rejected = await page.request.post("/user_session", { form: { _method: method }, maxRedirects: 0 });
    expect(rejected.status()).toBe(403);
    const stillSignedIn = await page.request.get("/home", { maxRedirects: 0 });
    expect(stillSignedIn.status()).toBe(200);

    const logout = await page.request.post("/user_session", {
      form: { _method: method, csrf_token: token },
      maxRedirects: 0,
    });
    expect(logout.status()).toBe(302);
    expect(logout.headers()["location"]).toBe("/");
    const sessions = await query("SELECT id FROM user_sessions WHERE user_id = $1", [user.id]);
    expect(sessions.rows).toEqual([]);
    const signedOut = await page.request.get("/home", { maxRedirects: 0 });
    expect(signedOut.status()).toBe(302);
    expect(new URL(signedOut.headers()["location"], page.url()).pathname).toBe("/sign_in");
  } finally {
    await cleanupTestData([user.id]);
  }
});
