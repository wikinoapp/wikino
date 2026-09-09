// Puts every table of a rendered page body into a focusable, horizontally scrollable region. The
// style (.wikino-markdown table in style.css) already keeps a wide table from widening the page,
// but a scroll container is not reachable by keyboard on its own in every browser, so a viewer
// without a pointing device would have no way to reach the columns that scrolled out of sight.
// Wrapping the table in a container that carries tabindex="0" and a labelled role="region" makes
// the scrolled area focusable and announces it, and leaves the table's own role untouched: the same
// attributes on the table itself would replace the table role and lose its structure.
//
// A body reaches the DOM the two ways the attachment loader also deals with, so the wrapping has
// two entry points. The page detail screen renders the body on the server, which is already in the
// DOM when the initializers run. The page-editor preview is injected by htmx, which announces
// itself with htmx:after:settle, so each settled subtree is wrapped as it arrives.
//
// [Ja] 描画されたページ本文のテーブルを、フォーカスできる横スクロール領域に入れる。スタイル
// (style.css の .wikino-markdown table) の時点で横に長いテーブルがページを広げることは無くなるが、
// スクロールコンテナ自体はすべてのブラウザでキーボードから到達できるわけではないため、ポインティング
// デバイスを使わない閲覧者には画面外へスクロールした列に辿り着く手段が無い。tabindex="0" とラベル付き
// の role="region" を持つコンテナでテーブルを包むと、スクロール領域がフォーカス可能になり、その存在も
// 伝わる。テーブル自身のロールも保たれる。同じ属性をテーブルに直接付けるとテーブルのロールが
// 置き換わり、構造が失われてしまう。
//
// 本文が DOM に現れる経路は添付ファイルローダーと同じく 2 つあるため、包む処理の起点も 2 つある。
// ページ表示画面は本文をサーバー側で描画し、初期化処理が走る時点で既に DOM にある。ページエディタの
// プレビューは htmx が差し込み、htmx:after:settle で通知されるため、スワップされた部分木ごとに包む。

// The body containers carry the label to give the region, so the attribute marks the subtrees to
// scan and supplies the localized name in one place (pages/page/show.templ, pages/page/preview.templ).
//
// [Ja] 本文のコンテナは領域に付けるラベルを持つため、この属性が走査対象の部分木の目印と、地域化された
// 名前の供給を兼ねる (pages/page/show.templ, pages/page/preview.templ)。
const BODY_SELECTOR = "[data-markdown-table-label]";

const WRAPPER_CLASS = "wikino-markdown-table-scroll";

export function initializeMarkdownTables(): void {
  document.addEventListener("htmx:after:settle", handleAfterSettle);

  // A server-rendered body arrives with the document itself, so no swap event ever announces it.
  //
  // [Ja] サーバー側で描画された本文は文書と一緒に届くため、スワップイベントでは通知されない。
  wrapTables(document.body);
}

// handleAfterSettle wraps the tables of the subtree that just settled. The event bubbles from every
// htmx swap, so scoping the scan to the swapped element keeps swaps that carry no body (the
// related-page listings, the autosave OOB swaps) from rescanning the rest of the document.
//
// [Ja] handleAfterSettle は今スワップされた部分木のテーブルを包む。イベントはあらゆる htmx スワップ
// から伝播するため、走査をスワップされた要素に絞ることで、本文を持たないスワップ (関連ページ一覧、
// 自動保存の OOB スワップ) が文書の残りを走査し直さないようにする。
function handleAfterSettle(event: Event): void {
  const settled = event.target;
  if (!(settled instanceof HTMLElement)) return;

  wrapTables(settled);
}

// wrapTables scans root and its descendants. htmx swaps the preview container itself, so the
// settled element can be the body container rather than an ancestor of one.
//
// [Ja] wrapTables は root 自身とその子孫を走査する。htmx はプレビューのコンテナ自体をスワップする
// ため、settle した要素が本文コンテナの祖先ではなく本文コンテナそのものになることがある。
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

// wrapTable leaves a table that a previous run already wrapped alone, so a re-settled body keeps
// the one wrapper it has instead of gaining a nested one on every swap.
//
// [Ja] wrapTable は前回の実行で既に包んだテーブルには手を付けず、再スワップされた本文がスワップの
// たびに入れ子のラッパーを増やさず、既にある 1 つのラッパーを保つようにする。
function wrapTable(table: HTMLTableElement, label: string): void {
  const parent = table.parentElement;
  if (parent?.classList.contains(WRAPPER_CLASS)) return;

  const wrapper = document.createElement("div");
  wrapper.className = WRAPPER_CLASS;
  wrapper.tabIndex = 0;

  // A region is exposed as a landmark only once it has an accessible name, so the role is set with
  // the label and left off without one. The wrapper stays focusable either way: keyboard access to
  // the scrolled columns does not depend on the announcement.
  //
  // [Ja] region がランドマークとして公開されるのはアクセシブルな名前を持つときだけであるため、ロールは
  // ラベルと組で付け、ラベルが無ければ付けない。ラッパーはどちらの場合もフォーカス可能なままとする。
  // スクロールした列へのキーボードでの到達は読み上げの有無に依存しない。
  if (label !== "") {
    wrapper.setAttribute("role", "region");
    wrapper.setAttribute("aria-label", label);
  }

  table.replaceWith(wrapper);
  wrapper.appendChild(table);
}
