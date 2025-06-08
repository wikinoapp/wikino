import { EditorView } from "codemirror";

/**
 * Cmd+Enter (Ctrl+Enter) キーが押されたときの処理
 * フォームを送信する
 * @param view CodeMirrorのEditorView
 * @returns 処理が実行された場合はtrue、そうでなければfalse
 */
export function handleSubmitShortcut(view: EditorView): boolean {
  // エディターの親要素からフォームを検索
  const editorElement = view.dom;
  const form = editorElement.closest("form");

  if (!form) {
    return false;
  }

  // メインの送信ボタンを検索してクリック
  const submitButton = form.querySelector('button[type="submit"]') as HTMLButtonElement;
  if (submitButton && !submitButton.disabled) {
    submitButton.click();
    return true;
  }

  // 送信ボタンが見つからないか無効な場合はfalseを返す
  return false;
}
