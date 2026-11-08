import { highlightTree } from "@lezer/highlight";
import { describe, expect, it } from "vitest";

import { markdownHighlighter } from "./markdown-highlighter";
import { markdownLanguage } from "./markdown-language";

// highlightを、本文を解析して色を当てた範囲を [文字, クラス名] の並びで返す。色の無い範囲は含めない。
function highlight(text: string): [string, string][] {
  const ranges: [string, string][] = [];
  highlightTree(markdownLanguage.parser.parse(text), markdownHighlighter, (from, to, classes) => {
    ranges.push([text.slice(from, to), classes]);
  });
  return ranges;
}

describe("markdownHighlighter", () => {
  it("見出しの記号と見出しの文字にクラス名を付ける", () => {
    expect(highlight("## 見出し")).toEqual([
      ["##", "wikino-md-heading wikino-md-mark"],
      [" 見出し", "wikino-md-heading"],
    ]);
  });

  it("強調・太字・取り消し線の文字と記号にクラス名を付ける", () => {
    expect(highlight("*斜体* **太字** ~~取り消し~~")).toEqual([
      ["*", "wikino-md-emphasis wikino-md-mark"],
      ["斜体", "wikino-md-emphasis"],
      ["*", "wikino-md-emphasis wikino-md-mark"],
      ["**", "wikino-md-strong wikino-md-mark"],
      ["太字", "wikino-md-strong"],
      ["**", "wikino-md-strong wikino-md-mark"],
      ["~~", "wikino-md-strikethrough wikino-md-mark"],
      ["取り消し", "wikino-md-strikethrough"],
      ["~~", "wikino-md-strikethrough wikino-md-mark"],
    ]);
  });

  it("和文に挟まれた太字を太字にする", () => {
    expect(highlight("Wikinoは**Wiki**アプリ")).toContainEqual(["Wiki", "wikino-md-strong"]);
  });

  it("goldmarkが太字にしない、括弧で始まり和文が続く太字を太字にしない", () => {
    expect(highlight("**「括弧」**です")).toEqual([]);
  });

  it("リンクのテキスト・記号・URLにクラス名を付け、`(...)` を括弧も含めてテキストの側と見分ける", () => {
    expect(highlight("[Wikino](https://wikino.app)")).toEqual([
      ["[", "wikino-md-link wikino-md-mark"],
      ["Wikino", "wikino-md-link"],
      ["]", "wikino-md-link wikino-md-mark"],
      ["(", "wikino-md-link wikino-md-link-destination wikino-md-mark"],
      ["https://wikino.app", "wikino-md-link wikino-md-link-destination wikino-md-url"],
      [")", "wikino-md-link wikino-md-link-destination wikino-md-mark"],
    ]);
  });

  it("リンクのタイトルと画像の `(...)` も、URLの側として見分ける", () => {
    expect(highlight('[a](u "題")')).toEqual([
      ["[", "wikino-md-link wikino-md-mark"],
      ["a", "wikino-md-link"],
      ["]", "wikino-md-link wikino-md-mark"],
      ["(", "wikino-md-link wikino-md-link-destination wikino-md-mark"],
      ["u", "wikino-md-link wikino-md-link-destination wikino-md-url"],
      [' "題"', "wikino-md-link wikino-md-link-destination"],
      [")", "wikino-md-link wikino-md-link-destination wikino-md-mark"],
    ]);
    expect(highlight("![画像](/attachments/1)")).toContainEqual([
      "(",
      "wikino-md-link wikino-md-link-destination wikino-md-mark",
    ]);
  });

  it("参照リンクと、リンクにならない括弧はURLの側にしない", () => {
    expect(highlight("[a][r]\n\n[r]: https://wikino.app").some(([, classes]) => classes.includes("destination"))).toBe(
      false,
    );
    expect(highlight("[a] (u)").some(([, classes]) => classes.includes("destination"))).toBe(false);
  });

  it("入れ子のリンクは、CommonMarkと同じく内側だけをリンクにする", () => {
    expect(highlight("[x [y](u) z](v)")).toEqual([
      ["[", "wikino-md-link wikino-md-mark"],
      ["y", "wikino-md-link"],
      ["]", "wikino-md-link wikino-md-mark"],
      ["(", "wikino-md-link wikino-md-link-destination wikino-md-mark"],
      ["u", "wikino-md-link wikino-md-link-destination wikino-md-url"],
      [")", "wikino-md-link wikino-md-link-destination wikino-md-mark"],
    ]);
  });

  it("本文に書いたURLをリンクにする", () => {
    expect(highlight("詳しくは https://wikino.app を見る")).toContainEqual(["https://wikino.app", "wikino-md-url"]);
  });

  it("山括弧で囲んだURLの記号にURLのクラス名を付ける", () => {
    expect(highlight("<https://wikino.app>")).toEqual([
      ["<", "wikino-md-url wikino-md-mark"],
      ["https://wikino.app", "wikino-md-url"],
      [">", "wikino-md-url wikino-md-mark"],
    ]);
  });

  it("Wikiリンクの文字と記号にクラス名を付ける", () => {
    expect(highlight("[[ページ]]")).toEqual([
      ["[[", "wikino-md-link wikino-md-wikilink wikino-md-mark"],
      ["ページ", "wikino-md-link wikino-md-wikilink"],
      ["]]", "wikino-md-link wikino-md-wikilink wikino-md-mark"],
    ]);
  });

  it("インラインコードとコードブロックの中身をコードの1色にする", () => {
    expect(highlight("`code`\n\n```go\nfunc main() {}\n```")).toEqual([
      ["`", "wikino-md-mark"],
      ["code", "wikino-md-code"],
      ["`", "wikino-md-mark"],
      ["```", "wikino-md-mark"],
      ["func main() {}", "wikino-md-code"],
      ["```", "wikino-md-mark"],
    ]);
  });

  it("引用・リスト・タスク記法・テーブルの記号にクラス名を付ける", () => {
    expect(highlight("> 引用")).toEqual([
      [">", "wikino-md-quote wikino-md-mark"],
      [" 引用", "wikino-md-quote"],
    ]);
    expect(highlight("- [ ] タスク")).toEqual([
      ["-", "wikino-md-mark"],
      ["[ ]", "wikino-md-mark"],
    ]);
    expect(highlight("| 名前 |\n| --- |\n| 値 |")).toEqual([
      ["|", "wikino-md-mark"],
      ["|", "wikino-md-mark"],
      ["| --- |", "wikino-md-mark"],
      ["|", "wikino-md-mark"],
      ["|", "wikino-md-mark"],
    ]);
  });

  it("表の見出し行の中の記法に、本文の行と同じクラス名を付ける", () => {
    expect(highlight("| **太字** [[ページ]] |\n| --- |")).toEqual([
      ["|", "wikino-md-mark"],
      ["**", "wikino-md-strong wikino-md-mark"],
      ["太字", "wikino-md-strong"],
      ["**", "wikino-md-strong wikino-md-mark"],
      ["[[", "wikino-md-link wikino-md-wikilink wikino-md-mark"],
      ["ページ", "wikino-md-link wikino-md-wikilink"],
      ["]]", "wikino-md-link wikino-md-wikilink wikino-md-mark"],
      ["|", "wikino-md-mark"],
      ["| --- |", "wikino-md-mark"],
    ]);
  });

  it("HTMLのcode要素内のWikiリンクにリンク色を付けない", () => {
    const ranges = highlight("<code>[[コード]]</code> と [[ページ]]");
    expect(ranges).toContainEqual(["<code>[[コード]]</code>", "wikino-md-code"]);
    expect(ranges.some(([text, classes]) => text.includes("コード") && classes.includes("wikino-md-link"))).toBe(false);
    expect(ranges).toContainEqual(["ページ", "wikino-md-link wikino-md-wikilink"]);
  });

  it("閉じていない開始括弧の後ろでもコードとWikiリンクを別々の色にする", () => {
    const ranges = highlight("[[ と書く `x` [[ページ1]]");
    expect(ranges).toContainEqual(["x", "wikino-md-code"]);
    expect(ranges).toContainEqual(["[[", "wikino-md-link wikino-md-wikilink wikino-md-mark"]);
    expect(ranges).toContainEqual(["ページ1", "wikino-md-link wikino-md-wikilink"]);
    expect(ranges.some(([text]) => text.includes("と書く"))).toBe(false);
  });
  it("HTMLブロックの中のWikiリンクの文字と記号にクラス名を付け、コードの中には付けない", () => {
    const ranges = highlight("<details>\n<summary>詳細</summary>\n[[ページ]] と <code>[[コード]]</code>\n</details>");
    expect(ranges).toContainEqual(["[[", "wikino-md-link wikino-md-wikilink wikino-md-mark"]);
    expect(ranges).toContainEqual(["ページ", "wikino-md-link wikino-md-wikilink"]);
    expect(ranges.some(([text, classes]) => text.includes("コード") && classes.includes("wikino-md-link"))).toBe(false);
  });

  it("引用の中のHTMLブロックでは、引用の記号とWikiリンクにクラス名を付ける", () => {
    const ranges = highlight("> <div>\n> [[ページ]]\n> </div>");
    expect(ranges.filter(([text]) => text === ">")).toHaveLength(3);
    expect(ranges).toContainEqual(["ページ", "wikino-md-quote wikino-md-link wikino-md-wikilink"]);
  });

  it("HTMLの属性値やraw textのコメント開始文字の後ろにもWikiリンクの色を付ける", () => {
    for (const body of [
      '<div title="<!--">\n[[ページ]]\n</div>',
      '<div>\n<span title="<!--">説明</span> [[ページ]]\n</div>',
      "<div>\n<textarea><!--</textarea> [[ページ]]\n</div>",
    ]) {
      const ranges = highlight(body);
      expect(ranges).toContainEqual(["ページ", "wikino-md-link wikino-md-wikilink"]);
      expect(ranges).toContainEqual(["[[", "wikino-md-link wikino-md-wikilink wikino-md-mark"]);
      expect(ranges.some(([text]) => text.includes("<!--"))).toBe(false);
    }
  });
});
