import { ensureSyntaxTree, type Language } from "@codemirror/language";
import { EditorState } from "@codemirror/state";
import { commonmarkLanguage } from "@codemirror/lang-markdown";
import { Autolink, type MarkdownParser, Strikethrough, Table, TaskList } from "@lezer/markdown";
import { describe, expect, it } from "vitest";

import { markdownLanguage } from "./markdown-language";

// wikilinksを、本文を解析してWikiリンクになった範囲の文字を順に返す。
function wikilinks(text: string): string[] {
  const found: string[] = [];
  markdownLanguage.parser.parse(text).iterate({
    enter: (node) => {
      if (node.name === "Wikilink") found.push(text.slice(node.from, node.to));
    },
  });
  return found;
}

// editorWikilinksを、エディタの状態の構文木を本文の末尾まで作り、Wikiリンクになった範囲の文字を順に返す。
// 編集のあとは、前の構文木を再利用するインクリメンタルな解析の結果になる。
function editorWikilinks(state: EditorState): string[] {
  const tree = ensureSyntaxTree(state, state.doc.length, 2000);
  expect(tree).not.toBeNull();
  const found: string[] = [];
  tree?.iterate({
    enter(node) {
      if (node.name === "Wikilink") found.push(`${node.from}:${state.doc.sliceString(node.from, node.to)}`);
    },
  });
  return found;
}

// standardParserは、Wikiリンクの拡張を足していない、ほかは同じ拡張の解析器。
const standardParser = (commonmarkLanguage.parser as MarkdownParser).configure([
  Table,
  Strikethrough,
  TaskList,
  Autolink,
]);

// blocksを、本文を解析したブロックの種類と範囲を順に返す。Wikiリンクとその記号は含めない。
function blocks(parser: MarkdownParser, text: string): string[] {
  const found: string[] = [];
  parser.parse(text).iterate({
    enter(node) {
      if (["HTMLBlock", "CommentBlock", "Paragraph", "Blockquote", "ListItem", "QuoteMark"].includes(node.name)) {
        found.push(`${node.name}:${node.from}-${node.to}`);
      }
    },
  });
  return found;
}

// longBodyを、見出し・段落・リスト・HTMLブロックを繰り返し、ところどころにHTMLのコードを含む、本文の上限
// (50万字) 近くの長さの本文を返す。
function longBody(): string {
  return Array.from(
    { length: 4400 },
    (_, i) =>
      `## 見出し${i}\n\n本文の段落です。**強調**と\`コード\`と[リンク](https://wikino.app)と[[ページ${i}]]があります。` +
      (i % 10 === 0 ? ` <code>x${i}</code>` : "") +
      `\n\n- 項目1 [[項目${i}]]\n- 項目2\n\n` +
      (i % 10 === 5
        ? `<details>\n<summary>詳細${i}</summary>\n[[折りたたみ${i}]] と <code>[[コード]]</code>\n</details>\n\n`
        : ""),
  ).join("");
}

// fastestParseTimeを、本文の解析を最大3回繰り返したうちの最も短い時間 (ミリ秒) を返す。GCなどによる
// 一時的な遅れで比較が揺れないようにするため。limitを渡したときは、その時間に収まった時点で止める。
function fastestParseTime(parser: typeof markdownLanguage.parser, text: string, limit = 0): number {
  let fastest = Infinity;
  for (let i = 0; i < 3 && fastest >= limit; i++) {
    const start = performance.now();
    parser.parse(text);
    fastest = Math.min(fastest, performance.now() - start);
  }
  return fastest;
}

// expectParseTimeNearCommonmarkは、本文の解析にかかる時間が、拡張を足していない標準の解析器
// (commonmarkLanguage) で解析する時間の10倍に収まることを確かめる。実装の内部の呼び出しに依らずに、
// 処理量が入力長の二乗に増えていないことを見るためである。本文の上限 (50万字) 近くの長さでは、二乗の
// 処理量になると標準の解析器の十数倍以上の時間がかかる。標準の解析器はHTMLブロックの中を読まないため、
// HTMLブロックの本文では、同じ長さの普通の本文 (longBody) を解析する時間と比べる。
function expectParseTimeNearCommonmark(text: string, baseline = text) {
  const limit = fastestParseTime(commonmarkLanguage.parser, baseline) * 10;
  expect(fastestParseTime(markdownLanguage.parser, text, limit)).toBeLessThan(limit);
}

// fastestUpdateTimeを、本文を開いて構文木をuptoまで作ったあと、posに1文字ずつ10回挿入したときの、状態の
// 更新にかかった最も短い時間 (ミリ秒) を返す。CodeMirrorは編集のたびに前の構文木を渡して解析を始め直すため、
// 更新の時間に、編集のたびの解析の量が表れる。
function fastestUpdateTime(language: Language, text: string, pos: number, upto: number): number {
  let state = EditorState.create({ doc: text, extensions: [language] });
  ensureSyntaxTree(state, upto, 1e9);
  let fastest = Infinity;
  for (let i = 0; i < 10; i++) {
    const start = performance.now();
    state = state.update({ changes: { from: pos, insert: "a" } }).state;
    fastest = Math.min(fastest, performance.now() - start);
    ensureSyntaxTree(state, upto + i + 1, 1e9);
  }
  return fastest;
}

// expectUpdateTimeNearCommonmarkは、編集のたびの解析にかかる時間が、標準の解析器の3倍に収まることを
// 確かめる。編集した箇所の外まで解析し直すと、本文の上限近くの長さでは標準の解析器の数倍から数十倍かかる。
// 1ms前後の時間の比は揺れやすいため、上限は10msより短くしない。
function expectUpdateTimeNearCommonmark(text: string, pos: number, upto: number) {
  const limit = Math.max(fastestUpdateTime(commonmarkLanguage, text, pos, upto) * 3, 10);
  expect(fastestUpdateTime(markdownLanguage, text, pos, upto)).toBeLessThan(limit);
}

describe("markdownLanguage", () => {
  it("[[ から最初の ]] までをWikiリンクにする", () => {
    expect(wikilinks("[[ページA]]と[[トピック/ページB]]")).toEqual(["[[ページA]]", "[[トピック/ページB]]"]);
  });

  it("段落・見出し・リスト・テーブルの中のWikiリンクを解析する", () => {
    expect(wikilinks("## [[見出し]]\n\n- [[項目]]\n\n| [[セル]] |\n| --- |")).toEqual([
      "[[見出し]]",
      "[[項目]]",
      "[[セル]]",
    ]);
  });

  it("コードとして書いた [[...]] をWikiリンクにしない", () => {
    expect(wikilinks("`[[インライン]]`\n\n```\n[[コードブロック]]\n```\n\n    [[インデント]]")).toEqual([]);
  });

  it("エスケープした [[ をWikiリンクにしない", () => {
    expect(wikilinks("\\[[ページ]]")).toEqual([]);
  });

  it("行をまたぐ [[...]] をWikiリンクにしない", () => {
    expect(wikilinks("[[ページ\n名]]")).toEqual([]);
  });

  it("中身が空白だけの [[ ]] をWikiリンクにしない", () => {
    expect(wikilinks("[[ ]]")).toEqual([]);
  });

  it("HTMLのcodeとpreの中をWikiリンクにしない", () => {
    expect(wikilinks('<code class="sample">[[コード]]</code> と [[ページ]]')).toEqual(["[[ページ]]"]);
    expect(wikilinks("前置き <PRE>[[コード]]</PRE> と [[ページ]]")).toEqual(["[[ページ]]"]);
    expect(wikilinks("<code>[[閉じていないコード]]")).toEqual([]);
  });

  it("コードの後ろのWikiリンクを、手前の未閉鎖の開始括弧とは分けて解析する", () => {
    expect(wikilinks("[[ と書く `x` [[ページ1]]")).toEqual(["[[ページ1]]"]);
    expect(wikilinks("[[ページ`code`名]]")).toEqual([]);
    expect(wikilinks("[[前 **`code`** [[後ろ]]")).toEqual(["[[後ろ]]"]);
    expect(wikilinks("[[前 <code>code</code> [[後ろ]]")).toEqual(["[[後ろ]]"]);
  });

  it("閉じていないバッククォートはWikiリンクのタイトルに含める", () => {
    expect(wikilinks("[[ページ`名]]")).toEqual(["[[ページ`名]]"]);
  });

  it("見出しのインライン本文をコードブロックとして読み直さない", () => {
    expect(wikilinks("## ```code``` [[ページ]]")).toEqual(["[[ページ]]"]);
  });

  it("HTMLタグの属性の開始括弧が後続のWikiリンクに影響しない", () => {
    expect(wikilinks('<span title="[[">本文</span> [[ページ]]')).toEqual(["[[ページ]]"]);
  });

  it("閉じ記号が無い長い本文を、開始括弧ごとに末尾まで読み直さずに解析する", () => {
    expectParseTimeNearCommonmark("[".repeat(500000));
  });

  it("開いていないcode・preの閉じタグが続く長い本文を、閉じタグごとにタグを探し直さずに解析する", () => {
    expectParseTimeNearCommonmark("a " + "<pre>".repeat(50000) + "</code>".repeat(35000));
    expectParseTimeNearCommonmark("a " + "<code>".repeat(40000) + "</pre>".repeat(40000));
  });

  it("対応する閉じタグの後ろのWikiリンクを解析する", () => {
    expect(wikilinks("<code><pre>[[コード]]</code> [[ページ]]")).toEqual(["[[ページ]]"]);
    expect(wikilinks("前置き <pre><code>[[コード]]</code>[[まだpre]]</pre> [[ページ]]")).toEqual(["[[ページ]]"]);
  });

  it("1行に多数のWikiリンクがあってもすべて解析する", () => {
    expect(wikilinks("[[ページ]] ".repeat(2000))).toEqual(Array<string>(2000).fill("[[ページ]]"));
  });

  // 本文の描画は、段落をまたいで開いたままのタグや、HTMLの表のセルの境目まで含めてHTMLの木を組み立てる。
  // エディタではタグを1つのインライン本文の中だけで読み、色の付き方だけが違う入力として許容する。
  it("HTMLのコードを、開いたインライン本文の中だけで判定する", () => {
    expect(wikilinks("<code>コード\n\n[[中]]</code> と [[外]]")).toEqual(["[[中]]", "[[外]]"]);
    expect(wikilinks("前置き <code>[[中]]\n\n[[外]]")).toEqual(["[[外]]"]);
    expect(wikilinks("前置き <table><tr><td><code>[[中]]</td></tr></table> [[同じ段落]]\n\n[[外]]")).toEqual([
      "[[外]]",
    ]);
  });

  it("コード記法やコメント内のHTMLタグを、後続のコードの開始としない", () => {
    expect(wikilinks("`<code>` [[外]]")).toEqual(["[[外]]"]);
    expect(wikilinks("前置き <!-- <code> --> [[外]]")).toEqual(["[[外]]"]);
  });

  it("本文の描画でタグにならない `<code>` を、後続のコードの開始としない", () => {
    expect(wikilinks("[x](<code>) [[外]]")).toEqual(["[[外]]"]);
    expect(wikilinks('[x](url "<code>") [[外]]')).toEqual(["[[外]]"]);
    expect(wikilinks("![<code>](a.png) [[外]]")).toEqual(["[[外]]"]);
    expect(wikilinks("a <textarea><code></textarea> [[外]]")).toEqual(["[[外]]"]);
    expect(wikilinks("a <script><code></script> [[外]]")).toEqual(["[[外]]"]);
  });

  it("リンクのテキストに書いた `<code>` は、後続のコードの開始にする", () => {
    expect(wikilinks("[<code>](url) [[中]]")).toEqual([]);
  });

  it("本文の描画が中身を捨てるraw textの要素の中をWikiリンクにしない", () => {
    expect(wikilinks("前置き <script>[[中]]</script> [[外]]")).toEqual(["[[外]]"]);
    expect(wikilinks("前置き <title>[[中]]</title> [[外]]")).toEqual(["[[外]]"]);
    // textareaの中身は文字として描画され、Wikiリンクにもなる。
    expect(wikilinks("前置き <textarea>[[中]]</textarea> [[外]]")).toEqual(["[[中]]", "[[外]]"]);
  });

  it("表のセルで開いたコードを、セルの終わりで閉じる", () => {
    expect(wikilinks("| <code> | [[中]] |\n| --- | --- |\n\n[[外]]")).toEqual(["[[中]]", "[[外]]"]);
    expect(wikilinks("| <code>x | [[隣のセル]] |\n| --- | --- |")).toEqual(["[[隣のセル]]"]);
  });

  it("HTMLブロックの中のWikiリンクを解析する", () => {
    expect(wikilinks("<details>\n<summary>詳細</summary>\n[[ページ]]\n</details>")).toEqual(["[[ページ]]"]);
    expect(wikilinks('<div align="center">\n[[ページ]] と <code>[[コード]]</code>\n</div>')).toEqual(["[[ページ]]"]);
    expect(wikilinks('<div>\n<span title="[[属性]]">x</span> [[ページ]]\n</div>')).toEqual(["[[ページ]]"]);
  });

  it("HTMLブロックの中のタグ・コメント・コード・中身を捨てるraw textの要素をWikiリンクにしない", () => {
    expect(wikilinks("<div>\n<!-- [[中]] --> [[外]]\n</div>")).toEqual(["[[外]]"]);
    expect(wikilinks("<div>\n<pre>[[中]]</pre> [[外]]\n</div>")).toEqual(["[[外]]"]);
    expect(wikilinks("<div>\n<script>[[中]]</script> [[外]]\n</div>")).toEqual(["[[外]]"]);
    expect(wikilinks("<div>\n<textarea>[[中]]</textarea> [[外]]\n</div>")).toEqual(["[[中]]", "[[外]]"]);
    expect(wikilinks("<div>\n[[a <b>x</b> b]] [[外]]\n</div>")).toEqual(["[[外]]"]);
    expect(wikilinks("<div>\n<code>[[閉じていないコード]]\n</div>\n\n[[外]]")).toEqual(["[[外]]"]);
  });

  it("HTMLブロックの属性値のコメント開始文字を、コメントとして読まない", () => {
    for (const body of [
      '<div title="<!--">\n[[ページ]]\n</div>',
      "<div title='<!--'>[[ページ]]</div>",
      '<div>\n<span title="<!--">説明</span> [[ページ]]\n</div>',
      '<div title="<!-- > [[属性]]">[[ページ]]</div>',
      '<div title="<!--">\n<!-- [[コメント]] --> [[ページ]]\n</div>',
    ]) {
      expect(wikilinks(body)).toEqual(["[[ページ]]"]);
    }
  });

  it("HTMLブロックのraw textのコメント開始文字を、コメントとして読まない", () => {
    for (const name of ["textarea", "xmp"]) {
      expect(wikilinks(`<div>\n<${name}><!--</${name}> [[ページ]]\n</div>`)).toEqual(["[[ページ]]"]);
      expect(wikilinks(`<div><${name}><!--</${name}> [[ページ]] --></div>`)).toEqual(["[[ページ]]"]);
    }
    expect(wikilinks("<div><script><!--</script> [[ページ]]</div>")).toEqual(["[[ページ]]"]);
  });

  it("HTMLブロック内のコメントは、空行や引用符を含んでも中のタグを読まない", () => {
    expect(wikilinks('<!-- title="\n\n<code>[[中]] --> [[ページ]]')).toEqual(["[[ページ]]"]);
    expect(wikilinks("<div><!---> [[ページ]]</div>")).toEqual(["[[ページ]]"]);
    expect(wikilinks("<div><!-- [[中]] --!> [[ページ]]</div>")).toEqual(["[[ページ]]"]);
    expect(wikilinks("<div><!-- [[閉じていないコメント]]</div>")).toEqual([]);
  });

  it("HTMLブロック内の処理命令・宣言・CDATAの中をWikiリンクにしない", () => {
    expect(wikilinks('<div><?sample "<!-- [[中]]" ?> [[ページ]]</div>')).toEqual(["[[ページ]]"]);
    expect(wikilinks('<div><!DOCTYPE html PUBLIC "<!-- [[中]]"> [[ページ]]</div>')).toEqual(["[[ページ]]"]);
    expect(wikilinks("<div><![CDATA[ <!-- [[中]] ]]> [[ページ]]</div>")).toEqual(["[[ページ]]"]);
  });

  it("HTMLブロック内の処理命令・CDATA・宣言を、HTMLと同じく最初の `>` で終える", () => {
    expect(wikilinks("<div>\n<?a> [[x]] ?>\n</div>")).toEqual(["[[x]]"]);
    expect(wikilinks("<div>\n<![CDATA[ a> [[x]] ]]>\n</div>")).toEqual(["[[x]]"]);
    expect(wikilinks('<div>\n<!x "> [[x]] ">\n</div>')).toEqual(["[[x]]"]);
  });

  it("HTMLブロック内のタグで、属性値の始まりの引用符だけを引用符として読む", () => {
    expect(wikilinks('<div>\n<p a"b>[[x]]</p>\n</div>')).toEqual(["[[x]]"]);
    expect(wikilinks('<div>\n<p title=a"b>[[x]]</p>\n</div>')).toEqual(["[[x]]"]);
    expect(wikilinks('<div>\n<p a=b="x>[[x]]">\n</div>')).toEqual(["[[x]]"]);
    // 属性値の始まりの引用符の中の `>` は、`=` の前後に空白や改行があってもタグを終えない。
    expect(wikilinks("<div>\n<p title\n=\n'a>b'>[[x]]</p>\n</div>")).toEqual(["[[x]]"]);
    expect(wikilinks("<div>\n<p title=\"a>b\" data-x='[[属性]]'>[[x]]</p>\n</div>")).toEqual(["[[x]]"]);
  });

  it("行頭から書いたコメントの後ろのWikiリンクを解析し、コメントの中は解析しない", () => {
    expect(wikilinks("<!-- <code> [[中]] --> [[外]]")).toEqual(["[[外]]"]);
    expect(wikilinks("<!-- [[中]]\n\n[[中2]] --> [[外]]\n\n[[後]]")).toEqual(["[[外]]", "[[後]]"]);
    expect(wikilinks("<div>\n<!--> [[外]]\n</div>")).toEqual(["[[外]]"]);
    // コメントの中の </pre> はタグにならず、preは閉じない。
    expect(wikilinks("<pre>\n<!-- [[中]]\n\n</pre> --> [[外]]")).toEqual([]);
  });

  it("HTMLブロックの中では、Markdownのコードとエスケープを働かせない", () => {
    expect(wikilinks("<div>\n`[[ページ]]`\n</div>")).toEqual(["[[ページ]]"]);
    expect(wikilinks("<div>\n\\[[ページ]]\n</div>")).toEqual(["[[ページ]]"]);
  });

  it("行頭のpreで始まるHTMLブロックは、空行をまたいで閉じタグまでをコードにする", () => {
    expect(wikilinks("<pre>\n[[中]]\n\n[[中2]]\n</pre> [[外]]\n\n[[後]]")).toEqual(["[[外]]", "[[後]]"]);
  });

  it("引用・リストの中のHTMLブロックのWikiリンクを解析する", () => {
    expect(wikilinks("> <div>\n> [[ページ]]\n> </div>")).toEqual(["[[ページ]]"]);
    expect(wikilinks("> <div>\n> [[中]]\n[[外]]")).toEqual(["[[中]]", "[[外]]"]);
    expect(wikilinks("- <div>\n  [[ページ]]\n  </div>")).toEqual(["[[ページ]]"]);
    expect(wikilinks("- <div>\n  [[中]]\n- [[外]]")).toEqual(["[[中]]", "[[外]]"]);
  });

  it("HTMLブロックの範囲を、Wikiリンクの拡張を足していない解析器と同じにする", () => {
    for (const text of [
      "<details>\n<summary>詳細</summary>\n[[ページ]]\n</details>\n\n後ろ",
      "<pre>x</pre> [[外]]\n後ろ",
      "<pre>\n[[中]]\n\n</pre>\n後ろ",
      "<pre>\n[[閉じていない]]\n\n後ろ",
      "<div>\n[[ページ]]",
      "<!-- [[中]]\n-->\n\n<div>\n[[外]]\n</div>",
      "<!-- a\n\nb --> [[外]]\n\n後ろ",
      "<!-- 閉じていない\n\n[[中]]",
      "<?php [[中]] ?>\n\n後ろ",
      "> <div>\n> [[中]]\n>\n> [[外]]\n後ろ",
      "> <div>\n> [[中]]\n[[外]]",
      "- <div>\n  [[中]]\n- [[外]]\n\n  後ろ",
      "- a\n\n  > <pre>\n  > [[中]]\n  > </pre>\n- [[外]]",
      "段落\n<div>\n[[中]]\n</div>",
      "段落\n<span>\n[[中]]",
    ]) {
      expect(blocks(markdownLanguage.parser as MarkdownParser, text)).toEqual(blocks(standardParser, text));
    }
  });

  it("長いHTMLブロックの中の多数の開始括弧とタグを、開始括弧ごとに読み直さずに解析する", () => {
    for (const unit of ["<b>[[ページ]]</b> [[ ", "<code>[[", "<!--[[", "<pre></code>[["]) {
      const text = "<div>\n" + unit.repeat(Math.floor(400000 / unit.length));
      expectParseTimeNearCommonmark(text, longBody().slice(0, text.length));
    }
  });

  it("長いHTMLブロックのraw text内で、タグに似た文字列を位置ごとに末尾まで読み直さない", () => {
    const text = "<div>\n<textarea>" + "<a".repeat(200000) + "</textarea> [[ページ]]\n</div>";
    expectParseTimeNearCommonmark(text, longBody().slice(0, text.length));
    expect(wikilinks(text)).toEqual(["[[ページ]]"]);
  });

  it("編集でHTMLブロックの中が変わると、Wikiリンクを解析し直す", () => {
    const head = "段落です。\n\n".repeat(30);
    const body = head + "<div>\n[[中]]\n</div>\n\n[[外]]\n\n" + head;
    let state = EditorState.create({ doc: body, extensions: [markdownLanguage] });
    const fresh = () =>
      editorWikilinks(EditorState.create({ doc: state.doc.toString(), extensions: [markdownLanguage] }));
    expect(editorWikilinks(state).map((link) => link.split(":")[1])).toEqual(["[[中]]", "[[外]]"]);

    const inner = body.indexOf("[[中]]");
    state = state.update({ changes: { from: inner, insert: "<code>" } }).state;
    expect(editorWikilinks(state)).toEqual(fresh());
    expect(editorWikilinks(state).map((link) => link.split(":")[1])).toEqual(["[[外]]"]);

    state = state.update({ changes: { from: inner, to: inner + "<code>".length } }).state;
    expect(editorWikilinks(state)).toEqual(fresh());
    // HTMLブロックの手前に空行を足して位置がずれても、前の構文木のWikiリンクの位置を正しく読み替える。
    state = state.update({ changes: { from: 0, insert: "追記\n\n" } }).state;
    expect(editorWikilinks(state)).toEqual(fresh());
    // HTMLブロックの途中に空行を入れると、後ろの行は段落になる。
    state = state.update({ changes: { from: state.doc.toString().indexOf("[[中]]"), insert: "\n" } }).state;
    expect(editorWikilinks(state)).toEqual(fresh());
  });

  it("長い本文の中ほどを編集したとき、標準の解析器と同じくらいの時間で解析し直す", () => {
    const body = longBody();
    expectUpdateTimeNearCommonmark(body, Math.floor(body.length / 2), body.length);
  });

  it("背景の解析が途中までの長い本文の先頭近くを編集したとき、標準の解析器と同じくらいの時間で解析し直す", () => {
    // エディタの背景の解析は、表示範囲の10万字先で止まる。
    expectUpdateTimeNearCommonmark(longBody(), 1000, 103000);
  });

  it("本文の末尾だけを編集したときは、変更の無い段落とHTMLブロックの構文木を再利用する", () => {
    const body =
      ("段落です。".repeat(400) + "\n\n").repeat(20) +
      "<details>\n<summary>詳細</summary>\n[[中]]\n</details>\n\n[[外]]";
    const previous = markdownLanguage.parser.parse(body);
    const fragments = [{ from: 0, to: body.length, tree: previous, offset: 0, openStart: false, openEnd: true }];
    const current = markdownLanguage.parser.parse(body + "追記", fragments);
    // ルートのグループは組み直されることがあるため、子孫のノードも含めて再利用を確かめる。
    const nodes = (tree: typeof previous): object[] =>
      tree.children.flatMap((child) => [child, ...("children" in child ? nodes(child) : [])]);
    const originalNodes = new Set(nodes(previous));
    expect(nodes(current).some((node) => originalNodes.has(node))).toBe(true);
  });
});
