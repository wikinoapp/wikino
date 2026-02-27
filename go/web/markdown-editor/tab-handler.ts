import { EditorView } from "codemirror";
import { EditorSelection, SelectionRange } from "@codemirror/state";
import { detectListPattern } from "./list-continuation";

const INDENT_SIZE = "  ";

type ChangeSpec = {
  from: number;
  to?: number;
  insert: string;
};

export function handleTab(view: EditorView): boolean {
  const { state } = view;
  const { ranges } = state.selection;

  const changes = ranges
    .map((range: SelectionRange) => {
      if (range.from !== range.to) {
        const startLine = state.doc.lineAt(range.from).number;
        let endLine = state.doc.lineAt(range.to).number;

        const endLineInfo = state.doc.line(endLine);
        if (range.to === endLineInfo.from && endLine > startLine) {
          endLine = endLine - 1;
        }

        const lineChanges: ChangeSpec[] = [];

        for (let i = startLine; i <= endLine; i++) {
          const line = state.doc.line(i);
          lineChanges.push({
            from: line.from,
            insert: INDENT_SIZE,
          });
        }

        return lineChanges;
      }

      const line = state.doc.lineAt(range.from);
      const lineText = line.text;
      const cursorPositionInLine = range.from - line.from;

      const listInfo = detectListPattern(lineText);

      if (listInfo) {
        let markerEndPosition: number;
        if (listInfo.type === "task") {
          markerEndPosition =
            listInfo.indent.length + listInfo.marker.length + 5;
        } else {
          markerEndPosition =
            listInfo.indent.length + listInfo.marker.length + 1;
        }

        if (cursorPositionInLine === markerEndPosition) {
          const originalMarker = lineText.substring(
            listInfo.indent.length,
            markerEndPosition,
          );

          if (listInfo.content === "") {
            return {
              from: line.from,
              to: line.from + markerEndPosition,
              insert: listInfo.indent + INDENT_SIZE + originalMarker,
            };
          } else {
            const content = lineText.substring(markerEndPosition);
            return {
              from: line.from,
              to: line.to,
              insert:
                listInfo.indent + INDENT_SIZE + originalMarker + content,
            };
          }
        }
      }

      return {
        from: range.from,
        insert: INDENT_SIZE,
      };
    })
    .flat();

  const transaction = state.update({
    changes,
    selection: EditorSelection.create(
      ranges.map((range: SelectionRange, index: number) => {
        if (range.from === range.to) {
          const change = changes[index] as ChangeSpec;

          if (change.to !== undefined && change.to > change.from) {
            const line = state.doc.lineAt(change.from);
            const lineText = line.text;
            const listInfo = detectListPattern(lineText);

            if (listInfo && listInfo.content !== "") {
              let newMarkerLength: number;
              if (listInfo.type === "task") {
                newMarkerLength =
                  INDENT_SIZE.length + listInfo.marker.length + 5;
              } else {
                newMarkerLength =
                  INDENT_SIZE.length + listInfo.marker.length + 1;
              }
              return EditorSelection.cursor(
                change.from + listInfo.indent.length + newMarkerLength,
              );
            }

            return EditorSelection.cursor(
              change.from + change.insert.length,
            );
          }

          if (change.insert.startsWith("\n")) {
            return EditorSelection.cursor(
              change.from + change.insert.length,
            );
          }

          return EditorSelection.cursor(range.from + INDENT_SIZE.length);
        } else {
          const startLine = state.doc.lineAt(range.from).number;
          let endLine = state.doc.lineAt(range.to).number;

          const endLineInfo = state.doc.line(endLine);
          if (range.to === endLineInfo.from && endLine > startLine) {
            endLine = endLine - 1;
          }

          const processedLines = endLine - startLine + 1;

          const originalEndLineInfo = state.doc.line(
            state.doc.lineAt(range.to).number,
          );
          let newTo;
          if (
            range.to === originalEndLineInfo.from &&
            state.doc.lineAt(range.to).number > startLine
          ) {
            newTo = range.to + processedLines * INDENT_SIZE.length;
          } else {
            newTo = range.to + processedLines * INDENT_SIZE.length;
          }

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

export function handleShiftTab(view: EditorView): boolean {
  const { state } = view;
  const { ranges } = state.selection;

  const changes = ranges.flatMap((range: SelectionRange) => {
    const startLine = state.doc.lineAt(range.from).number;
    let endLine = state.doc.lineAt(range.to).number;

    const endLineInfo = state.doc.line(endLine);
    if (range.to === endLineInfo.from && endLine > startLine) {
      endLine = endLine - 1;
    }

    const lineChanges: ChangeSpec[] = [];

    for (let i = startLine; i <= endLine; i++) {
      const line = state.doc.line(i);
      const lineText = line.text;

      if (lineText.startsWith(INDENT_SIZE)) {
        lineChanges.push({
          from: line.from,
          to: line.from + INDENT_SIZE.length,
          insert: "",
        });
      } else if (lineText.startsWith(" ")) {
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

  const transaction = state.update({
    changes,
    scrollIntoView: true,
  });

  view.dispatch(transaction);
  return true;
}
