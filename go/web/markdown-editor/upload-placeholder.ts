import type { EditorView } from "@codemirror/view";

interface UploadPlaceholder {
  id: string;
  fileName: string;
  position: number;
  length: number;
}

const placeholders = new Map<string, UploadPlaceholder>();

export function insertUploadPlaceholder(view: EditorView, fileName: string, position?: number): string {
  const id = `upload-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  const placeholderText = `<!-- Uploading "${fileName}"... -->`;

  const insertPos = position ?? view.state.selection.main.head;

  const transaction = view.state.update({
    changes: {
      from: insertPos,
      to: insertPos,
      insert: placeholderText,
    },
    selection: { anchor: insertPos + placeholderText.length },
  });

  view.dispatch(transaction);

  placeholders.set(id, {
    id,
    fileName,
    position: insertPos,
    length: placeholderText.length,
  });

  return id;
}

export function replacePlaceholderWithUrl(
  view: EditorView,
  placeholderId: string,
  url: string,
  altText?: string,
  width?: number,
  _height?: number,
  fileType?: string,
): boolean {
  const placeholder = placeholders.get(placeholderId);

  if (!placeholder) return false;

  const { position, length, fileName } = placeholder;
  let newText: string;

  if (fileType?.startsWith("image/")) {
    if (width) {
      newText = `<img width="${width}" alt="${altText || fileName}" src="${url}">`;
    } else {
      newText = `<img alt="${altText || fileName}" src="${url}">`;
    }
  } else {
    newText = `[${altText || fileName}](${url})`;
  }

  const currentText = view.state.doc.toString();
  const expectedText = `<!-- Uploading "${fileName}"... -->`;
  const actualText = currentText.slice(position, position + length);

  if (actualText !== expectedText) {
    const searchIndex = currentText.indexOf(expectedText);
    if (searchIndex === -1) {
      console.warn("Upload placeholder not found:", fileName);
      placeholders.delete(placeholderId);
      return false;
    }

    const transaction = view.state.update({
      changes: {
        from: searchIndex,
        to: searchIndex + expectedText.length,
        insert: newText,
      },
    });

    view.dispatch(transaction);
  } else {
    const transaction = view.state.update({
      changes: {
        from: position,
        to: position + length,
        insert: newText,
      },
    });

    view.dispatch(transaction);
  }

  placeholders.delete(placeholderId);

  return true;
}

export function removePlaceholder(view: EditorView, placeholderId: string): boolean {
  const placeholder = placeholders.get(placeholderId);
  if (!placeholder) return false;

  const { position, length, fileName } = placeholder;
  const expectedText = `<!-- Uploading "${fileName}"... -->`;
  const currentText = view.state.doc.toString();
  const actualText = currentText.slice(position, position + length);

  if (actualText === expectedText) {
    const transaction = view.state.update({
      changes: {
        from: position,
        to: position + length,
        insert: "",
      },
    });

    view.dispatch(transaction);
  } else {
    const searchIndex = currentText.indexOf(expectedText);
    if (searchIndex !== -1) {
      const transaction = view.state.update({
        changes: {
          from: searchIndex,
          to: searchIndex + expectedText.length,
          insert: "",
        },
      });

      view.dispatch(transaction);
    }
  }

  placeholders.delete(placeholderId);

  return true;
}
