import { EditorView } from "codemirror";
import { EditorSelection } from "@codemirror/state";

/**
 * インデントサイズ（半角スペース2つ）
 */
const INDENT_SIZE = "  ";

/**
 * Tabキーが押されたときの処理
 * @param view CodeMirrorのEditorView
 * @returns 処理が実行された場合はtrue、そうでなければfalse
 */
export function handleTab(view: EditorView): boolean {
  const { state } = view;
  const { ranges } = state.selection;

  // 複数行選択または単一行の処理
  const changes = ranges
    .map((range) => {
      const line = state.doc.lineAt(range.from);

      // 選択範囲がある場合は、選択されている全ての行にインデントを追加
      if (range.from !== range.to) {
        const startLine = state.doc.lineAt(range.from).number;
        let endLine = state.doc.lineAt(range.to).number;

        // 選択終了位置が行の先頭（改行文字の直後）にある場合、
        // その行は含めない（前の行の改行文字まで選択している状態）
        const endLineInfo = state.doc.line(endLine);
        if (range.to === endLineInfo.from && endLine > startLine) {
          endLine = endLine - 1;
        }

        const lineChanges = [];

        for (let i = startLine; i <= endLine; i++) {
          const line = state.doc.line(i);
          lineChanges.push({
            from: line.from,
            insert: INDENT_SIZE,
          });
        }

        return lineChanges;
      }

      // 選択範囲がない場合は、カーソル位置にインデントを挿入
      return {
        from: range.from,
        insert: INDENT_SIZE,
      };
    })
    .flat();

  // 変更を適用
  const transaction = state.update({
    changes,
    selection: EditorSelection.create(
      ranges.map((range, i) => {
        if (range.from === range.to) {
          // 単一カーソルの場合、インデント分だけカーソルを移動
          return EditorSelection.cursor(range.from + INDENT_SIZE.length);
        } else {
          // 選択範囲がある場合、選択を維持
          const startLine = state.doc.lineAt(range.from).number;
          let endLine = state.doc.lineAt(range.to).number;

          // 選択終了位置が行の先頭の場合、実際の処理対象行数を計算
          const endLineInfo = state.doc.line(endLine);
          if (range.to === endLineInfo.from && endLine > startLine) {
            endLine = endLine - 1;
          }

          const processedLines = endLine - startLine + 1;
          const addedLength = processedLines * INDENT_SIZE.length;

          // 選択終了位置を調整
          const originalEndLineInfo = state.doc.line(state.doc.lineAt(range.to).number);
          let newTo;
          if (range.to === originalEndLineInfo.from && state.doc.lineAt(range.to).number > startLine) {
            // 改行文字の直後で終わる選択の場合、処理対象行数分のインデントを加算
            newTo = range.to + processedLines * INDENT_SIZE.length;
          } else {
            // 通常の選択の場合
            newTo = range.to + processedLines * INDENT_SIZE.length;
          }

          // 選択開始位置は行の先頭に設定
          const startLineAfterIndent = state.doc.line(startLine);
          const newFrom = startLineAfterIndent.from;
          return EditorSelection.range(newFrom, newTo);
        }
      }),
      state.selection.mainIndex,
    ),
    scrollIntoView: true,
  });

  view.dispatch(transaction);
  return true;
}

/**
 * Shift+Tabキーが押されたときの処理（インデント削除）
 * @param view CodeMirrorのEditorView
 * @returns 処理が実行された場合はtrue、そうでなければfalse
 */
export function handleShiftTab(view: EditorView): boolean {
  const { state } = view;
  const { ranges } = state.selection;

  // 各選択範囲に対して処理
  const changes = ranges.flatMap((range) => {
    const startLine = state.doc.lineAt(range.from).number;
    let endLine = state.doc.lineAt(range.to).number;

    // 選択終了位置が行の先頭（改行文字の直後）にある場合、
    // その行は含めない（前の行の改行文字まで選択している状態）
    const endLineInfo = state.doc.line(endLine);
    if (range.to === endLineInfo.from && endLine > startLine) {
      endLine = endLine - 1;
    }

    const lineChanges = [];

    for (let i = startLine; i <= endLine; i++) {
      const line = state.doc.line(i);
      const lineText = line.text;

      // 行頭のインデントを検出
      if (lineText.startsWith(INDENT_SIZE)) {
        // 半角スペース2つを削除
        lineChanges.push({
          from: line.from,
          to: line.from + INDENT_SIZE.length,
          insert: "",
        });
      } else if (lineText.startsWith(" ")) {
        // 半角スペース1つしかない場合はそれを削除
        lineChanges.push({
          from: line.from,
          to: line.from + 1,
          insert: "",
        });
      }
    }

    return lineChanges;
  });

  if (changes.length === 0) {
    return false;
  }

  // 変更を適用
  const transaction = state.update({
    changes,
    scrollIntoView: true,
  });

  view.dispatch(transaction);
  return true;
}
