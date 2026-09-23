import { beforeEach, describe, expect, it } from "vitest";

import { initializeMarkdownTables } from "./markdown-table";

const LABEL = "Scrollable table";

// pages/page/show.templとpages/page/preview.templが描画する本文コンテナ。ラベル属性が部分木の
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

  // ページ表示画面は本文をサーバー側で描画するため、初期化処理が走る時点でテーブルは既に文書の
  // 中にある。テーブルを知らせるスワップは起きない。
  it("サーバーで描画されたテーブルをフォーカス可能なラベル付き領域で包む", () => {
    document.body.innerHTML = bodyMarkup();

    initializeMarkdownTables();

    const wrapper = table().parentElement as HTMLElement;
    expect(wrapper.className).toBe("wikino-markdown-table-scroll");
    expect(wrapper.tabIndex).toBe(0);
    expect(wrapper.getAttribute("role")).toBe("region");
    expect(wrapper.getAttribute("aria-label")).toBe(LABEL);
  });

  // ラッパーがスクロールを引き取るため、テーブルはテーブルのままである必要がある。テーブルの
  // ロールをregionに置き換えると、ラッパーが見せようとしている行と列の構造をスクリーンリーダー
  // 利用者が失ってしまう。
  it("テーブル自体は変更しない", () => {
    document.body.innerHTML = bodyMarkup();

    initializeMarkdownTables();

    expect(table().hasAttribute("role")).toBe(false);
    expect(table().hasAttribute("tabindex")).toBe(false);
    expect(table().querySelector("th")?.textContent).toBe("列");
  });

  // ページ編集画面のプレビューはhtmx経由で届く。htmxは本文コンテナ自体をスワップするため、
  // settleした要素は本文コンテナの祖先ではなくコンテナそのものになる。
  it("htmxでスワップされた本文のテーブルを包む", () => {
    initializeMarkdownTables();

    document.body.innerHTML = bodyMarkup();
    const body = document.querySelector("[data-markdown-table-label]") as HTMLElement;
    body.dispatchEvent(new Event("htmx:after:settle", { bubbles: true }));

    expect(wrappers()).toHaveLength(1);
    expect(table().parentElement).toBe(wrappers()[0]);
  });

  // プレビューは入力による再描画のたびにsettleし、ページ表示画面は既に包み終えた本文の上で
  // 関連ページ一覧をsettleさせる。
  it("同じ本文が再度スワップされてもラッパーを1つに保つ", () => {
    document.body.innerHTML = bodyMarkup();

    initializeMarkdownTables();
    const body = document.querySelector("[data-markdown-table-label]") as HTMLElement;
    body.dispatchEvent(new Event("htmx:after:settle", { bubbles: true }));

    expect(wrappers()).toHaveLength(1);
    expect(table().parentElement).toBe(wrappers()[0]);
  });

  // 本文以外の画面も自前のテーブルを描画するが、それらは幅を自分で制御しているレイアウトの中に
  // ある。包んでしまうと、読み上げる名前を持たないランドマークとタブ位置が増えるだけになる。
  it("ページ本文の外にあるテーブルは処理しない", () => {
    document.body.innerHTML = "<div><table><tbody><tr><td>値</td></tr></tbody></table></div>";

    initializeMarkdownTables();

    expect(wrappers()).toHaveLength(0);
  });

  // regionがランドマークとして公開されるのはアクセシブルな名前を持つときだけであるため、名前の
  // 無いregionは付けない。スクロールした列へのキーボードでの到達は読み上げの有無に依存しない。
  it("本文にラベルが無ければラッパーにラベルを付けず、フォーカス可能な状態を保つ", () => {
    document.body.innerHTML = bodyMarkup("");

    initializeMarkdownTables();

    const wrapper = table().parentElement as HTMLElement;
    expect(wrapper.tabIndex).toBe(0);
    expect(wrapper.hasAttribute("role")).toBe(false);
    expect(wrapper.hasAttribute("aria-label")).toBe(false);
  });
});
