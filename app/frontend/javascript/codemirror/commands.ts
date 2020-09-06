import { deleteCharBackward, deleteCharForward } from '@codemirror/next/commands';
import { EditorSelection, EditorState, IndentContext, StateCommand } from '@codemirror/next/state';
import { Line, Text } from '@codemirror/next/text';
import { KeyBinding } from '@codemirror/next/view';
import { NodeProp } from 'lezer-tree';

// Original:
// https://github.com/codemirror/codemirror.next/blob/469db41099a5dff31aca59e79dfb902b111a1acc/commands/src/commands.ts#L481-L489
function isBetweenBrackets(state: EditorState, pos: number): { from: number; to: number } | null {
  if (/\(\)|\[\]|\{\}/.test(state.sliceDoc(pos - 1, pos + 1))) return { from: pos, to: pos };
  let context = state.tree.resolve(pos);
  let before = context.childBefore(pos),
    after = context.childAfter(pos),
    closedBy;
  if (
    before &&
    after &&
    before.end <= pos &&
    after.start >= pos &&
    (closedBy = before.type.prop(NodeProp.closedBy)) &&
    closedBy.indexOf(after.name) > -1
  )
    return { from: before.end, to: after.start };
  return null;
}

// Original:
// https://github.com/codemirror/codemirror.next/blob/469db41099a5dff31aca59e79dfb902b111a1acc/commands/src/commands.ts#L467-L473
function getIndentation(cx: IndentContext, pos: number): number {
  for (let f of cx.state.facet(EditorState.indentation)) {
    let result = f(cx, pos);
    if (result > -1) return result;
  }
  return -1;
}

function listString(line: Line) {
  const lineString = line.doc.sliceString(line.from, line.to).trim();
  const firstChar = lineString.charAt(0);

  if (firstChar === '-' || firstChar === '*') {
    return `${firstChar} `;
  }

  if (lineString.slice(0, 2) === '1.') {
    return '1. ';
  }

  return '';
}

function isListBlank(line: Line) {
  const lineString = line.doc.sliceString(line.from, line.to).trim();

  return /^(\-|\*|(1\.))$/.test(lineString);
}

// Original:
// https://github.com/codemirror/codemirror.next/blob/469db41099a5dff31aca59e79dfb902b111a1acc/commands/src/commands.ts#L496-L514
const insertNewlineAndIndent: StateCommand = ({ state, dispatch }): boolean => {
  let changes = state.changeByRange(({ from, to }) => {
    let cx = new IndentContext(state, { simulateBreak: from });
    let indent = getIndentation(cx, from);
    if (indent < 0) indent = /^\s*/.exec(state.doc.lineAt(from).slice(0, 50))![0].length;

    let line = state.doc.lineAt(from);
    while (to < line.to && /\s/.test(line.slice(to - line.from, to + 1 - line.from))) to++;
    if (from > line.from && from < line.from + 100 && !/\S/.test(line.slice(0, from))) {
      from = line.from;
    }

    let insert = [''];

    if (isListBlank(line)) {
      insert.push('');

      return {
        changes: { from: line.from, to: to, insert: Text.of(insert) },
        range: EditorSelection.cursor(line.from + 1),
      };
    } else {
      const lineStr = listString(line);
      insert.push(`${state.indentString(indent)}${lineStr}`);

      return {
        changes: { from, to, insert: Text.of(insert) },
        range: EditorSelection.cursor(from + 1 + indent + lineStr.length),
      };
    }
  });

  dispatch(state.update(changes, { scrollIntoView: true }));

  return true;
};

export const nonotoKeymap: readonly KeyBinding[] = [
  { key: 'Enter', run: insertNewlineAndIndent },
  { key: 'Backspace', run: deleteCharBackward },
  { key: 'Delete', run: deleteCharForward },
];
