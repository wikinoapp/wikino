import { commonmarkLanguage } from "@codemirror/lang-markdown";
import { Language } from "@codemirror/language";
import { styleTags, Tag, tags } from "@lezer/highlight";
import {
  Autolink,
  type BlockContext,
  type BlockParser,
  type Element,
  type InlineContext,
  type Line,
  type MarkdownConfig,
  type MarkdownParser,
  Strikethrough,
  Table,
  TaskList,
} from "@lezer/markdown";

import { htmlBlockCodeRanges, markdownCodeRanges, type Range } from "./markdown-code-ranges";

const OPEN_BRACKET = 91; // "["
const CLOSE_BRACKET = 93; // "]"
const OPEN_PAREN = 40; // "("
const LESS_THAN = 60; // "<"

interface Boundary {
  from: number;
  to: number;
  closing: boolean;
}

// findBoundariesは、Wikiリンクの候補が越えられない位置 (改行・コード) と閉じ記号を、位置の順に並べる。
function findBoundaries(text: string, codeRanges: readonly Range[]): Boundary[] {
  const boundaries: Boundary[] = [];
  let codeIndex = 0;
  for (let pos = 0; pos < text.length; pos++) {
    const code = codeRanges[codeIndex];
    if (code && pos >= code.from) {
      boundaries.push({ ...code, closing: false });
      pos = code.to - 1;
      codeIndex++;
    } else if (text.charCodeAt(pos) === 10) {
      boundaries.push({ from: pos, to: pos + 1, closing: false });
    } else if (text.charCodeAt(pos) === CLOSE_BRACKET && text.charCodeAt(pos + 1) === CLOSE_BRACKET) {
      boundaries.push({ from: pos, to: pos + 2, closing: true });
      pos++;
    }
  }
  return boundaries;
}

// 改行・コード・閉じ記号の位置をインライン本文ごとに1度だけ走査する。閉じ記号が無い本文でも、
// 開始括弧ごとに末尾まで探し直さない。キーはLezerの解析コンテキストなので、次の編集には持ち越さない。
// Lezerはインラインの解析器を同じコンテキストの中で位置の昇順に呼ぶため、次に調べる境界 (index) は
// 戻さずに進めるだけでよい。
const wikilinkBoundaries = new WeakMap<InlineContext, { boundaries: Boundary[]; index: number }>();

function nextBoundary(cx: InlineContext, pos: number): Boundary | undefined {
  let scan = wikilinkBoundaries.get(cx);
  if (!scan) {
    scan = { boundaries: findBoundaries(cx.text, markdownCodeRanges(cx.text)), index: 0 };
    wikilinkBoundaries.set(cx, scan);
  }
  const start = pos - cx.offset;
  let boundary = scan.boundaries[scan.index];
  while (boundary && boundary.to <= start) boundary = scan.boundaries[++scan.index];
  return boundary;
}

// htmlBlockWikilinksは、HTMLブロックの本文の中のWikiリンクの範囲を返す。範囲の決め方はインライン本文と
// 同じで、タグ・コメントとコードの中を除く。HTMLブロックの中ではMarkdownのコード・エスケープは働かない。
function htmlBlockWikilinks(text: string): Range[] {
  if (!text.includes("[[")) return [];
  const boundaries = findBoundaries(text, htmlBlockCodeRanges(text));
  const links: Range[] = [];
  let index = 0;
  for (let pos = text.indexOf("[["); pos >= 0; pos = text.indexOf("[[", pos + 1)) {
    let boundary = boundaries[index];
    while (boundary && boundary.to <= pos) boundary = boundaries[++index];
    if (!boundary?.closing || text.slice(pos + 2, boundary.from).trim() === "") continue;
    links.push({ from: pos, to: boundary.to });
    pos = boundary.to - 1;
  }
  return links;
}

function wikilinkElement(cx: BlockContext | InlineContext, from: number, to: number): Element {
  return cx.elt("Wikilink", from, to, [cx.elt("WikilinkMark", from, from + 2), cx.elt("WikilinkMark", to - 2, to)]);
}

const EMPTY_LINE = /^[ \t]*$/;
// LezerのHTMLBlockと同じ、HTMLブロックの始まりと終わりの条件と、ブロックのノードの名前。CommonMarkの
// 7種類のHTMLブロックのうち、Wikiリンクを書きうる1・2・6・7番目を持つ。処理命令などの残りの種類は
// 始まりの文字が重ならないため、この解析器が読まなければ、Lezerの標準の解析がそのまま読む。
const HTML_BLOCK_STYLES: readonly { start: RegExp; end: RegExp; node: string }[] = [
  { start: /^<(?:script|pre|style)(?:\s|>|$)/i, end: /<\/(?:script|pre|style)>/i, node: "HTMLBlock" },
  { start: /^\s*<!--/, end: /-->/, node: "CommentBlock" },
  {
    start:
      /^\s*<\/?(?:address|article|aside|base|basefont|blockquote|body|caption|center|col|colgroup|dd|details|dialog|dir|div|dl|dt|fieldset|figcaption|figure|footer|form|frame|frameset|h1|h2|h3|h4|h5|h6|head|header|hr|html|iframe|legend|li|link|main|menu|menuitem|nav|noframes|ol|optgroup|option|p|param|section|source|summary|table|tbody|td|tfoot|th|thead|title|tr|track|ul)(?:\s|\/?>|$)/i,
    end: EMPTY_LINE,
    node: "HTMLBlock",
  },
  {
    start:
      /^\s*(?:<\/[a-z][\w-]*\s*>|<[a-z][\w-]*(\s+[a-z:_][\w-.]*(?:\s*=\s*(?:[^\s"'=<>`]+|'[^']*'|"[^"]*"))?)*\s*>)\s*$/i,
    end: EMPTY_LINE,
    node: "HTMLBlock",
  },
];

// lineDepthは、その行で続いている入れ物 (引用・リストの項目など) の数を返す。公開の型には無い値だが、
// LezerのHTMLBlockと同じく、入れ物が途中で終わった行でHTMLブロックを終えるために読む。値が読めなければ
// undefinedを返し、HTMLブロックをLezerの標準の解析に任せる。
function lineDepth(line: Line): number | undefined {
  const depth = (line as unknown as { depth?: unknown }).depth;
  return typeof depth === "number" ? depth : undefined;
}

// WikilinkHTMLBlockは、LezerのHTMLBlockと同じ範囲でHTMLブロックを読み、中のWikiリンクを子の要素にする。
// LezerのHTMLBlockは中をインラインとして解析しないが、本文の描画はHTMLブロックの中のWikiリンクもリンクにする。
const WikilinkHTMLBlock: BlockParser = {
  name: "WikilinkHTMLBlock",
  before: "HTMLBlock",
  parse(cx, line) {
    if (line.next !== LESS_THAN || lineDepth(line) === undefined) return false;
    const rest = line.text.slice(line.pos);
    const style = HTML_BLOCK_STYLES.find(({ start }) => start.test(rest));
    if (!style) return false;

    const { end } = style;
    const from = cx.lineStart + line.pos;
    const lines = [rest];
    const marks: Element[] = [];
    let trailing = end !== EMPTY_LINE;
    while (!end.test(line.text) && cx.nextLine()) {
      if ((lineDepth(line) ?? 0) < cx.depth) {
        trailing = false;
        break;
      }
      marks.push(...line.markers);
      // 入れ物の記号 (引用の `>` など) を空白にし、文字の位置を文書の位置に揃える。
      lines.push(" ".repeat(line.basePos) + line.text.slice(line.basePos));
    }
    if (trailing) cx.nextLine();
    const to = cx.prevLineEnd();

    const links = htmlBlockWikilinks(lines.join("\n").slice(0, to - from));
    const children = [...marks, ...links.map((link) => wikilinkElement(cx, from + link.from, from + link.to))];
    cx.addElement(
      cx.elt(
        style.node,
        from,
        to,
        children.sort((a, b) => a.from - b.from),
      ),
    );
    return true;
  },
};

// wikilinkTagはWikiリンクの記法のタグ。リンクのタグの下位にし、Wikiリンクを区別しない色付けでは
// リンクとして扱われるようにする。
export const wikilinkTag = Tag.define(tags.link);

// Wikilinkは、同じ行の `[[` から最初の `]]` までを1つの要素にする。範囲は本文の描画
// (internal/markup/wikilink.goのwikilinkRegex) に揃え、中身が空白だけのものはWikiリンクにしない。
// インラインコードとHTMLのcode・pre要素に重なる候補を除き、コードの後ろのリンクは独立して扱う。
// HTMLのタグは、段落・見出し・表のセルの1つのインライン本文かHTMLブロックの中だけで読む。
// コードブロックとエスケープされた開始括弧は、Lezerの標準の解析でも除かれる。
export const Wikilink: MarkdownConfig = {
  defineNodes: [
    { name: "Wikilink", style: { "Wikilink/...": wikilinkTag } },
    { name: "WikilinkMark", style: tags.processingInstruction },
  ],
  parseBlock: [WikilinkHTMLBlock],
  parseInline: [
    {
      name: "WikilinkHTMLCode",
      before: "HTMLTag",
      parse(cx, next, pos) {
        if (next !== LESS_THAN || !/^<(?:code|pre)(?=[\s/>])/i.test(cx.slice(pos, pos + 6))) return -1;
        const boundary = nextBoundary(cx, pos);
        if (!boundary || boundary.closing || boundary.from !== pos - cx.offset) return -1;
        // HTMLのコードもInlineCodeにし、中の角括弧を標準のリンク解析に渡さない。
        return cx.addElement(cx.elt("InlineCode", pos, cx.offset + boundary.to));
      },
    },
    {
      name: "Wikilink",
      before: "Link",
      parse(cx, next, pos) {
        if (next !== OPEN_BRACKET || cx.char(pos + 1) !== OPEN_BRACKET) return -1;

        const boundary = nextBoundary(cx, pos);
        const start = pos - cx.offset;
        if (!boundary?.closing || cx.text.slice(start + 2, boundary.from).trim() === "") return -1;

        return cx.addElement(wikilinkElement(cx, pos, cx.offset + boundary.to));
      },
    },
  ],
};

// linkDestinationTagは、リンクの `(...)` の部分 (括弧・URL・タイトル) のタグ。リンクのテキストの側の
// `[...]` と色を分けるために使う。
export const linkDestinationTag = Tag.define();

type InlineParse = NonNullable<MarkdownConfig["parseInline"]>[number]["parse"];

// 公開の型には無い値だが、標準のリンクの解析器と、解析中のインラインの要素の並び・要素の子を読む。
// 読めなければundefinedを返し、リンクを標準の解析のままにする (`(...)` の色がリンクのテキストと同じになる)。
const standardLinkEnd = ((): InlineParse | undefined => {
  const parser = commonmarkLanguage.parser as unknown as { inlineParsers?: unknown; inlineNames?: unknown };
  if (!Array.isArray(parser.inlineParsers) || !Array.isArray(parser.inlineNames)) return undefined;
  const parse: unknown = parser.inlineParsers[parser.inlineNames.indexOf("LinkEnd")];
  return typeof parse === "function" ? (parse as InlineParse) : undefined;
})();

function inlineParts(cx: InlineContext): unknown[] | undefined {
  const parts = (cx as unknown as { parts?: unknown }).parts;
  return Array.isArray(parts) ? parts : undefined;
}

function elementChildren(element: Element): readonly Element[] | undefined {
  const children = (element as unknown as { children?: unknown }).children;
  return Array.isArray(children) ? (children as Element[]) : undefined;
}

// LinkDestinationは、標準のリンクの解析 (LinkEnd) を同じ名前で置き換え、標準の解析が作ったリンクと画像の
// `(...)` の部分を、LinkDestinationの要素で包む。リンクの範囲と子の要素は標準の解析のまま変えない。
// 参照リンク (`[テキスト][ラベル]`) は `(...)` を持たないため包まない。
export const LinkDestination: MarkdownConfig = {
  defineNodes: [{ name: "LinkDestination", style: { "LinkDestination/...": linkDestinationTag } }],
  parseInline: standardLinkEnd
    ? [
        {
          name: "LinkEnd",
          parse(cx, next, pos) {
            const end = standardLinkEnd(cx, next, pos);
            const parts = inlineParts(cx);
            if (end < 0 || !parts || cx.char(pos + 1) !== OPEN_PAREN) return end;

            // 標準の解析は、リンクの要素を解析中の並びの末尾に置き、`]` の直後の `(` を子の記号にする。
            const link = parts[parts.length - 1] as Element | undefined;
            const children = link && link.to === end ? elementChildren(link) : undefined;
            const index = children?.findIndex((child) => child.from === pos + 1) ?? -1;
            if (!link || !children || index < 0) return end;

            const type = link.type === cx.elt("Image", 0, 0).type ? "Image" : "Link";
            const destination = cx.elt("LinkDestination", pos + 1, end, children.slice(index));
            parts[parts.length - 1] = cx.elt(type, link.from, end, [...children.slice(0, index), destination]);
            return end;
          },
        },
      ]
    : [],
};

// AutolinkMarkは、山括弧で囲んだURL (`<https://...>`) の `<` `>` に、記号のタグに加えてURLのタグを当て、
// URLと同じ色にする。
const AutolinkMark: MarkdownConfig = {
  props: [styleTags({ "Autolink/LinkMark": [tags.url, tags.processingInstruction] })],
};

// TableWithoutHeaderStyleは、表の見出し行に見出しのタグを当てない表の拡張。見出し行の文字は本文の行と同じく
// 色を付けず、見出し行の `|` もほかの行と同じ記号の色にする。
const TableWithoutHeaderStyle: MarkdownConfig = {
  ...Table,
  defineNodes: Table.defineNodes?.map((node) =>
    typeof node !== "string" && node.name === "TableHeader" ? { ...node, style: undefined } : node,
  ),
};

// markdownLanguageはエディタと差分で共有するMarkdownの言語。本文の描画 (internal/markup/markup.goの
// goldmark) と同じ拡張を足す。goldmarkのLinkifyにあたるのがAutolinkである。
export const markdownLanguage = new Language(
  commonmarkLanguage.data,
  (commonmarkLanguage.parser as MarkdownParser).configure([
    TableWithoutHeaderStyle,
    Strikethrough,
    TaskList,
    Autolink,
    AutolinkMark,
    Wikilink,
    LinkDestination,
  ]),
  [],
  "markdown",
);
