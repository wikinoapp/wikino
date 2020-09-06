import { deleteCharBackward, deleteCharForward } from '@codemirror/next/commands';
import { EditorSelection, EditorState, IndentContext, StateCommand, Transaction } from '@codemirror/next/state';
import { Line, Text } from '@codemirror/next/text';
import { KeyBinding } from '@codemirror/next/view';
import { NodeProp } from 'lezer-tree';

// export const withListComplement = (command: ({ state, dispatch }: any) => boolean): StateCommand => {
//   return ({ state, dispatch }): boolean => {
//     const changes = state.changeByRange(({ from, to }) => {
//       console.log(`from: ${from}, to: ${to}`);
//       let line = state.doc.lineAt(from);
//       console.log('line: ', line);
//
//       return {
//         changes: { from, to: to, insert: Text.of(['- ']) },
//         range: EditorSelection.cursor(from + 2),
//       };
//     });
//
//     const tr = state.update(changes, { scrollIntoView: true });
//     dispatch(tr);
//     command({ state: tr.state, dispatch });
//     return true;
//   };
// };

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
  // console.log('firstChar: ', firstChar);

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
  // console.log('lineString: ', lineString);
  return /^(\-|\*|(1\.))$/.test(lineString);
}

/// Replace the selection with a newline and indent the newly created
/// line(s). If the current line consists only of whitespace, this
/// will also delete that whitespace. When the cursor is between
/// matching brackets, an additional newline will be inserted after
/// the cursor.
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
    console.log('line: ', line);
    console.log(`from: ${from}, to: ${to}, indent: ${indent}, insert: ${insert}`);
    let lineStr = listString(line);
    if (isListBlank(line)) {
      insert.push('');
      return {
        changes: { from: line.from, to: to, insert: Text.of(insert) },
        range: EditorSelection.cursor(line.from + 1),
      };
    } else {
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
