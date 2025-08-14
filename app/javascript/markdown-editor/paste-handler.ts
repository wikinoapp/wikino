import { EditorView } from "@codemirror/view";

export function pasteHandler(view: EditorView, event: ClipboardEvent): boolean {
  const items = event.clipboardData?.items;

  if (!items) return false;

  const mediaItems: DataTransferItem[] = [];

  for (let i = 0; i < items.length; i++) {
    const item = items[i];

    // 画像と動画の両方をサポート
    if (item.type.startsWith("image/") || item.type.startsWith("video/")) {
      mediaItems.push(item);
    }
  }

  // メディアファイルが含まれていない場合は通常のペースト処理
  if (mediaItems.length === 0) return false;

  // メディアペースト時のブラウザのデフォルト動作を無効化する
  event.preventDefault();

  mediaItems.forEach((item) => {
    const file = item.getAsFile();

    if (file) {
      // メディアペーストイベントを発火（画像と動画の両方に対応）
      view.dom.dispatchEvent(
        new CustomEvent("media-paste", {
          detail: {
            file,
            position: view.state.selection.main.head,
          },
          bubbles: true,
        }),
      );
    }
  });

  return true;
}
