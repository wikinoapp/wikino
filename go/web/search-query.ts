// 検索の画面のキーワードの欄 (pages/search/show.templ) に、`space:` の指定の後ろへカーソルを置く。
//
// スペースの中から検索を開くと、欄には `space:識別子` だけが入っている。そのまま打ち始めると
// キーワードが指定の前に入ってしまうため、指定の後ろに空白を足してカーソルを移す。キーワードが
// 既にあるときは、入れ直しやすいよう先頭に置く。

const INPUT_SELECTOR = "[data-search-query][data-search-query-has-space-filter]";
const SPACE_FILTER_PATTERN = /space:\S+/g;

export function initializeSearchQuery(): void {
  const input = document.querySelector<HTMLInputElement>(INPUT_SELECTOR);
  if (!input) return;

  // Safariでは、ホットキーで検索の画面を開いたとき、描画の直後にはまだ欄の値とフォーカスが
  // 整っておらず、置いたカーソルが戻されてしまう。描画が2回進むのを待ってから置く。
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      placeCursorAfterSpaceFilters(input);
    });
  });
}

// placeCursorAfterSpaceFiltersは、欄の値が `space:` の指定だけのとき、最後の指定の後ろに空白を
// 足してカーソルを置く。指定以外のキーワードがあるときは、カーソルを先頭に置く。
export function placeCursorAfterSpaceFilters(input: HTMLInputElement): void {
  const value = input.value;
  const matches = [...value.matchAll(SPACE_FILTER_PATTERN)];
  const last = matches.at(-1);
  if (!last) return;

  if (value.replace(SPACE_FILTER_PATTERN, "").trim().length > 0) {
    input.setSelectionRange(0, 0);
    return;
  }

  const end = (last.index ?? 0) + last[0].length;
  if (value[end] !== " ") {
    input.value = `${value.slice(0, end)} ${value.slice(end)}`;
  }
  input.setSelectionRange(end + 1, end + 1);
}
