import { EditorView } from "codemirror";

// The id of the manual save ("save draft") button rendered by edit.templ (saveDraftButtonID).
// Keep the two in sync.
//
// [Ja] edit.templ (saveDraftButtonID) が描画する手動保存 (「下書き保存」) ボタンの id。
// 両者を同期させること。
const SAVE_DRAFT_BUTTON_ID = "page-edit-save-draft-button";

// clickManualSaveButton triggers a manual save by clicking the save-draft button, which sends
// the htmx PATCH request without navigating. Returns whether the button was actually clicked.
//
// [Ja] clickManualSaveButton は「下書き保存」ボタンのクリックで手動保存を実行する。ボタンが
// 画面遷移なしの htmx PATCH リクエストを送信する。実際にクリックしたかどうかを返す。
export function clickManualSaveButton(): boolean {
  const button = document.getElementById(SAVE_DRAFT_BUTTON_ID) as HTMLButtonElement | null;
  if (button && !button.disabled) {
    button.click();
    return true;
  }
  return false;
}

// handleManualSaveShortcut runs a manual save for the Mod-s keymap. It always returns true so
// CodeMirror consumes the key event and the browser's "save page" dialog never appears while
// typing in the editor (even when the button is temporarily disabled during a request).
//
// [Ja] handleManualSaveShortcut は Mod-s キーマップ用に手動保存を実行する。常に true を返して
// CodeMirror にキーイベントを消費させ、エディタ入力中にブラウザの「ページを保存」ダイアログが
// 出ないようにする (リクエスト中でボタンが一時的に無効な場合も含む)。
export function handleManualSaveShortcut(_view: EditorView): boolean {
  clickManualSaveButton();
  return true;
}
