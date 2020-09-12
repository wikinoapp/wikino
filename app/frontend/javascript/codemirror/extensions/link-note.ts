import axios from 'axios';
import { Completion, CompletionResult, CompletionContext } from '@codemirror/next/autocomplete';
import { Line } from '@codemirror/next/text';
import { EditorView } from '@codemirror/next/view';

import { reverseString } from '../../utils/string';
import { nonotoConfig } from '../../utils/nonoto-config';

function isEnclosedLinkSymbols(line: Line, pos: number) {
  return linkStartPos(line, pos) !== -1 && linkEndPos(line, pos) !== -1;
}

function linkKeyword(line: Line, pos: number) {
  if (!isEnclosedLinkSymbols(line, pos)) {
    return null;
  }

  const str = line.slice(0, pos + 1);
  const found = reverseString(str).match(/^].*\S\[\[/);

  return found ? reverseString(found[0].replace(/\[|]/g, '')) : '';
}

async function searchNotes(keyword: string) {
  try {
    const res = (await axios.get('/api/internal/notes', {
      params: {
        q: keyword,
      },
    })) as any;

    return res?.data?.notes || [];
  } catch (err) {
    console.log('err: ', err);
  }
}

function linkStartPos(line: Line, from: number) {
  const chars = reverseString(line.doc.slice(line.from, from).toString()).split('');
  let pos = from;
  let i = 0;

  while (i <= chars.length) {
    if (chars[i] === '[' && chars[i + 1] === '[') {
      return pos - 2;
    } else {
      i = i + 1;
      pos = pos - 1;
    }
  }

  return -1;
}

function linkEndPos(line: Line, from: number) {
  const chars = line.doc
    .slice(from - 1, line.to)
    .toString()
    .split('');
  let pos = from;
  let i = 0;

  while (i <= chars.length) {
    if (chars[i] === ']' && chars[i + 1] === ']') {
      return pos - 2;
    } else {
      i = i + 1;
      pos = pos - 1;
    }
  }

  return -1;
}

function applyCompletion(databaseId: string, title: string) {
  return (view: EditorView, completion: Completion, from: number, to: number) => {
    const { state } = view;
    const line = state.doc.lineAt(from);
    const pos = linkStartPos(line, from);
    const text = `[${title}](${nonotoConfig.nonotoUrl}/notes/${databaseId})`;

    view.dispatch(
      state.update({
        changes: { from: pos, to: from + 2, insert: text },
        selection: { anchor: pos + text.length },
      }),
    );
  };
}

function createNote(keyword: string) {
  return async (view: EditorView, completion: Completion, from: number, to: number) => {
    const { state } = view;
    const line = state.doc.lineAt(from);
    const pos = linkStartPos(line, from);

    try {
      const res = (await axios.post('/api/internal/notes', {
        keyword,
      })) as any;

      const { databaseId, title } = res.data;

      const text = `[${title}](${nonotoConfig.nonotoUrl}/notes/${databaseId})`;
      view.dispatch(
        state.update({
          changes: { from: pos, to: from + 2, insert: text },
          selection: { anchor: pos + text.length },
        }),
      );
    } catch (err) {
      console.log('err: ', err);
    }
  };
}

export async function linkNoteCompletionSource(context: CompletionContext): Promise<CompletionResult | null> {
  const { state, pos } = context;
  const line = state.doc.lineAt(pos);
  const keyword = linkKeyword(line, pos);

  if (!keyword) {
    return null;
  }

  const notes = await searchNotes(keyword);

  const options = notes.map((note: any) => {
    return {
      label: note.title,
      apply: applyCompletion(note.databaseId, note.title),
    };
  });
  options.push({
    label: nonotoConfig.i18n.messages.createNoteWithKeyword.replace('__keyword__', keyword),
    apply: await createNote(keyword),
  });

  return {
    from: pos,
    to: pos,
    options,
  };
}
