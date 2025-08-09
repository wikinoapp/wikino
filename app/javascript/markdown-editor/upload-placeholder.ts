import { EditorView } from "@codemirror/view";

interface UploadPlaceholder {
  id: string;
  fileName: string;
  position: number;
  length: number;
}

// アップロード中のプレースホルダーを管理
const placeholders = new Map<string, UploadPlaceholder>();

// アップロード中のプレースホルダーを挿入
export function insertUploadPlaceholder(view: EditorView, fileName: string, position?: number): string {
  const id = `upload-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  const placeholderText = `![アップロード中: ${fileName}]()`;

  const insertPos = position ?? view.state.selection.main.head;

  // テキストを挿入
  const transaction = view.state.update({
    changes: {
      from: insertPos,
      to: insertPos,
      insert: placeholderText,
    },
    selection: { anchor: insertPos + placeholderText.length },
  });

  view.dispatch(transaction);

  // プレースホルダー情報を保存
  placeholders.set(id, {
    id,
    fileName,
    position: insertPos,
    length: placeholderText.length,
  });

  return id;
}

// アップロード完了後、プレースホルダーをURLに置換
export function replacePlaceholderWithUrl(
  view: EditorView,
  placeholderId: string,
  url: string,
  altText?: string,
  width?: number,
  height?: number,
  isImage?: boolean,
): boolean {
  const placeholder = placeholders.get(placeholderId);
  if (!placeholder) return false;

  const { position, length, fileName } = placeholder;

  // 画像タグの生成
  let newText: string;
  if (isImage) {
    // 画像の場合は常に<img>タグを使用
    if (width && height) {
      newText = `<img width="${width}" height="${height}" alt="${altText || fileName}" src="${url}">`;
    } else {
      newText = `<img alt="${altText || fileName}" src="${url}">`;
    }
  } else {
    // 画像以外の場合は従来のMarkdown形式
    newText = `![${altText || fileName}](${url})`;
  }

  // 現在のドキュメントのテキストを取得して、プレースホルダーがまだ存在するか確認
  const currentText = view.state.doc.toString();
  const expectedText = `![アップロード中: ${fileName}]()`;
  const actualText = currentText.slice(position, position + length);

  // プレースホルダーが変更されていないか確認
  if (actualText !== expectedText) {
    // テキストが変更されている場合、検索して置換
    const searchIndex = currentText.indexOf(expectedText);
    if (searchIndex === -1) {
      console.warn("Upload placeholder not found:", fileName);
      placeholders.delete(placeholderId);
      return false;
    }

    // 新しい位置で置換
    const transaction = view.state.update({
      changes: {
        from: searchIndex,
        to: searchIndex + expectedText.length,
        insert: newText,
      },
    });

    view.dispatch(transaction);
  } else {
    // プレースホルダーを置換
    const transaction = view.state.update({
      changes: {
        from: position,
        to: position + length,
        insert: newText,
      },
    });

    view.dispatch(transaction);
  }

  // プレースホルダー情報を削除
  placeholders.delete(placeholderId);

  return true;
}

// アップロードエラー時の処理
export function removePlaceholder(view: EditorView, placeholderId: string): boolean {
  const placeholder = placeholders.get(placeholderId);
  if (!placeholder) return false;

  const { position, length, fileName } = placeholder;
  const expectedText = `![アップロード中: ${fileName}]()`;
  const currentText = view.state.doc.toString();
  const actualText = currentText.slice(position, position + length);

  if (actualText === expectedText) {
    // プレースホルダーを削除
    const transaction = view.state.update({
      changes: {
        from: position,
        to: position + length,
        insert: "",
      },
    });

    view.dispatch(transaction);
  } else {
    // テキストが変更されている場合、検索して削除
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
