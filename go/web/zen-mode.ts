// ページエディタのZenモード切り替え。[data-zen-mode-toggle] のクリックで
// [data-zen-mode-container] 要素の "page-edit-zen" クラスを反転し (edit.templ内の子孫要素が
// Tailwindのin-[.page-edit-zen]: バリアントで反応し、左右カラムとリンク一覧の非表示・中央カラム
// の拡幅を行う)、状態をwikino_zen_modeクッキーへ保存して次回も同じモードで開けるようにする。
// 初期クラスはサーバーがクッキーから描画するため、本スクリプトは切り替えのみを担当する。
// イベント委譲により、後からスワップされた内容でも結線が機能する。

// クッキー名はinternal/handler/page/edit.goと、クラス名はedit.templと同期させること。
const ZEN_MODE_COOKIE_NAME = "wikino_zen_mode";
const ZEN_MODE_CLASS = "page-edit-zen";
const ZEN_MODE_COOKIE_MAX_AGE = 60 * 60 * 24 * 365; // 1年

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

// syncToggleButtonsは現在の状態をすべてのトグルボタンのaria-pressedに反映し、支援技術が
// ZenモードのON/OFFを読み上げられるようにする。
function syncToggleButtons(enabled: boolean): void {
  document
    .querySelectorAll("[data-zen-mode-toggle]")
    .forEach((button) => button.setAttribute("aria-pressed", String(enabled)));
}

// persistZenModeはONを "1" として保存し、OFFはクッキー削除 (max-age=0) で表現する。
// サーバー側が「クッキーなし = OFF」と解釈するのに合わせている。
function persistZenMode(enabled: boolean): void {
  if (enabled) {
    document.cookie = `${ZEN_MODE_COOKIE_NAME}=1;path=/;max-age=${ZEN_MODE_COOKIE_MAX_AGE};SameSite=Lax`;
  } else {
    document.cookie = `${ZEN_MODE_COOKIE_NAME}=;path=/;max-age=0;SameSite=Lax`;
  }
}
