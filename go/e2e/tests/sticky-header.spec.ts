import { expect, test, type Page } from "@playwright/test";

import { signIn } from "../helpers/auth";
import {
  cleanupTestData,
  createTestPage,
  createTestSpace,
  createTestSpaceMember,
  createTestTopic,
  createTestTopicMember,
  createTestUser,
  type TestUser,
} from "../helpers/database";

const VIEWPORTS = [
  {
    label: "md未満",
    width: 375,
    height: 700,
    expectsHeaderBeyondViewport: true,
  },
  { label: "md以上", width: 900, height: 700, expectsHeaderBeyondViewport: false },
] as const;
type StickyViewport = (typeof VIEWPORTS)[number];
const LONG_TITLE = "長".repeat(200);
const LONG_BODY = "本文 ".repeat(600).trim();

// このspecがサインインする閲覧者と、その閲覧者が開くページ。ケースごとにユーザーとスペースを
// 持たせ、afterAllでspecが作ったものを消せるようにする。
interface StickyViewer {
  user: TestUser;
  pagePath: string;
}

let adminViewer: StickyViewer | undefined;
let readerViewer: StickyViewer | undefined;

// scopesはヘッダーが操作領域を持つかどうかを決める。2つのケースの違いはそこだけで、200文字の
// タイトルを含めページの条件は同じにする。
async function createStickyViewer(scopes: string[]): Promise<StickyViewer> {
  const user = await createTestUser();
  const space = await createTestSpace();
  const spaceMemberId = await createTestSpaceMember(space.id, user.id, scopes);
  const topic = await createTestTopic(space.id);
  await createTestTopicMember(space.id, topic.id, spaceMemberId);
  const page = await createTestPage(space.id, topic.id, { title: LONG_TITLE, body: LONG_BODY });

  return { user, pagePath: `/s/${space.identifier}/pages/${page.number}` };
}

function requireViewer(viewer: StickyViewer | undefined, label: string): StickyViewer {
  if (!viewer) {
    throw new Error(`閲覧者 (${label}) が作成されていません`);
  }
  return viewer;
}

test.beforeAll(async () => {
  adminViewer = await createStickyViewer(["space:admin"]);
  readerViewer = await createStickyViewer(["page:read"]);
});

test.afterAll(async () => {
  const userIds = [adminViewer, readerViewer]
    .filter((viewer): viewer is StickyViewer => viewer !== undefined)
    .map((viewer) => viewer.user.id);

  await cleanupTestData(userIds);
});

interface StickyLayout {
  bodyDocumentTop: number;
  headerHeight: number;
  headerViewportBottom: number;
  spacerHeight: number;
}

async function waitForRendering(page: Page): Promise<void> {
  await page.evaluate(
    () =>
      new Promise<void>((resolve) => {
        window.requestAnimationFrame(() => {
          window.requestAnimationFrame(() => resolve());
        });
      }),
  );
}

async function measureStickyLayout(page: Page): Promise<StickyLayout> {
  return page.evaluate(() => {
    const body = document.querySelector<HTMLElement>(".wikino-markdown");
    const header = document.querySelector<HTMLElement>("[data-sticky-header]");
    const spacer = document.querySelector<HTMLElement>("[data-sticky-header-spacer]");
    if (!body || !header || !spacer) {
      throw new Error("スティッキーヘッダーのレイアウトに必要な要素が揃っていません");
    }

    const bodyRect = body.getBoundingClientRect();
    const headerRect = header.getBoundingClientRect();
    return {
      bodyDocumentTop: bodyRect.top + window.scrollY,
      headerHeight: headerRect.height,
      headerViewportBottom: headerRect.bottom,
      spacerHeight: spacer.getBoundingClientRect().height,
    };
  });
}

async function expectStableStickyLayout(page: Page, path: string, viewport: StickyViewport): Promise<void> {
  await page.setViewportSize(viewport);
  await page.goto(path);

  const sentinel = page.locator("[data-sticky-header-sentinel]");
  const header = page.locator("[data-sticky-header]");
  const spacer = page.locator("[data-sticky-header-spacer]");
  await expect(sentinel).toHaveCount(1);
  await expect(header).toHaveCount(1);
  await expect(spacer).toHaveCount(1);

  // 2 frame待って初回のIntersectionObserver通知を走らせる。狭い幅では200文字のタイトルが
  // ビューポート下端を越えるが、sentinelはroot上端より下にあり、固定済みと誤判定してはならない。
  // 広い幅ではmdレイアウトも検証する。
  await waitForRendering(page);
  await expect(header).not.toHaveAttribute("data-stuck", "");
  const expanded = await measureStickyLayout(page);
  if (viewport.expectsHeaderBeyondViewport) {
    expect(expanded.headerViewportBottom).toBeGreaterThan(viewport.height);
  }
  expect(expanded.spacerHeight).toBe(0);

  const sentinelDocumentTop = await sentinel.evaluate(
    (element) => element.getBoundingClientRect().top + window.scrollY,
  );
  await page.evaluate((top) => window.scrollTo(0, top + 10), sentinelDocumentTop);
  await expect
    .poll(() => sentinel.evaluate((element) => element.getBoundingClientRect().bottom))
    .toBeLessThanOrEqual(0);
  await expect(header).toHaveAttribute("data-stuck", "");

  const pinned = await measureStickyLayout(page);
  expect(pinned.headerHeight).toBeLessThan(expanded.headerHeight);
  expect(Math.abs(pinned.bodyDocumentTop - expanded.bodyDocumentTop)).toBeLessThan(1);
  expect(Math.abs(pinned.spacerHeight - (expanded.headerHeight - pinned.headerHeight))).toBeLessThan(1);

  await page.evaluate(() => window.scrollTo(0, 0));
  await expect(header).not.toHaveAttribute("data-stuck", "");

  const restored = await measureStickyLayout(page);
  expect(Math.abs(restored.headerHeight - expanded.headerHeight)).toBeLessThan(1);
  expect(Math.abs(restored.bodyDocumentTop - expanded.bodyDocumentTop)).toBeLessThan(1);
  expect(restored.spacerHeight).toBe(0);
}

test.describe("ページ表示のスティッキーヘッダー", () => {
  for (const viewport of VIEWPORTS) {
    test(`操作ありの長いタイトルでも上端通過時だけ固定し本文位置を保つこと (${viewport.label})`, async ({ page }) => {
      const viewer = requireViewer(adminViewer, "admin");

      await page.context().clearCookies();
      await signIn(page, viewer.user);
      await expectStableStickyLayout(page, viewer.pagePath, viewport);

      await expect(page.locator("#page-actions-dropdown")).toHaveCount(1);
    });
  }

  for (const viewport of VIEWPORTS) {
    test(`操作なしの長いタイトルでも上端通過時だけ固定し本文位置を保つこと (${viewport.label})`, async ({ page }) => {
      const viewer = requireViewer(readerViewer, "reader");

      await page.context().clearCookies();
      await signIn(page, viewer.user);
      await expectStableStickyLayout(page, viewer.pagePath, viewport);

      await expect(page.locator("#page-actions-dropdown")).toHaveCount(0);
    });
  }
});
