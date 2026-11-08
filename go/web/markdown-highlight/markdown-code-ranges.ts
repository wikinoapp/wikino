import { parser } from "@lezer/markdown";

export interface Range {
  from: number;
  to: number;
}

const CODE_ELEMENTS = new Set(["code", "pre"]);
// HTMLの字句解析で、閉じタグまでの中身をタグとして読まない要素。
const RAW_TEXT_ELEMENTS = new Set([
  "iframe",
  "noembed",
  "noframes",
  "noscript",
  "plaintext",
  "script",
  "style",
  "textarea",
  "title",
  "xmp",
]);
// raw textの要素のうち、本文の描画が中身を捨てる要素。中の [[...]] もWikiリンクにならない。
const DISCARDED_RAW_TEXT_ELEMENTS = new Set(["iframe", "noembed", "noframes", "noscript", "script", "style", "title"]);

const BLOCK_PARSERS = [
  "LinkReference",
  "IndentedCode",
  "FencedCode",
  "Blockquote",
  "HorizontalRule",
  "BulletList",
  "OrderedList",
  "ATXHeading",
  "HTMLBlock",
  "SetextHeading",
];

// 渡されるのは段落・見出し・表のセルのインライン本文なので、ブロックとしては解析しない。強調は外すが、
// リンクと画像は解析し、本文の描画と同じく、リンク先・リンクのタイトル・画像の説明の中の `<code>` を
// タグとして読まない。
const inlineParser = parser.configure({ remove: [...BLOCK_PARSERS, "Emphasis"] });

// HTMLCodeTagsは、タグを位置の順に読み、code・preの中と、中身を捨てるraw textの要素の中の範囲を求める。
class HTMLCodeTags {
  private readonly ranges: Range[] = [];
  // 開いたcode・preの名前を開いた順に持つ。名前ごとの開いている数も持ち、その名前が開いていない閉じタグは
  // 並びを探さずに飛ばす。開いていれば探した範囲はすべて取り除くため、探す量は全体で入力長に比例する。
  private readonly tags: string[] = [];
  private readonly counts = new Map<string, number>();
  private opening = 0;
  private raw: { name: string; from: number } | undefined;

  get rawTextElement(): string | undefined {
    return this.raw?.name;
  }

  read(tag: string, from: number, to: number) {
    const match = /^<(\/?)([a-z][a-z0-9-]*)(?=[\s/>])/i.exec(tag);
    const name = match?.[2]?.toLowerCase();
    if (!match || !name) return;
    const closing = match[1] === "/";

    if (this.raw) {
      if (!closing || name !== this.raw.name) return;
      if (DISCARDED_RAW_TEXT_ELEMENTS.has(name)) this.ranges.push({ from: this.raw.from, to });
      this.raw = undefined;
      return;
    }
    if (RAW_TEXT_ELEMENTS.has(name)) {
      if (!closing) this.raw = { name, from };
      return;
    }
    if (!CODE_ELEMENTS.has(name)) return;

    if (!closing) {
      if (this.tags.length === 0) this.opening = from;
      this.tags.push(name);
      this.counts.set(name, (this.counts.get(name) ?? 0) + 1);
      return;
    }
    if (!this.counts.get(name)) return;
    for (const closed of this.tags.splice(this.tags.lastIndexOf(name))) {
      this.counts.set(closed, (this.counts.get(closed) ?? 1) - 1);
    }
    if (this.tags.length === 0) this.ranges.push({ from: this.opening, to });
  }

  // finishは、閉じていない要素を本文の終わりまでとして、読んだ範囲を返す。
  finish(end: number): Range[] {
    if (this.raw && DISCARDED_RAW_TEXT_ELEMENTS.has(this.raw.name)) this.ranges.push({ from: this.raw.from, to: end });
    if (this.tags.length > 0) this.ranges.push({ from: this.opening, to: end });
    return this.ranges;
  }
}

// mergeは、範囲を位置の順に並べ、重なる範囲を1つにまとめる。
function merge(ranges: Range[]): Range[] {
  ranges.sort((a, b) => a.from - b.from);
  const merged: Range[] = [];
  for (const range of ranges) {
    const last = merged.at(-1);
    if (last && range.from < last.to) last.to = Math.max(last.to, range.to);
    else merged.push({ ...range });
  }
  return merged;
}

// markdownCodeRangesは、インライン本文のうち、Wikiリンクの候補がまたいではいけない範囲 (コードスパン、
// HTMLのcode・preの中、中身を捨てるraw textの要素の中) を位置の順に返す。HTMLのタグはこのインライン本文の
// 中だけで読み、閉じていない要素はインライン本文の終わりまでとする。
export function markdownCodeRanges(text: string): Range[] {
  if (!text.includes("`") && !text.includes("<")) return [];

  const ranges: Range[] = [];
  const html = new HTMLCodeTags();
  inlineParser.parse(text).iterate({
    enter(node) {
      if (node.name === "InlineCode") {
        ranges.push({ from: node.from, to: node.to });
        return false;
      }
      if (node.name === "Image") return false;
      if (node.name === "HTMLTag") html.read(text.slice(node.from, node.to), node.from, node.to);
      return undefined;
    },
  });
  return merge([...ranges, ...html.finish(text.length)]);
}

const HTML_SPACE = /[\t\n\f ]/;

// tagEndは、タグの終わりを返す。HTMLの字句解析と同じく、属性値の始まり (`=` と空白の直後) の引用符だけを
// 引用符として読み、その中の `>` を読み飛ばす。属性名や引用符の無い属性値の中の引用符は文字になる。
// タグの属性値の中ではコメントも始まらない。
function tagEnd(text: string, from: number): number {
  let state: "tagName" | "beforeName" | "name" | "afterName" | "beforeValue" | "unquotedValue" = "tagName";
  let quote: string | undefined;
  for (let pos = text[from + 1] === "/" ? from + 2 : from + 1; pos < text.length; pos++) {
    const char = text[pos]!;
    if (quote) {
      if (char === quote) {
        quote = undefined;
        state = "beforeName";
      }
      continue;
    }
    if (char === ">") return pos + 1;

    const space = HTML_SPACE.test(char);
    if (state === "tagName") {
      if (space || char === "/") state = "beforeName";
    } else if (state === "beforeName") {
      if (!space && char !== "/") state = "name";
    } else if (state === "name" || state === "afterName") {
      if (char === "=") state = "beforeValue";
      else if (char === "/") state = "beforeName";
      else if (space) state = "afterName";
      else state = "name";
    } else if (state === "beforeValue") {
      if (char === '"' || char === "'") quote = char;
      else if (!space) state = "unquotedValue";
    } else if (space) {
      state = "beforeName";
    }
  }
  return text.length;
}

// declarationEndは、処理命令・CDATA・宣言の終わりを返す。HTMLの本文の中では、これらはどれも最初の `>` で
// 終わる (bogus comment)。
function declarationEnd(text: string, from: number): number {
  const close = text.indexOf(">", from + 2);
  return close < 0 ? text.length : close + 1;
}

// htmlBlockCodeRangesは、HTMLブロックの本文のうち、Wikiリンクの候補がまたいではいけない範囲 (タグ・コメント、
// code・preの中、中身を捨てるraw textの要素の中) を位置の順に返す。閉じていない要素はHTMLブロックの
// 終わりまでとする。
export function htmlBlockCodeRanges(text: string): Range[] {
  const ranges: Range[] = [];
  const html = new HTMLCodeTags();
  for (let from = text.indexOf("<"); from >= 0;) {
    // raw textの中は、対応する終了タグ以外をタグやコメントとして読まない。
    // コメントの形の文字列の途中にある終了タグも、ここで通常どおり閉じる。
    const raw = html.rawTextElement;
    if (
      raw &&
      (text.slice(from, from + raw.length + 2).toLowerCase() !== `</${raw}` ||
        !/[\s/>]/.test(text.charAt(from + raw.length + 2)))
    ) {
      from = text.indexOf("<", from + 1);
      continue;
    }

    const tag = /^<\/?[a-z]/i.test(text.slice(from, from + 3));
    let to: number;
    if (!raw && text.startsWith("<!--", from)) {
      // `<!-->` と `<!--->` は、HTMLでは中身の無いコメントになる。
      const empty = /^<!---?>/.exec(text.slice(from, from + 6));
      const end = /--!?>/g;
      end.lastIndex = from + 4;
      const close = empty ? undefined : end.exec(text);
      to = empty ? from + empty[0].length : close ? close.index + close[0].length : text.length;
    } else if (!raw && (text.startsWith("<?", from) || text.startsWith("<!", from))) {
      to = declarationEnd(text, from);
    } else if (tag) {
      to = tagEnd(text, from);
      html.read(text.slice(from, to), from, to);
    } else {
      from = text.indexOf("<", from + 1);
      continue;
    }
    ranges.push({ from, to });
    from = text.indexOf("<", to);
  }
  return merge([...ranges, ...html.finish(text.length)]);
}
