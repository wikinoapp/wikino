import { EditorView } from "codemirror";

export interface ListInfo {
  type: "unordered" | "ordered" | "task";
  indent: string;
  marker: string;
  content: string;
  number?: number;
  taskState?: "incomplete" | "complete";
}

const LIST_PATTERNS = {
  unordered: /^(\s*)([-*+])\s+(.*)$/,
  ordered: /^(\s*)(\d+)\.\s+(.*)$/,
  task: /^(\s*)([-*+])\s+\[([ xX])\]\s+(.*)$/,
};

export function detectListPattern(line: string): ListInfo | null {
  const taskMatch = line.match(LIST_PATTERNS.task);

  if (taskMatch) {
    const checkboxState = taskMatch[3];
    return {
      type: "task",
      indent: taskMatch[1],
      marker: taskMatch[2],
      content: taskMatch[4],
      taskState: checkboxState === " " ? "incomplete" : "complete",
    };
  }

  const unorderedMatch = line.match(LIST_PATTERNS.unordered);

  if (unorderedMatch) {
    return {
      type: "unordered",
      indent: unorderedMatch[1],
      marker: unorderedMatch[2],
      content: unorderedMatch[3],
    };
  }

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

export function generateContinuationText(listInfo: ListInfo | null): string {
  if (!listInfo) return "";

  if (listInfo.type === "task") {
    return `${listInfo.indent}${listInfo.marker} [ ] `;
  } else if (listInfo.type === "unordered") {
    return `${listInfo.indent}${listInfo.marker} `;
  } else if (listInfo.type === "ordered" && listInfo.number !== undefined) {
    return `${listInfo.indent}${listInfo.number + 1}. `;
  }

  return "";
}

export function insertNewlineAndContinueList(view: EditorView): boolean {
  const { state } = view;
  const { from, to } = state.selection.main;

  const line = state.doc.lineAt(from);
  const lineText = line.text;

  const listInfo = detectListPattern(lineText);

  if (!listInfo) {
    return false;
  }

  const cursorPositionInLine = from - line.from;

  const markerStartPosition = listInfo.indent.length;
  const markerEndPosition = markerStartPosition + listInfo.marker.length + 1;

  const listMarkerEndPosition = listInfo.type === "task" ? markerEndPosition + 4 : markerEndPosition;

  if (cursorPositionInLine < listMarkerEndPosition) {
    if (cursorPositionInLine === 0) {
      return false;
    }

    const beforeCursor = lineText.slice(0, cursorPositionInLine);
    const continuationText = generateContinuationText(listInfo);

    const contentToMove = listInfo.content;

    const transaction = state.update({
      changes: {
        from: line.from,
        to: line.to,
        insert: beforeCursor + "\n" + continuationText + contentToMove,
      },
      selection: {
        anchor: line.from + beforeCursor.length + 1 + listInfo.indent.length,
      },
    });

    view.dispatch(transaction);
    return true;
  }

  if (listInfo.content.trim() === "") {
    const indentLevel = listInfo.indent.length / 2;

    if (indentLevel > 0) {
      const newIndent = " ".repeat((indentLevel - 1) * 2);
      const newMarker = listInfo.type === "ordered" ? "1" : listInfo.marker;
      let newListText: string;

      if (listInfo.type === "task") {
        newListText = `${newIndent}${newMarker} [ ] `;
      } else if (listInfo.type === "ordered") {
        newListText = `${newIndent}${newMarker}. `;
      } else {
        newListText = `${newIndent}${newMarker} `;
      }

      const transaction = state.update({
        changes: {
          from: line.from,
          to: line.to,
          insert: newListText,
        },
        selection: { anchor: line.from + newListText.length },
      });

      view.dispatch(transaction);
    } else {
      const transaction = state.update({
        changes: {
          from: line.from,
          to: line.to,
          insert: "",
        },
        selection: { anchor: line.from },
      });

      view.dispatch(transaction);
    }

    return true;
  }

  const continuationText = generateContinuationText(listInfo);
  const insertText = `\n${continuationText}`;

  const transaction = state.update({
    changes: { from: to, insert: insertText },
    selection: { anchor: to + insertText.length },
  });

  view.dispatch(transaction);

  return true;
}
