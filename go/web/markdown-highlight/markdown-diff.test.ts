import { beforeEach, describe, expect, it, vi } from "vitest";
import { highlightTree } from "@lezer/highlight";

import { highlightDiffs, highlightLines, initializeMarkdownDiffs } from "./markdown-diff";
import { markdownHighlighter } from "./markdown-highlighter";
import { markdownLanguage } from "./markdown-language";

interface Row {
  source: "old" | "new";
  line: number;
  text: string;
}

// renderDiffは、サーバーが描画する差分 (internal/templates/components/diff.templ) と同じ形の要素を作る。
// 本文はJSONのまま埋め、セルには旧・新のどちらの何行目かを持たせる。
function renderDiff(sources: string, rows: Row[]): HTMLElement {
  const diff = document.createElement("div");
  diff.setAttribute("data-markdown-diff", "");

  const script = document.createElement("script");
  script.type = "application/json";
  script.textContent = sources;
  diff.appendChild(script);

  const tbody = document.createElement("tbody");
  for (const row of rows) {
    const tr = document.createElement("tr");
    const td = document.createElement("td");
    td.dataset.markdownDiffSource = row.source;
    td.dataset.markdownDiffLine = String(row.line);
    td.textContent = row.text;
    tr.appendChild(td);
    tbody.appendChild(tr);
  }
  const table = document.createElement("table");
  table.appendChild(tbody);
  diff.appendChild(table);

  document.body.appendChild(diff);
  return diff;
}

function cells(diff: HTMLElement): HTMLElement[] {
  return Array.from(diff.querySelectorAll<HTMLElement>("[data-markdown-diff-line]"));
}

function firstCell(diff: HTMLElement): HTMLElement {
  const cell = diff.querySelector<HTMLElement>("[data-markdown-diff-line]");
  if (cell === null) throw new Error("本文のセルが無い");
  return cell;
}

beforeEach(() => {
  document.body.replaceChildren();
});

describe("highlightLines", () => {
  it("本文を行ごとに分け、記法の範囲にクラス名を付ける", () => {
    expect(highlightLines("# 見出し\n本文の**太字**\n")).toEqual([
      [
        { text: "#", className: "wikino-md-heading wikino-md-mark" },
        { text: " 見出し", className: "wikino-md-heading" },
      ],
      [
        { text: "本文の", className: "" },
        { text: "**", className: "wikino-md-strong wikino-md-mark" },
        { text: "太字", className: "wikino-md-strong" },
        { text: "**", className: "wikino-md-strong wikino-md-mark" },
      ],
      [],
    ]);
  });

  it("複数行にまたがる記法 (コードブロック) を行ごとに分ける", () => {
    const lines = highlightLines("```\n[[コード]]\n```\n");

    expect(lines[1]).toEqual([{ text: "[[コード]]", className: "wikino-md-code" }]);
  });

  it("多数の記法を含む長い1行で、同じ行末まで改行を探し直さない", () => {
    // 計算量の退行を検出するため、長い1行で重複走査による差を大きくする。
    const source = "*x* ".repeat(99999);
    const fastest = (run: () => unknown): number => {
      let time = Infinity;
      for (let i = 0; i < 3; i++) {
        const start = performance.now();
        run();
        time = Math.min(time, performance.now() - start);
      }
      return time;
    };
    const collectRanges = () => {
      const ranges: [number, number, string][] = [];
      highlightTree(markdownLanguage.parser.parse(source), markdownHighlighter, (from, to, classes) => {
        ranges.push([from, to, classes]);
      });
      return ranges;
    };
    // 初回の最適化を済ませ、解析と色の範囲の取得を基準に、行分割の重複走査による増加を調べる。
    // 短い時間の比は揺れやすいため、上限は150msより短くしない。
    highlightLines(source);
    const highlightTime = fastest(collectRanges);
    expect(fastest(() => highlightLines(source))).toBeLessThan(Math.max(highlightTime * 2, 150));
  }, 15_000);
});

describe("highlightDiffs", () => {
  it("新旧の本文の行番号に従って、本文のセルに記法の色を当てる", () => {
    const diff = renderDiff(JSON.stringify({ old: "前\n*旧*\n", new: "前\n[[新]]\n" }), [
      { source: "new", line: 1, text: "前" },
      { source: "old", line: 2, text: "*旧*" },
      { source: "new", line: 2, text: "[[新]]" },
    ]);

    highlightDiffs(document.body);

    const [equal, deleted, inserted] = cells(diff);
    if (equal === undefined || deleted === undefined || inserted === undefined) throw new Error("本文のセルが足りない");
    expect(equal.innerHTML).toBe("前");
    expect(deleted.querySelector(".wikino-md-emphasis:not(.wikino-md-mark)")?.textContent).toBe("旧");
    expect(inserted.querySelector(".wikino-md-wikilink:not(.wikino-md-mark)")?.textContent).toBe("新");
    expect(inserted.textContent).toBe("[[新]]");
  });

  it("本文に書いたHTMLを要素にせず、文字のまま残す", () => {
    const text = "<b>太字</b> と <img src=x onerror=alert(1)>";
    const diff = renderDiff(JSON.stringify({ old: "", new: `${text}\n` }), [{ source: "new", line: 1, text }]);

    highlightDiffs(document.body);

    const cell = firstCell(diff);
    expect(cell.textContent).toBe(text);
    expect(cell.querySelector("b, img")).toBeNull();
  });

  it("空行の追加・削除の後ろの変更行に、本文の行番号に従って色を付ける", () => {
    for (const insertedBlank of [true, false]) {
      const old = insertedBlank ? "# 見出し\n同じ行\n*旧*\n" : "# 見出し\n\n同じ行\n*旧*\n";
      const changed = insertedBlank ? "# 見出し\n\n同じ行\n*新*\n" : "# 見出し\n同じ行\n*新*\n";
      const diff = renderDiff(JSON.stringify({ old, new: changed }), [
        { source: "old", line: insertedBlank ? 3 : 4, text: "*旧*" },
        { source: "new", line: insertedBlank ? 4 : 3, text: "*新*" },
      ]);

      highlightDiffs(diff);

      for (const cell of cells(diff)) {
        expect(cell.querySelector(".wikino-md-emphasis:not(.wikino-md-mark)")).not.toBeNull();
      }
    }
  });

  it("セルの文字と解析した行の文字が食い違うときは、そのセルに色を当てない", () => {
    const diff = renderDiff(JSON.stringify({ old: "", new: "**一行目**\n**二行目**\n" }), [
      { source: "new", line: 1, text: "**二行目**" },
      { source: "new", line: 3, text: "**無い行**" },
    ]);

    highlightDiffs(document.body);

    for (const cell of cells(diff)) {
      expect(cell.children).toHaveLength(0);
    }
  });

  it("同じまとまりに2回色を当てても、組み替えは1回だけにする", () => {
    const diff = renderDiff(JSON.stringify({ old: "", new: "**太字**\n" }), [
      { source: "new", line: 1, text: "**太字**" },
    ]);

    highlightDiffs(document.body);
    const cell = firstCell(diff);
    const first = cell.firstChild;
    highlightDiffs(document.body);

    expect(cell.firstChild).toBe(first);
    expect(cell.querySelectorAll("span")).toHaveLength(3);
  });

  it("埋めた本文が読めないまとまりは色を当てず、ほかのまとまりの色付けを続ける", () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const broken = renderDiff("{", [{ source: "new", line: 1, text: "**太字**" }]);
    const malformed = renderDiff(JSON.stringify({ old: 1, new: "**太字**\n" }), [
      { source: "new", line: 1, text: "**太字**" },
    ]);
    const valid = renderDiff(JSON.stringify({ old: "", new: "**太字**\n" }), [
      { source: "new", line: 1, text: "**太字**" },
    ]);

    expect(() => highlightDiffs(document.body)).not.toThrow();

    expect(firstCell(broken).children).toHaveLength(0);
    expect(firstCell(malformed).children).toHaveLength(0);
    expect(firstCell(valid).querySelector(".wikino-md-strong")).not.toBeNull();
    errorSpy.mockRestore();
  });

  it("目印の無い差分 (本文を埋めない差分) には触れない", () => {
    const diff = renderDiff(JSON.stringify({ old: "", new: "**太字**\n" }), [
      { source: "new", line: 1, text: "**太字**" },
    ]);
    diff.removeAttribute("data-markdown-diff");

    highlightDiffs(document.body);

    expect(firstCell(diff).children).toHaveLength(0);
  });
});

describe("initializeMarkdownDiffs", () => {
  it("読み込み時の差分と、htmxが差し込んで落ち着いた差分に色を当てる", () => {
    const initial = renderDiff(JSON.stringify({ old: "", new: "**太字**\n" }), [
      { source: "new", line: 1, text: "**太字**" },
    ]);

    initializeMarkdownDiffs();
    expect(firstCell(initial).querySelector(".wikino-md-strong")).not.toBeNull();

    const container = document.createElement("div");
    document.body.appendChild(container);
    const swapped = renderDiff(JSON.stringify({ old: "", new: "`コード`\n" }), [
      { source: "new", line: 1, text: "`コード`" },
    ]);
    container.appendChild(swapped);
    container.dispatchEvent(new Event("htmx:after:settle", { bubbles: true }));

    expect(firstCell(swapped).querySelector(".wikino-md-code")).not.toBeNull();
  });
});
