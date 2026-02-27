import { EditorView } from "@codemirror/view";

export function pasteHandler(view: EditorView, event: ClipboardEvent): boolean {
  const items = event.clipboardData?.items;

  if (!items) return false;

  const fileItems: DataTransferItem[] = [];

  for (let i = 0; i < items.length; i++) {
    const item = items[i];

    if (item.kind === "file" && isAcceptedFileType(item.type)) {
      fileItems.push(item);
    }
  }

  if (fileItems.length === 0) return false;

  event.preventDefault();

  fileItems.forEach((item) => {
    const file = item.getAsFile();

    if (file) {
      const eventName = getEventNameForFileType(item.type);

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

// NOTE: `.log` ファイルは `mimeType` が空文字列になることがあるためサポートしない
function isAcceptedFileType(mimeType: string): boolean {
  if (mimeType.startsWith("image/")) return true;

  if (mimeType.startsWith("video/")) return true;

  if (mimeType === "application/pdf") return true;

  if (mimeType.startsWith("application/vnd.ms-")) return true;
  if (mimeType.startsWith("application/vnd.openxmlformats-officedocument.")) return true;

  if (mimeType.startsWith("text/")) return true;

  if (mimeType === "application/json") return true;

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

function getEventNameForFileType(mimeType: string): string {
  if (mimeType.startsWith("image/") || mimeType.startsWith("video/")) {
    return "media-paste";
  }

  return "file-paste";
}
