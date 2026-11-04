import { DRAFT_SAVE_CONFLICT_EVENT } from "./draft-version";
import { overwriteConflictedDraft } from "./markdown-editor";
import { republishConflictedDraft } from "./publish-handler";

// 保存が競合した (編集画面を開いたあとに、下書きが別の場所で更新されていた) ことを知らせ、
// 利用者に「最新の内容を読み込む」か「このまま上書きする」かを選ばせる。
// edit.templの知らせ ([data-draft-conflict]) は隠して描画し、競合したときに表示する。
// 中のrole="alert" の要素が、表示されたときに読み上げられる。
// 公開が競合して再表示した編集画面では、知らせを表示した状態で描画し ([data-draft-conflict-publish])、
// 「このまま上書きする」は下書きの保存ではなく公開をし直す。

const CONFLICT_SELECTOR = "[data-draft-conflict]";
const RELOAD_BUTTON_SELECTOR = "[data-draft-conflict-reload]";
const OVERWRITE_BUTTON_SELECTOR = "[data-draft-conflict-overwrite]";
const PUBLISH_ATTRIBUTE = "data-draft-conflict-publish";

export function initializeDraftConflict(): void {
  document.addEventListener(DRAFT_SAVE_CONFLICT_EVENT, showConflict);
  document.addEventListener("click", handleClick);
  focusPublishConflict();
}

// focusPublishConflictは、公開が競合して再表示した編集画面で、表示した状態で描画した知らせへ
// フォーカスを移す。読み込んだときからあるrole="alert" の要素は読み上げられないことがあり、
// エディタの初期化が入力欄へ移したフォーカスのままでは、公開しなかったことに気付けないため。
function focusPublishConflict(): void {
  const conflict = document.querySelector<HTMLElement>(`${CONFLICT_SELECTOR}[${PUBLISH_ATTRIBUTE}]`);
  if (conflict && !conflict.hidden) conflict.focus();
}

function showConflict(): void {
  const conflict = document.querySelector<HTMLElement>(CONFLICT_SELECTOR);
  if (conflict) conflict.hidden = false;
}

function handleClick(event: MouseEvent): void {
  const target = event.target;
  if (!(target instanceof Element)) return;
  const conflict = target.closest<HTMLElement>(CONFLICT_SELECTOR);
  if (!conflict) return;

  if (target.closest(RELOAD_BUTTON_SELECTOR)) {
    if (!window.confirm(conflict.dataset.reloadConfirm ?? "")) return;
    reloadEditor(conflict);
    return;
  }

  if (target.closest(OVERWRITE_BUTTON_SELECTOR)) {
    conflict.hidden = true;
    if (conflict.hasAttribute(PUBLISH_ATTRIBUTE)) {
      republishConflictedDraft();
      return;
    }
    overwriteConflictedDraft();
  }
}

// reloadEditorは、編集画面をサーバーから取り直して最新の下書きを表示する。再読み込み (reload) では
// ブラウザがタイトルの入力欄に画面の値を戻すことがあるため、data-reload-urlにある編集画面のURLへ
// 移り直す。公開の競合後は表示中のURLが公開先になるため、移動先にはテンプレートが渡したURLを使う。
function reloadEditor(conflict: HTMLElement): void {
  const path = conflict.dataset.reloadUrl;
  if (!path) return;
  const url = new URL(path, window.location.href);
  window.location.replace(url.toString());
}
