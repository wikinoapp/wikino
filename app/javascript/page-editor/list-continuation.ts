// typed: strict
// frozen_string_literal: true

import { EditorView } from "codemirror";

/**
 * リスト記法の情報を表す型
 */
export interface ListInfo {
  type: "unordered" | "ordered";
  indent: string;
  marker: string;
  content: string;
  number?: number;
}

/**
 * リスト記法のパターンを検出する正規表現
 */
const LIST_PATTERNS = {
  // 順序なしリスト: '- ', '* ', '+ '
  unordered: /^(\s*)([-*+])\s+(.*)$/,
  // 順序付きリスト: '1. ', '2. ', etc.
  ordered: /^(\s*)(\d+)\.\s+(.*)$/,
};

/**
 * 現在行のリスト記法を検出する
 * @param line 検出対象の行テキスト
 * @returns リスト記法の情報、または null
 */
export function detectListPattern(line: string): ListInfo | null {
  // 順序なしリストの検出
  const unorderedMatch = line.match(LIST_PATTERNS.unordered);
  if (unorderedMatch) {
    return {
      type: "unordered",
      indent: unorderedMatch[1],
      marker: unorderedMatch[2],
      content: unorderedMatch[3],
    };
  }

  // 順序付きリストの検出
  const orderedMatch = line.match(LIST_PATTERNS.ordered);
  if (orderedMatch) {
    return {
      type: "ordered",
      indent: orderedMatch[1],
      marker: orderedMatch[2],
      content: orderedMatch[3],
      number: parseInt(orderedMatch[2], 10),
    };
  }

  return null;
}

/**
 * 継続すべきリスト記法のテキストを生成する
 * @param listInfo リスト記法の情報
 * @returns 継続するリスト記法のテキスト
 */
export function generateContinuationText(listInfo: ListInfo | null): string {
  if (!listInfo) return "";

  if (listInfo.type === "unordered") {
    return `${listInfo.indent}${listInfo.marker} `;
  } else if (listInfo.type === "ordered" && listInfo.number !== undefined) {
    return `${listInfo.indent}${listInfo.number + 1}. `;
  }

  return "";
}

/**
 * Enterキーが押されたときのリスト継続処理
 * @param view CodeMirrorのEditorView
 * @returns 処理が実行された場合はtrue、そうでなければfalse
 */
export function insertNewlineAndContinueList(view: EditorView): boolean {
  const { state } = view;
  const { from, to } = state.selection.main;
  
  // 現在の行を取得
  const line = state.doc.lineAt(from);
  const lineText = line.text;
  
  // リスト記法を検出
  const listInfo = detectListPattern(lineText);
  
  if (!listInfo) {
    // リスト記法でない場合は通常の改行
    return false;
  }

  // 空のリスト項目の場合（マーカーのみでコンテンツがない）
  if (listInfo.content.trim() === "") {
    // リスト記法を削除して通常の改行
    const transaction = state.update({
      changes: {
        from: line.from,
        to: line.to,
        insert: listInfo.indent,
      },
      selection: { anchor: line.from + listInfo.indent.length },
    });
    view.dispatch(transaction);
    return true;
  }

  // リスト記法を継続
  const continuationText = generateContinuationText(listInfo);
  const insertText = `\n${continuationText}`;
  
  const transaction = state.update({
    changes: { from: to, insert: insertText },
    selection: { anchor: to + insertText.length },
  });
  
  view.dispatch(transaction);
  return true;
}