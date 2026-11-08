import { highlightTree } from "@lezer/highlight";

import { markdownHighlighter } from "./markdown-highlighter";
import { markdownLanguage } from "./markdown-language";

// 差分の表 (internal/templates/components/diff.templ) の本文のセルに、エディタと同じ記法の色を当てる。
// 差分の計算と表の描画はサーバーが行い、ここでは埋めた新旧の本文を解析して、セルの文字を記法の
// クラス名を付けたspanに組み替えるだけにする。色付けはプログレッシブエンハンスメントで、失敗しても
// サーバーが描画した色の無い差分がそのまま残る。
//
// 差分は編集提案の画面ではサーバーで描画され、編集履歴のモーダルではhtmxが差し込むため、
// web/markdown-table.tsと同じく、読み込み時とhtmx:after:settleの2つを起点にする。

const DIFF_SELECTOR = "[data-markdown-diff]";

const CELL_SELECTOR = "[data-markdown-diff-line]";

// 色を当て終えたまとまりの目印。同じフラグメントが2回落ち着いても、二重に組み替えない。
const HIGHLIGHTED_ATTRIBUTE = "data-markdown-diff-highlighted";

interface DiffSources {
  old: string;
  new: string;
}

// Segmentは1行の中の、同じクラス名が付く文字の並び。classNameが空の部分は記法の色が付かない。
interface Segment {
  text: string;
  className: string;
}

export function initializeMarkdownDiffs(): void {
  document.addEventListener("htmx:after:settle", handleAfterSettle);

  highlightDiffs(document.body);
}

function handleAfterSettle(event: Event): void {
  const settled = event.target;
  if (!(settled instanceof HTMLElement)) return;

  highlightDiffs(settled);
}

// highlightDiffsはroot自身とその子孫の差分に色を当てる。1つのまとまりで失敗しても、ほかのまとまりと
// 呼び出し元の初期化を巻き込まないよう、まとまりごとに例外を握る。
export function highlightDiffs(root: HTMLElement): void {
  const diffs = root.matches(DIFF_SELECTOR)
    ? [root, ...root.querySelectorAll<HTMLElement>(DIFF_SELECTOR)]
    : Array.from(root.querySelectorAll<HTMLElement>(DIFF_SELECTOR));

  for (const diff of diffs) {
    if (diff.hasAttribute(HIGHLIGHTED_ATTRIBUTE)) continue;
    diff.setAttribute(HIGHLIGHTED_ATTRIBUTE, "");

    try {
      highlightDiff(diff);
    } catch (error) {
      console.error("差分の色付けに失敗しました", error);
    }
  }
}

function highlightDiff(diff: HTMLElement): void {
  const sources = readSources(diff);
  if (sources === null) return;

  // 解析は、その本文の行を使うセルがあるときだけ行う (削除だけ・追加だけの差分では片方が要らない)。
  const lines: Partial<Record<keyof DiffSources, Segment[][]>> = {};

  for (const cell of diff.querySelectorAll<HTMLElement>(CELL_SELECTOR)) {
    const source = cell.dataset.markdownDiffSource;
    if (source !== "old" && source !== "new") continue;

    const lineNumber = Number(cell.dataset.markdownDiffLine);
    if (!Number.isInteger(lineNumber) || lineNumber < 1) continue;

    lines[source] ??= highlightLines(sources[source]);
    const segments = lines[source][lineNumber - 1];
    if (segments === undefined) continue;

    // セルの文字と解析した行の文字が食い違うときは、行を取り違えているため色を当てない。
    if (cell.textContent !== segments.map((segment) => segment.text).join("")) continue;

    // 本文は利用者の入力のため、innerHTMLを使わず、文字のノードと要素の作成だけで組み立てる。
    cell.replaceChildren(...segments.map(renderSegment));
  }
}

// readSourcesはまとまりの直下に埋めた新旧の本文を読む。形が合わないときはnullを返す。
function readSources(diff: HTMLElement): DiffSources | null {
  const script = diff.querySelector<HTMLScriptElement>(':scope > script[type="application/json"]');
  if (script === null) return null;

  const data: unknown = JSON.parse(script.textContent ?? "");
  if (typeof data !== "object" || data === null) return null;

  const { old: oldSource, new: newSource } = data as Record<string, unknown>;
  if (typeof oldSource !== "string" || typeof newSource !== "string") return null;

  return { old: oldSource, new: newSource };
}

// highlightLinesは本文を解析し、行ごとのSegmentの並びを返す。返す配列のi番目は本文のi+1行目である。
// 差分の行は改行を含まないため、改行の位置で行を分ける。
export function highlightLines(source: string): Segment[][] {
  const tree = markdownLanguage.parser.parse(source);
  let line: Segment[] = [];
  const lines: Segment[][] = [line];
  let pos = 0;
  // 次の改行位置を範囲間で使い回し、同じ行末までの重複した探索を避ける。
  let newline = source.indexOf("\n");

  const push = (to: number, className: string): void => {
    while (pos < to) {
      const end = newline === -1 || newline >= to ? to : newline;
      if (end > pos) line.push({ text: source.slice(pos, end), className });
      if (end === newline) {
        line = [];
        lines.push(line);
        pos = end + 1;
        newline = source.indexOf("\n", pos);
      } else {
        pos = end;
      }
    }
  };

  // highlightTreeは色の付く範囲を前から順に、重ならないように渡す。範囲の間の文字は色を付けずに置く。
  highlightTree(tree, markdownHighlighter, (from, to, classes) => {
    push(from, "");
    push(to, classes);
  });
  push(source.length, "");

  return lines;
}

function renderSegment(segment: Segment): Node {
  if (segment.className === "") return document.createTextNode(segment.text);

  const span = document.createElement("span");
  span.className = segment.className;
  span.textContent = segment.text;
  return span;
}
