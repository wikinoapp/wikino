import { EditorView } from "@codemirror/view";

// ペーストハンドラー
export function pasteHandler(view: EditorView, event: ClipboardEvent): boolean {
  const items = event.clipboardData?.items;
  if (!items) return false;

  // クリップボードアイテムを確認
  const imageItems: DataTransferItem[] = [];
  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    if (item.type.startsWith("image/")) {
      imageItems.push(item);
    }
  }

  // 画像が含まれていない場合は通常のペースト処理
  if (imageItems.length === 0) return false;

  // 画像のペーストを処理
  event.preventDefault();
  
  imageItems.forEach((item) => {
    const file = item.getAsFile();
    if (file) {
      // ペーストイベントを発火
      view.dom.dispatchEvent(
        new CustomEvent("image-paste", {
          detail: {
            file,
            position: view.state.selection.main.head
          },
          bubbles: true
        })
      );
    }
  });

  return true;
}