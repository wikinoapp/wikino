import { EditorView } from "@codemirror/view";

export function pasteHandler(view: EditorView, event: ClipboardEvent): boolean {
  const items = event.clipboardData?.items;

  if (!items) return false;

  const fileItems: DataTransferItem[] = [];

  for (let i = 0; i < items.length; i++) {
    const item = items[i];

    // サポートするファイルタイプを判定
    if (isAcceptedFileType(item.type)) {
      fileItems.push(item);
    }
  }

  // 対象ファイルが含まれていない場合は通常のペースト処理
  if (fileItems.length === 0) return false;

  // ファイルペースト時のブラウザのデフォルト動作を無効化する
  event.preventDefault();

  fileItems.forEach((item) => {
    const file = item.getAsFile();

    if (file) {
      // ファイルタイプに応じてイベント名を決定
      const eventName = getEventNameForFileType(item.type);

      // ファイルペーストイベントを発火
      view.dom.dispatchEvent(
        new CustomEvent(eventName, {
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

// サポートするファイルタイプを判定
// NOTE: `.log` ファイルは `mimeType` が空文字列になることがあるためサポートしない
function isAcceptedFileType(mimeType: string): boolean {
  // 画像
  if (mimeType.startsWith("image/")) return true;

  // 動画
  if (mimeType.startsWith("video/")) return true;

  // PDF
  if (mimeType === "application/pdf") return true;

  // Microsoft Office文書
  if (mimeType.startsWith("application/vnd.ms-")) return true;
  if (mimeType.startsWith("application/vnd.openxmlformats-officedocument.")) return true;

  // テキストファイル
  if (mimeType.startsWith("text/")) return true;

  // JSONファイル
  if (mimeType === "application/json") return true;

  // テキストファイル
  if (mimeType === "text/plain") return true;

  // 圧縮ファイル
  const compressionTypes = [
    "application/zip",
    "application/x-zip-compressed",
    "application/x-rar-compressed",
    "application/gzip",
    "application/x-gzip",
    "application/x-tar",
    "application/x-7z-compressed",
  ];
  if (compressionTypes.includes(mimeType)) return true;

  return false;
}

// ファイルタイプに応じたイベント名を返す
function getEventNameForFileType(mimeType: string): string {
  // メディアファイル（画像・動画）は既存のイベント名を使用
  if (mimeType.startsWith("image/") || mimeType.startsWith("video/")) {
    return "media-paste";
  }

  // その他のファイルは汎用のファイルペーストイベント
  return "file-paste";
}
