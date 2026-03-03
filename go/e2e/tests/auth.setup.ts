import { test as setup } from "@playwright/test";
import {
  createTestUser,
  createTestSpace,
  createTestSpaceMember,
  createTestFeatureFlag,
  saveSharedTestData,
} from "../helpers/database";

setup("authenticate", async ({ page }) => {
  const user = await createTestUser();
  const space = await createTestSpace();
  const spaceMemberId = await createTestSpaceMember(space.id, user.id);
  await createTestFeatureFlag(user.id, "go_page_edit");

  await page.goto("/sign_in");
  await page.locator('input[name="email"]').fill(user.email);
  await page.locator('input[name="password"]').fill(user.password);
  const responsePromise = page.waitForResponse(
    (resp) => resp.url().includes("/sign_in") && resp.request().method() === "POST",
  );
  await page.locator('button[type="submit"]').click();
  const response = await responsePromise;
  if (response.status() !== 302) {
    throw new Error(`Sign-in failed with status ${response.status()}`);
  }

  await page.context().storageState({ path: "playwright/.auth/user.json" });

  saveSharedTestData({ user, space, spaceMemberId });
});
