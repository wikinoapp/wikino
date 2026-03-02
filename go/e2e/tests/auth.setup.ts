import { test as setup } from "@playwright/test";
import { createTestUser, createTestSpace, createTestSpaceMember, saveSharedTestData } from "../helpers/database";

setup("authenticate", async ({ page }) => {
  const user = await createTestUser();
  const space = await createTestSpace();
  const spaceMemberId = await createTestSpaceMember(space.id, user.id);

  await page.goto("/sign_in");
  await page.locator('input[name="email"]').fill(user.email);
  await page.locator('input[name="password"]').fill(user.password);
  await page.locator('button[type="submit"]').click();
  await page.waitForURL((url) => !url.pathname.includes("/sign_in"));

  await page.context().storageState({ path: "playwright/.auth/user.json" });

  saveSharedTestData({ user, space, spaceMemberId });
});
