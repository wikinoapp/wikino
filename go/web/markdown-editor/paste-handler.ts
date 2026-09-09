import { EditorView } from "@codemirror/view";
import { ALL_ALLOWED_TYPES } from "./file-upload-handler";

export function pasteHandler(view: EditorView, event: ClipboardEvent): boolean {
  const items = event.clipboardData?.items;

  if (!items) return false;

  const fileItems: DataTransferItem[] = [];

  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    // noUncheckedIndexedAccess widens indexed access to DataTransferItem |
    // undefined even within bounds, so guard before use.
    //
    // [Ja] noUncheckedIndexedAccess は範囲内でもインデックスアクセスを
    // DataTransferItem | undefined に広げるため、使用前にガードする。
    if (!item) continue;

    if (item.kind === "file" && ALL_ALLOWED_TYPES.includes(item.type)) {
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

function getEventNameForFileType(mimeType: string): string {
  if (mimeType.startsWith("image/") || mimeType.startsWith("video/")) {
    return "media-paste";
  }

  return "file-paste";
}
