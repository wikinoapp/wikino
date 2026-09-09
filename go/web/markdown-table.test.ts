import { beforeEach, describe, expect, it } from "vitest";

import { initializeMarkdownTables } from "./markdown-table";

const LABEL = "Scrollable table";

// The body container pages/page/show.templ and pages/page/preview.templ render: the label attribute
// marks the subtree and carries the localized name for the region.
//
// [Ja] pages/page/show.templ と pages/page/preview.templ が描画する本文コンテナ。ラベル属性が部分木の
// 目印と、領域に付ける地域化された名前を兼ねる。
function bodyMarkup(label: string = LABEL): string {
  return `
    <div class="wikino-markdown" data-markdown-table-label="${label}">
      <p>本文</p>
      <table>
        <thead>
          <tr><th>列</th></tr>
        </thead>
        <tbody>
          <tr><td>値</td></tr>
        </tbody>
      </table>
    </div>
  `;
}

function table(): HTMLTableElement {
  return document.querySelector("table") as HTMLTableElement;
}

function wrappers(): HTMLElement[] {
  return Array.from(document.querySelectorAll<HTMLElement>(".wikino-markdown-table-scroll"));
}

describe("initializeMarkdownTables", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
  });

  // The page detail screen renders the body on the server, so its tables are already in the document
  // when the initializers run: no swap ever announces them.
  //
  // [Ja] ページ表示画面は本文をサーバー側で描画するため、初期化処理が走る時点でテーブルは既に文書の
  // 中にある。テーブルを知らせるスワップは起きない。
  it("wraps a server-rendered table in a focusable labelled region", () => {
    document.body.innerHTML = bodyMarkup();

    initializeMarkdownTables();

    const wrapper = table().parentElement as HTMLElement;
    expect(wrapper.className).toBe("wikino-markdown-table-scroll");
    expect(wrapper.tabIndex).toBe(0);
    expect(wrapper.getAttribute("role")).toBe("region");
    expect(wrapper.getAttribute("aria-label")).toBe(LABEL);
  });

  // The wrapper takes over the scrolling, so the table has to stay a table: replacing its role with
  // region would cost screen reader users the row and column structure the wrapper exists to expose.
  //
  // [Ja] ラッパーがスクロールを引き取るため、テーブルはテーブルのままである必要がある。テーブルの
  // ロールを region に置き換えると、ラッパーが見せようとしている行と列の構造をスクリーンリーダー
  // 利用者が失ってしまう。
  it("leaves the table itself untouched", () => {
    document.body.innerHTML = bodyMarkup();

    initializeMarkdownTables();

    expect(table().hasAttribute("role")).toBe(false);
    expect(table().hasAttribute("tabindex")).toBe(false);
    expect(table().querySelector("th")?.textContent).toBe("列");
  });

  // The page-editor preview arrives through htmx, which swaps the body container itself, so the
  // settled element is the container rather than an ancestor of one.
  //
  // [Ja] ページ編集画面のプレビューは htmx 経由で届く。htmx は本文コンテナ自体をスワップするため、
  // settle した要素は本文コンテナの祖先ではなくコンテナそのものになる。
  it("wraps the tables of a body swapped in by htmx", () => {
    initializeMarkdownTables();

    document.body.innerHTML = bodyMarkup();
    const body = document.querySelector("[data-markdown-table-label]") as HTMLElement;
    body.dispatchEvent(new Event("htmx:after:settle", { bubbles: true }));

    expect(wrappers()).toHaveLength(1);
    expect(table().parentElement).toBe(wrappers()[0]);
  });

  // The preview settles on every keystroke-driven refresh, and the page detail screen settles its
  // related-page listings above an already wrapped body.
  //
  // [Ja] プレビューは入力による再描画のたびに settle し、ページ表示画面は既に包み終えた本文の上で
  // 関連ページ一覧を settle させる。
  it("keeps one wrapper when the same body settles again", () => {
    document.body.innerHTML = bodyMarkup();

    initializeMarkdownTables();
    const body = document.querySelector("[data-markdown-table-label]") as HTMLElement;
    body.dispatchEvent(new Event("htmx:after:settle", { bubbles: true }));

    expect(wrappers()).toHaveLength(1);
    expect(table().parentElement).toBe(wrappers()[0]);
  });

  // Screens outside the page body render tables of their own, and those sit in a layout that already
  // controls its width. Wrapping them would add a tab stop and a landmark with no name to announce.
  //
  // [Ja] 本文以外の画面も自前のテーブルを描画するが、それらは幅を自分で制御しているレイアウトの中に
  // ある。包んでしまうと、読み上げる名前を持たないランドマークとタブ位置が増えるだけになる。
  it("ignores a table outside a page body", () => {
    document.body.innerHTML = "<div><table><tbody><tr><td>値</td></tr></tbody></table></div>";

    initializeMarkdownTables();

    expect(wrappers()).toHaveLength(0);
  });

  // A region is exposed as a landmark only once it has an accessible name, so an unnamed one is
  // dropped. Keyboard access to the scrolled columns does not depend on the announcement.
  //
  // [Ja] region がランドマークとして公開されるのはアクセシブルな名前を持つときだけであるため、名前の
  // 無い region は付けない。スクロールした列へのキーボードでの到達は読み上げの有無に依存しない。
  it("keeps the wrapper focusable but unlabelled when the body carries no label", () => {
    document.body.innerHTML = bodyMarkup("");

    initializeMarkdownTables();

    const wrapper = table().parentElement as HTMLElement;
    expect(wrapper.tabIndex).toBe(0);
    expect(wrapper.hasAttribute("role")).toBe(false);
    expect(wrapper.hasAttribute("aria-label")).toBe(false);
  });
});
