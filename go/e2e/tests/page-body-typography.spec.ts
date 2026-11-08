import { test, expect, type Page } from "@playwright/test";
import {
  createTestPage,
  createTestTopic,
  createTestTopicMember,
  loadSharedTestData,
  type TestSpace,
  type TestTopic,
} from "../helpers/database";

// 本文の見た目は文字列の検査では拾えないため、ブラウザが描画した結果 (計算済みのスタイルと矩形) を測る。

const BODY_MARKDOWN = [
  "段落の文章です。",
  "",
  "- 通常の項目。`go/internal/templates/pages/page/show.templ` と `go/internal/templates/pages/page/preview.templ` と `go/web/style.css` を並べて、折り返して複数行になるようにする。",
  "- [ ] 未完了のタスク",
  "- [x] 完了したタスク",
  "- 通常の項目",
  "",
  "1. 番号付きの項目",
  "",
  "- [ ] 空行を入れたタスク",
  "",
  "- 空行を入れた通常の項目",
  "",
  "| 名前 | 説明 | 備考 |",
  "| --- | --- | --- |",
  "| `inline-code-example` と `another-code` | 日本語の長い文章が入るセルです。狭い画面で数文字ごとに折り返さないかを確かめます。 | 備考の文章もそれなりに長く書いておきます。 |",
].join("\n");

// 折り返したインラインコードの背景どうしに空けておく上下の間隔。行の高さがいちばん詰まったテーブルのセル
// (24px) でも、端末の等幅書体の高さの差を吸収して背景が重ならないだけの余裕を求める。
const CODE_BACKGROUND_MIN_GAP_PX = 2;

// 640px未満で、テーブルのセルに付ける最小幅 (8em) を本文の文字サイズ (16px) で換算した値。
const CELL_MIN_WIDTH_PX = 128;

let space: TestSpace;
let topic: TestTopic;
let pagePath: string;

test.beforeAll(async () => {
  const shared = loadSharedTestData();
  space = shared.space;

  topic = await createTestTopic(space.id, { name: "PageBodyTypographyTopic" });
  await createTestTopicMember(space.id, topic.id, shared.spaceMemberId);
  const page_ = await createTestPage(space.id, topic.id, {
    title: "Page Body Typography Page",
    body: BODY_MARKDOWN,
  });
  pagePath = `/s/${space.identifier}/pages/${page_.number}`;
});

interface TextMetrics {
  fontSize: string;
  lineHeight: string;
}

interface BodyMetrics {
  paragraph: TextMetrics;
  unorderedList: TextMetrics;
  orderedList: TextMetrics;
}

async function measureBodyText(page: Page, bodySelector: string): Promise<BodyMetrics> {
  return page.locator(bodySelector).evaluate((body) => {
    const metrics = (selector: string) => {
      const el = body.querySelector(selector);
      if (!el) throw new Error(`本文に ${selector} がありません`);
      const style = getComputedStyle(el);
      return { fontSize: style.fontSize, lineHeight: style.lineHeight };
    };
    return { paragraph: metrics("p"), unorderedList: metrics("ul > li"), orderedList: metrics("ol > li") };
  });
}

interface ListLinePitches {
  // 最初の項目の中で、折り返した行どうしの上端の間隔
  wrapped: number;
  // 最初の項目の最後の行から、次の項目の1行目までの上端の間隔
  nextItem: number;
}

async function measureListLinePitches(page: Page, bodySelector: string): Promise<ListLinePitches> {
  return page
    .locator(`${bodySelector} ul`)
    .first()
    .evaluate((list) => {
      // インラインコードは書体が違い矩形の上端がずれるため、項目の直下の文字だけで行の位置を測る。
      const lineTops = (item: Element) => {
        const tops = [...item.childNodes]
          .filter((node) => node.nodeType === Node.TEXT_NODE && node.textContent?.trim())
          .flatMap((node) => {
            const range = document.createRange();
            range.selectNodeContents(node);
            return [...range.getClientRects()].map((rect) => rect.top);
          });
        return [...new Set(tops.map((top) => Math.round(top)))].sort((a, b) => a - b);
      };

      const [first, second] = list.querySelectorAll(":scope > li");
      if (!first || !second) throw new Error("箇条書きに2つ以上の項目がありません");
      const firstTops = lineTops(first);
      const secondTops = lineTops(second);
      if (firstTops.length < 2) throw new Error("最初の項目が複数行に折り返していません");

      return {
        wrapped: Math.min(...firstTops.slice(1).map((top, i) => top - firstTops[i])),
        nextItem: secondTops[0] - firstTops[firstTops.length - 1],
      };
    });
}

async function openPreview(page: Page): Promise<string> {
  await page.goto(`${pagePath}/edit`);
  await page.waitForSelector(".cm-content");
  await page.locator("#page-edit-tab-preview").click();
  const bodySelector = "#page-edit-preview-content .wikino-markdown";
  await expect(page.locator(`${bodySelector} table`)).toBeVisible({ timeout: 5000 });
  return bodySelector;
}

test.describe("ページ本文の表示", () => {
  test("段落と箇条書きの文字サイズと行の高さが、ページ詳細画面とプレビューで揃うこと", async ({ page }) => {
    await page.goto(pagePath);
    const show = await measureBodyText(page, "article .wikino-markdown");

    const previewSelector = await openPreview(page);
    const preview = await measureBodyText(page, previewSelector);

    for (const metrics of [show, preview]) {
      expect(metrics.unorderedList).toEqual(metrics.paragraph);
      expect(metrics.orderedList).toEqual(metrics.paragraph);
    }
    expect(preview).toEqual(show);
  });

  test("箇条書きの次の項目との間隔が、項目の中で折り返した行の間隔より広いこと", async ({ page }) => {
    // 最初の項目が複数行に折り返す幅にする。
    await page.setViewportSize({ width: 375, height: 800 });
    await page.goto(pagePath);
    const show = await measureListLinePitches(page, "article .wikino-markdown");

    const previewSelector = await openPreview(page);
    const preview = await measureListLinePitches(page, previewSelector);

    for (const pitches of [show, preview]) {
      expect(pitches.wrapped).toBeGreaterThan(0);
      expect(pitches.nextItem).toBeGreaterThan(pitches.wrapped);
    }
  });

  for (const container of ["ul > li", "td"]) {
    test(`複数行に折り返したインラインコードの背景が、上下の行と重ならないこと (${container})`, async ({ page }) => {
      // コードが1行に収まらず、行をまたいで並ぶ幅にする。
      await page.setViewportSize({ width: 375, height: 800 });
      await page.goto(pagePath);

      const lines = await page
        .locator(`article .wikino-markdown ${container}`)
        .first()
        .evaluate((el) =>
          [...el.querySelectorAll("code")].flatMap((code) =>
            [...code.getClientRects()].map((rect) => ({ top: rect.top, bottom: rect.bottom })),
          ),
        );

      const distinctTops = [...new Set(lines.map((line) => Math.round(line.top)))];
      expect(distinctTops.length).toBeGreaterThan(1);

      // 別の行にある背景の矩形どうしの間に、上下の間隔が空いていること。
      for (const upper of lines) {
        for (const lower of lines) {
          if (Math.round(lower.top) <= Math.round(upper.top)) continue;
          expect(lower.top - upper.bottom).toBeGreaterThanOrEqual(CODE_BACKGROUND_MIN_GAP_PX);
        }
      }
    });
  }

  test("タスク記法の項目だけ箇条書きの点が消えること", async ({ page }) => {
    await page.goto(pagePath);

    const items = await page.locator("article .wikino-markdown ul > li").evaluateAll((lis) =>
      lis.map((li) => ({
        isTask: li.querySelector(":scope > input, :scope > p:first-child > input") !== null,
        listStyleType: getComputedStyle(li).listStyleType,
      })),
    );

    // 空行を入れた箇条書きのタスクを含め、タスクの項目と通常の項目の両方があること。
    expect(items.filter((item) => item.isTask)).toHaveLength(3);
    expect(items.filter((item) => !item.isTask)).toHaveLength(3);

    for (const item of items) {
      expect(item.listStyleType).toBe(item.isTask ? "none" : "disc");
    }
  });

  test("640px未満では、テーブルのセルが最小幅を下回らず横スクロールになること", async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 800 });
    await page.goto(pagePath);

    const scroll = page.locator("article .wikino-markdown .wikino-markdown-table-scroll");
    await expect(scroll).toBeVisible();

    const layout = await scroll.evaluate((wrapper) => {
      const table = wrapper.querySelector("table");
      if (!table) throw new Error("スクロールの領域にテーブルがありません");
      return {
        scrollWidth: wrapper.clientWidth,
        tableWidth: table.getBoundingClientRect().width,
        cellWidths: [...table.querySelectorAll("th, td")].map((cell) => cell.getBoundingClientRect().width),
      };
    });

    for (const width of layout.cellWidths) {
      expect(width).toBeGreaterThanOrEqual(CELL_MIN_WIDTH_PX - 0.5);
    }
    expect(layout.scrollWidth).toBeLessThan(layout.tableWidth);
  });

  test("640px以上では、テーブルのセルに最小幅が付かないこと", async ({ page }) => {
    await page.setViewportSize({ width: 1000, height: 800 });
    await page.goto(pagePath);

    const minWidths = await page
      .locator("article .wikino-markdown table")
      .locator("th, td")
      .evaluateAll((cells) => cells.map((cell) => getComputedStyle(cell).minWidth));

    expect(minWidths.length).toBeGreaterThan(0);
    for (const minWidth of minWidths) {
      expect(["auto", "0px"]).toContain(minWidth);
    }
  });
});

async function measureShowFonts(page: Page): Promise<{ body: string; code: string }> {
  await page.goto(pagePath);
  return page.evaluate(() => {
    const code = document.querySelector("article .wikino-markdown code");
    if (!code) throw new Error("本文にインラインコードがありません");
    return { body: getComputedStyle(document.body).fontFamily, code: getComputedStyle(code).fontFamily };
  });
}

test.describe("アプリ全体の書体", () => {
  test("UIが欧文書体と和文書体を明示した並びになり、インラインコードは等幅のままであること", async ({ page }) => {
    const showFonts = await measureShowFonts(page);

    // 欧文書体を先頭に置き、Windowsの日本語環境でYu Gothic UIを選ばせるsystem-uiを並べない。
    expect(showFonts.body.startsWith('"Helvetica Neue"')).toBe(true);
    expect(showFonts.body).toContain('"Hiragino Kaku Gothic ProN"');
    expect(showFonts.body).toContain('"BIZ UDPGothic"');
    expect(showFonts.body).not.toContain("system-ui");

    expect(showFonts.code).not.toBe(showFonts.body);
    expect(showFonts.code).toContain("monospace");
  });

  test("エディタの英数字が本文のコードと同じ等幅書体になり、和文が本文と同じ書体になること", async ({ page }) => {
    const showFonts = await measureShowFonts(page);

    await page.goto(`${pagePath}/edit`);
    const editorFonts = await page.locator(".cm-scroller").evaluate((scroller) =>
      getComputedStyle(scroller)
        .fontFamily.split(",")
        .map((family) => family.trim()),
    );

    // 英数字は本文のコードの等幅書体の並びから選ばれる。
    expect(editorFonts[0]).toBe(showFonts.code.split(",")[0].trim());

    // 総称のmonospaceより前に和文書体を置き、和文にブラウザの既定の等幅書体が使われないようにする。
    const monospaceIndex = editorFonts.indexOf("monospace");
    expect(monospaceIndex).toBeGreaterThan(0);
    for (const family of ['"Hiragino Kaku Gothic ProN"', '"BIZ UDPGothic"']) {
      expect(showFonts.body).toContain(family);
      const index = editorFonts.indexOf(family);
      expect(index).toBeGreaterThan(0);
      expect(index).toBeLessThan(monospaceIndex);
    }
  });

  // エディタの文字サイズは768pxを境に切り替わる (768px未満で16px、768px以上で14px)。
  for (const { label, width, fontSize } of [
    { label: "768px未満", width: 375, fontSize: 16 },
    { label: "768px以上", width: 1000, fontSize: 14 },
  ]) {
    test(`${label}の幅で、エディタの行の高さが文字サイズの1.625倍になること`, async ({ page }) => {
      await page.setViewportSize({ width, height: 800 });
      await page.goto(`${pagePath}/edit`);
      await page.waitForSelector(".cm-line");

      const metrics = await page
        .locator(".cm-line")
        .first()
        .evaluate((line) => {
          const style = getComputedStyle(line);
          return { fontSize: parseFloat(style.fontSize), lineHeight: parseFloat(style.lineHeight) };
        });

      expect(metrics.fontSize).toBe(fontSize);
      expect(metrics.lineHeight).toBeCloseTo(fontSize * 1.625, 1);
    });
  }
});
