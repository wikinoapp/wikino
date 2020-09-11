import axios from 'axios';
import {
  Completion,
  CompletionResult,
  CompletionContext,
  startCompletion,
  closeCompletion,
} from '@codemirror/next/autocomplete';
import { Extension } from '@codemirror/next/state';
import { Line, Text } from '@codemirror/next/text';
import { EditorView, KeyBinding } from '@codemirror/next/view';
import debounce from 'lodash/debounce';

import { EventDispatcher } from '../../utils/event-dispatcher';
import { reverseString } from '../../utils/string';

export function linkNote(): Extension {
  return EditorView.inputHandler.of(handleInput);
}

function handleInput(view: EditorView, from: number, to: number, insert: string) {
  if (insert !== '[') {
    return false;
  }
  console.log(`from: ${from}, to: ${to}, insert: ${insert}`);

  let line = view.state.doc.lineAt(from);
  console.log('line: ', line);

  return false;
}

function symbolValue(str: string, symbol: string) {
  const startCharMatches = reverseString(str).match(new RegExp(`^\\S+?\\${symbol}`));

  return startCharMatches ? reverseString(startCharMatches[0]).replace(new RegExp(`\\${symbol}`), '') : null;
}

function prevString() {}

export const completionKeymap: readonly KeyBinding[] = [
  { key: '[', run: startCompletion },
  { key: 'Escape', run: closeCompletion },
];

function linkKeyword(line: Line, pos: number) {
  const str = line.slice(0, pos + 1);
  // console.log(`linkKeyword > line.from: ${line.from}, pos: ${pos}, str: ${str}`);
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
  console.log(`line.from: ${line.from}, from: ${from}, chars: ${chars}`);
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

function applyCompletion(databaseId: string, title: string) {
  return (view: EditorView, completion: Completion, from: number, to: number) => {
    const { state } = view;
    const line = state.doc.lineAt(from);
    const pos = linkStartPos(line, from);
    console.log(`pos: ${pos}`);
    const text = `[${title}](/notes/${databaseId})`;
    view.dispatch(
      state.update({
        changes: { from: pos, to: from + 2, insert: text },
        selection: { anchor: pos + text.length },
      }),
    );
  };
}

export async function linkNoteCompletionSource(context: CompletionContext): Promise<CompletionResult | null> {
  const { state, pos } = context;
  const line = state.doc.lineAt(pos);
  const keyword = linkKeyword(line, pos);
  console.log(`state: ${state}, pos: ${pos}, keyword: ${keyword}`);

  if (!keyword) {
    return null;
  }

  const notes = await searchNotes(keyword);
  console.log('notes: ', notes);

  return {
    from: pos,
    to: pos,
    options: notes.map((note: any) => {
      return {
        label: note.title,
        apply: applyCompletion(note.databaseId, note.title),
      };
    }),
  };
}
