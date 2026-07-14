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
  // Capture groups from a successful match are always present, but
  // noUncheckedIndexedAccess widens them to string | undefined. Each branch
  // destructures the groups and bails out if a required one is missing.
  //
  // [Ja] マッチ成功時のキャプチャグループは常に存在するが、noUncheckedIndexedAccess
  // により string | undefined に広がる。各分岐はグループを分解し、必須のものが欠けて
  // いたら中断する。
  const taskMatch = line.match(LIST_PATTERNS.task);

  if (taskMatch) {
    const [, indent, marker, checkboxState, content] = taskMatch;
    if (indent === undefined || marker === undefined || checkboxState === undefined || content === undefined) {
      return null;
    }
    return {
      type: "task",
      indent,
      marker,
      content,
      taskState: checkboxState === " " ? "incomplete" : "complete",
    };
  }

  const unorderedMatch = line.match(LIST_PATTERNS.unordered);

  if (unorderedMatch) {
    const [, indent, marker, content] = unorderedMatch;
    if (indent === undefined || marker === undefined || content === undefined) {
      return null;
    }
    return {
      type: "unordered",
      indent,
      marker,
      content,
    };
  }

  const orderedMatch = line.match(LIST_PATTERNS.ordered);

  if (orderedMatch) {
    const [, indent, marker, content] = orderedMatch;
    if (indent === undefined || marker === undefined || content === undefined) {
      return null;
    }
    return {
      type: "ordered",
      indent,
      marker,
      content,
      number: parseInt(marker, 10),
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
