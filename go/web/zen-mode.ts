// Zen mode toggle for the page editor. Clicking a [data-zen-mode-toggle] flips the
// "page-edit-zen" class on the [data-zen-mode-container] element (descendants in edit.templ react
// to it via Tailwind's in-[.page-edit-zen]: variant, hiding the side columns / link lists and
// widening the center column) and persists the state to the wikino_zen_mode cookie so the editor
// reopens in the same mode. The server renders the initial class from the cookie, so this script
// only handles toggling. Event delegation keeps the wiring working for content swapped in later.
//
// [Ja] ページエディタの Zenモード切り替え。[data-zen-mode-toggle] のクリックで
// [data-zen-mode-container] 要素の "page-edit-zen" クラスを反転し (edit.templ 内の子孫要素が
// Tailwind の in-[.page-edit-zen]: バリアントで反応し、左右カラムとリンク一覧の非表示・中央カラム
// の拡幅を行う)、状態を wikino_zen_mode クッキーへ保存して次回も同じモードで開けるようにする。
// 初期クラスはサーバーがクッキーから描画するため、本スクリプトは切り替えのみを担当する。
// イベント委譲により、後からスワップされた内容でも結線が機能する。

// Keep the cookie name in sync with internal/handler/page/edit.go, and the class name with
// edit.templ.
//
// [Ja] クッキー名は internal/handler/page/edit.go と、クラス名は edit.templ と同期させること。
const ZEN_MODE_COOKIE_NAME = "wikino_zen_mode";
const ZEN_MODE_CLASS = "page-edit-zen";
const ZEN_MODE_COOKIE_MAX_AGE = 60 * 60 * 24 * 365; // 1 year. [Ja] 1 年

export function initializeZenMode(): void {
  document.addEventListener("click", handleClick);
}

function handleClick(event: MouseEvent): void {
  const target = event.target as Element | null;
  if (!target) {
    return;
  }

  const toggle = target.closest("[data-zen-mode-toggle]");
  if (!toggle) {
    return;
  }

  const container = document.querySelector("[data-zen-mode-container]");
  if (!container) {
    return;
  }

  const enabled = container.classList.toggle(ZEN_MODE_CLASS);
  syncToggleButtons(enabled);
  persistZenMode(enabled);
}

// syncToggleButtons reflects the current state on every toggle button's aria-pressed so assistive
// technologies announce whether Zen mode is on.
//
// [Ja] syncToggleButtons は現在の状態をすべてのトグルボタンの aria-pressed に反映し、支援技術が
// Zenモードの ON/OFF を読み上げられるようにする。
function syncToggleButtons(enabled: boolean): void {
  document
    .querySelectorAll("[data-zen-mode-toggle]")
    .forEach((button) => button.setAttribute("aria-pressed", String(enabled)));
}

// persistZenMode stores ON as "1" and turns OFF by deleting the cookie (max-age=0), matching how
// the server interprets the cookie's absence as OFF.
//
// [Ja] persistZenMode は ON を "1" として保存し、OFF はクッキー削除 (max-age=0) で表現する。
// サーバー側が「クッキーなし = OFF」と解釈するのに合わせている。
function persistZenMode(enabled: boolean): void {
  if (enabled) {
    document.cookie = `${ZEN_MODE_COOKIE_NAME}=1;path=/;max-age=${ZEN_MODE_COOKIE_MAX_AGE};SameSite=Lax`;
  } else {
    document.cookie = `${ZEN_MODE_COOKIE_NAME}=;path=/;max-age=0;SameSite=Lax`;
  }
}
