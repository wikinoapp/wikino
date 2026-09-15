// 描画されたページ本文のテーブルを、フォーカスできる横スクロール領域に入れる。スタイル
// (style.cssの .wikino-markdown table) の時点で横に長いテーブルがページを広げることは無くなるが、
// スクロールコンテナ自体はすべてのブラウザでキーボードから到達できるわけではないため、ポインティング
// デバイスを使わない閲覧者には画面外へスクロールした列に辿り着く手段が無い。tabindex="0" とラベル付き
// のrole="region" を持つコンテナでテーブルを包むと、スクロール領域がフォーカス可能になり、その存在も
// 伝わる。テーブル自身のロールも保たれる。同じ属性をテーブルに直接付けるとテーブルのロールが
// 置き換わり、構造が失われてしまう。
//
// 本文がDOMに現れる経路は添付ファイルローダーと同じく2つあるため、包む処理の起点も2つある。
// ページ表示画面は本文をサーバー側で描画し、初期化処理が走る時点で既にDOMにある。ページエディタの
// プレビューはhtmxが差し込み、htmx:after:settleで通知されるため、スワップされた部分木ごとに包む。

// 本文のコンテナは領域に付けるラベルを持つため、この属性が走査対象の部分木の目印と、地域化された
// 名前の供給を兼ねる (pages/page/show.templ, pages/page/preview.templ)。
const BODY_SELECTOR = "[data-markdown-table-label]";

const WRAPPER_CLASS = "wikino-markdown-table-scroll";

export function initializeMarkdownTables(): void {
  document.addEventListener("htmx:after:settle", handleAfterSettle);

  // サーバー側で描画された本文は文書と一緒に届くため、スワップイベントでは通知されない。
  wrapTables(document.body);
}

// handleAfterSettleは今スワップされた部分木のテーブルを包む。イベントはあらゆるhtmxスワップ
// から伝播するため、走査をスワップされた要素に絞ることで、本文を持たないスワップ (関連ページ一覧、
// 自動保存のOOBスワップ) が文書の残りを走査し直さないようにする。
function handleAfterSettle(event: Event): void {
  const settled = event.target;
  if (!(settled instanceof HTMLElement)) return;

  wrapTables(settled);
}

// wrapTablesはroot自身とその子孫を走査する。htmxはプレビューのコンテナ自体をスワップする
// ため、settleした要素が本文コンテナの祖先ではなく本文コンテナそのものになることがある。
function wrapTables(root: HTMLElement): void {
  const bodies = root.matches(BODY_SELECTOR)
    ? [root, ...root.querySelectorAll<HTMLElement>(BODY_SELECTOR)]
    : Array.from(root.querySelectorAll<HTMLElement>(BODY_SELECTOR));

  for (const body of bodies) {
    const label = body.dataset.markdownTableLabel ?? "";

    for (const table of body.querySelectorAll("table")) {
      wrapTable(table, label);
    }
  }
}

// wrapTableは前回の実行で既に包んだテーブルには手を付けず、再スワップされた本文がスワップの
// たびに入れ子のラッパーを増やさず、既にある1つのラッパーを保つようにする。
function wrapTable(table: HTMLTableElement, label: string): void {
  const parent = table.parentElement;
  if (parent?.classList.contains(WRAPPER_CLASS)) return;

  const wrapper = document.createElement("div");
  wrapper.className = WRAPPER_CLASS;
  wrapper.tabIndex = 0;

  // regionがランドマークとして公開されるのはアクセシブルな名前を持つときだけであるため、ロールは
  // ラベルと組で付け、ラベルが無ければ付けない。ラッパーはどちらの場合もフォーカス可能なままとする。
  // スクロールした列へのキーボードでの到達は読み上げの有無に依存しない。
  if (label !== "") {
    wrapper.setAttribute("role", "region");
    wrapper.setAttribute("aria-label", label);
  }

  table.replaceWith(wrapper);
  wrapper.appendChild(table);
}
