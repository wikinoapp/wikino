import { EditorView } from "codemirror";

// edit.templ (saveDraftButtonID) が描画する手動保存 (「下書き保存」) ボタンのid。
// 両者を同期させること。
const SAVE_DRAFT_BUTTON_ID = "page-edit-save-draft-button";

// clickManualSaveButtonは「下書き保存」ボタンのクリックで手動保存を実行する。ボタンが
// 画面遷移なしのhtmx PATCHリクエストを送信する。実際にクリックしたかどうかを返す。
export function clickManualSaveButton(): boolean {
  const button = document.getElementById(SAVE_DRAFT_BUTTON_ID) as HTMLButtonElement | null;
  if (button && !button.disabled) {
    button.click();
    return true;
  }
  return false;
}

// handleManualSaveShortcutはMod-sキーマップ用に手動保存を実行する。常にtrueを返して
// CodeMirrorにキーイベントを消費させ、エディタ入力中にブラウザの「ページを保存」ダイアログが
// 出ないようにする (リクエスト中でボタンが一時的に無効な場合も含む)。
export function handleManualSaveShortcut(_view: EditorView): boolean {
  clickManualSaveButton();
  return true;
}
